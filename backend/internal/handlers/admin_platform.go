package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/evocrm/backend/internal/middleware"
	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func GetPlatformHealth(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		rows, err := svc.DB.Query(`
			SELECT co.id, co.name, co.is_active, co.deletion_scheduled_at,
			       COUNT(DISTINCT u.id) FILTER (WHERE u.is_online) AS online_users,
			       COUNT(DISTINCT wi.id) AS whatsapp_total,
			       COUNT(DISTINCT wi.id) FILTER (WHERE wi.status = 'connected') AS whatsapp_connected,
			       COUNT(DISTINCT wi.id) FILTER (WHERE wi.status IN ('error', 'disconnected')) AS whatsapp_issues,
			       MAX(GREATEST(COALESCE(cv.last_message_at, cv.created_at), COALESCE(wi.updated_at, wi.created_at))) AS last_activity
			FROM companies co
			LEFT JOIN users u ON u.company_id = co.id AND u.is_active
			LEFT JOIN whatsapp_instances wi ON wi.company_id = co.id
			LEFT JOIN conversations cv ON cv.company_id = co.id
			GROUP BY co.id, co.name, co.is_active, co.deletion_scheduled_at
			ORDER BY whatsapp_issues DESC, co.name
		`)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()
		items := []fiber.Map{}
		for rows.Next() {
			var id, name string
			var active bool
			var deletion, lastActivity sql.NullTime
			var online, total, connected, issues int
			if err := rows.Scan(&id, &name, &active, &deletion, &online, &total, &connected, &issues, &lastActivity); err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
			status := "healthy"
			if !active || deletion.Valid {
				status = "suspended"
			} else if issues > 0 {
				status = "attention"
			}
			items = append(items, fiber.Map{"id": id, "name": name, "status": status, "is_active": active, "online_users": online, "whatsapp_total": total, "whatsapp_connected": connected, "whatsapp_issues": issues, "last_activity": nullableTimeValue(lastActivity), "deletion_scheduled_at": nullableTimeValue(deletion)})
		}
		return c.JSON(fiber.Map{"tenants": items})
	}
}

func GetGlobalAuditLogs(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 150)
		if limit < 1 || limit > 500 {
			limit = 150
		}
		search := "%" + strings.ToLower(strings.TrimSpace(c.Query("search"))) + "%"
		rows, err := svc.DB.Query(`
			SELECT al.id, al.action, COALESCE(al.entity_type, al.resource_type, ''),
			       COALESCE(al.entity_id, al.resource_id), COALESCE(co.name, ''),
			       COALESCE(u.name, 'Sistema'), COALESCE(u.email, ''),
			       COALESCE(al.ip_address, ''), COALESCE(al.metadata, al.details, '{}'::jsonb), al.created_at
			FROM audit_logs al
			LEFT JOIN companies co ON co.id = al.company_id
			LEFT JOIN users u ON u.id = al.user_id
			WHERE $1 = '%%' OR LOWER(al.action) LIKE $1 OR LOWER(COALESCE(co.name, '')) LIKE $1 OR LOWER(COALESCE(u.email, '')) LIKE $1
			ORDER BY al.created_at DESC LIMIT $2
		`, search, limit)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()
		logs := []fiber.Map{}
		for rows.Next() {
			var id, action, entityType, tenantName, userName, email, ip string
			var entityID sql.NullString
			var metadata []byte
			var created time.Time
			if err := rows.Scan(&id, &action, &entityType, &entityID, &tenantName, &userName, &email, &ip, &metadata, &created); err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
			var parsed interface{}
			_ = json.Unmarshal(metadata, &parsed)
			logs = append(logs, fiber.Map{"id": id, "action": action, "entity_type": entityType, "entity_id": nullableStringMapValue(entityID), "tenant_name": tenantName, "user_name": userName, "user_email": email, "ip_address": ip, "metadata": parsed, "created_at": created})
		}
		return c.JSON(fiber.Map{"logs": logs})
	}
}

func GetPlatformIncidents(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		rows, err := svc.DB.Query(`
			SELECT al.id, al.action, COALESCE(co.name, ''), COALESCE(al.metadata, al.details, '{}'::jsonb), al.created_at
			FROM audit_logs al
			LEFT JOIN companies co ON co.id = al.company_id
			WHERE al.action ILIKE '%down%' OR al.action ILIKE '%fail%' OR al.action ILIKE '%error%'
			ORDER BY al.created_at DESC LIMIT 200
		`)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()
		incidents := []fiber.Map{}
		for rows.Next() {
			var id, action, tenant string
			var metadata []byte
			var created time.Time
			if err := rows.Scan(&id, &action, &tenant, &metadata, &created); err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
			var parsed interface{}
			_ = json.Unmarshal(metadata, &parsed)
			incidents = append(incidents, fiber.Map{"id": id, "action": action, "tenant_name": tenant, "metadata": parsed, "created_at": created})
		}
		return c.JSON(fiber.Map{"incidents": incidents})
	}
}

func RetryPlatformIncident(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var instanceName string
		err := svc.DB.QueryRow(`
			SELECT COALESCE(metadata->>'instance_name', details->>'instance_name', '')
			FROM audit_logs WHERE id = $1
		`, c.Params("id")).Scan(&instanceName)
		if err != nil || instanceName == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Este incidente não possui recuperação automática"})
		}
		status, retryErr := svc.Evolution.GetConnectionStatus(instanceName)
		metadata := fiber.Map{"incident_id": c.Params("id"), "instance_name": instanceName, "status": status}
		if retryErr != nil {
			metadata["error"] = retryErr.Error()
			logAuditEvent(svc.DB, c, "admin.incident.retry.failed", "audit_log", c.Params("id"), metadata)
			return c.Status(502).JSON(fiber.Map{"error": retryErr.Error(), "status": status})
		}
		logAuditEvent(svc.DB, c, "admin.incident.retry", "audit_log", c.Params("id"), metadata)
		return c.JSON(fiber.Map{"message": "Verificação executada", "status": status})
	}
}

func GetAdminSessions(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		rows, err := svc.DB.Query(`
			SELECT rt.id, u.name, u.email, co.name, COALESCE(rt.ip_address, ''), COALESCE(rt.user_agent, ''),
			       COALESCE(rt.last_used_at, rt.created_at), rt.expires_at
			FROM refresh_tokens rt
			JOIN users u ON u.id = rt.user_id
			JOIN companies co ON co.id = u.company_id
			WHERE rt.expires_at > NOW()
			ORDER BY COALESCE(rt.last_used_at, rt.created_at) DESC LIMIT 300
		`)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()
		sessions := []fiber.Map{}
		for rows.Next() {
			var id, name, email, tenant, ip, agent string
			var lastUsed, expires time.Time
			if err := rows.Scan(&id, &name, &email, &tenant, &ip, &agent, &lastUsed, &expires); err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
			sessions = append(sessions, fiber.Map{"id": id, "user_name": name, "email": email, "tenant_name": tenant, "ip_address": ip, "user_agent": agent, "last_used_at": lastUsed, "expires_at": expires})
		}
		return c.JSON(fiber.Map{"sessions": sessions})
	}
}

func RevokeAdminSession(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		result, err := svc.DB.Exec("DELETE FROM refresh_tokens WHERE id = $1", c.Params("id"))
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return c.Status(404).JSON(fiber.Map{"error": "Sessão não encontrada"})
		}
		logAuditEvent(svc.DB, c, "admin.session.revoke", "refresh_token", "", fiber.Map{"session_id": c.Params("id")})
		return c.JSON(fiber.Map{"message": "Sessão revogada"})
	}
}

func GetAdminSecuritySummary(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var failedLogins, superAdmins, without2FA, activeSessions int
		_ = svc.DB.QueryRow("SELECT COUNT(*) FROM login_events WHERE success = false AND created_at >= NOW() - INTERVAL '24 hours'").Scan(&failedLogins)
		_ = svc.DB.QueryRow("SELECT COUNT(*) FROM users WHERE is_super_admin = true").Scan(&superAdmins)
		_ = svc.DB.QueryRow("SELECT COUNT(*) FROM users WHERE is_super_admin = true AND COALESCE(two_factor_enabled, false) = false").Scan(&without2FA)
		_ = svc.DB.QueryRow("SELECT COUNT(*) FROM refresh_tokens WHERE expires_at > NOW()").Scan(&activeSessions)
		return c.JSON(fiber.Map{"failed_logins_24h": failedLogins, "super_admins": superAdmins, "super_admins_without_2fa": without2FA, "active_sessions": activeSessions})
	}
}

func ImpersonateTenant(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := c.Params("id")
		var body struct {
			Reason string `json:"reason"`
		}
		_ = c.BodyParser(&body)
		if len(strings.TrimSpace(body.Reason)) < 5 {
			return c.Status(400).JSON(fiber.Map{"error": "Informe uma justificativa para o acesso"})
		}
		var id, companyID, name, email, roleSlug, roleName string
		var isOnline, isSuper bool
		err := svc.DB.QueryRow(`
			SELECT u.id, u.company_id, u.name, u.email, COALESCE(r.slug, 'admin'), COALESCE(r.name, 'Administrador'),
			       COALESCE(u.is_online, false), COALESCE(u.is_super_admin, false)
			FROM users u LEFT JOIN roles r ON r.id = u.role_id
			WHERE u.company_id = $1 AND u.is_active = true AND COALESCE(u.is_super_admin, false) = false
			ORDER BY CASE WHEN r.slug = 'admin' THEN 0 ELSE 1 END, u.created_at LIMIT 1
		`, tenantID).Scan(&id, &companyID, &name, &email, &roleSlug, &roleName, &isOnline, &isSuper)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Nenhum usuário ativo encontrado no tenant"})
		}
		originalAccess := c.Cookies("crm_access")
		if originalAccess == "" {
			return c.Status(401).JSON(fiber.Map{"error": "Sessão administrativa inválida"})
		}
		sessionID := uuid.New().String()
		access, refresh, err := middleware.GenerateTokensForSession(id, companyID, roleSlug, email, sessionID, svc.Config)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		_, err = svc.DB.Exec(`
			INSERT INTO refresh_tokens (id, user_id, token, expires_at, session_id, device_name, ip_address, user_agent)
			VALUES ($1, $2, $3, $4, $5, 'Impersonação administrativa', $6, $7)
		`, uuid.New().String(), id, hashAdminToken(refresh), time.Now().Add(svc.Config.JWTRefreshExpiry),
			sessionID, c.IP(), c.Get("User-Agent"))
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Não foi possível criar a sessão de suporte"})
		}
		secure := strings.EqualFold(svc.Config.AppEnv, "production")
		c.Cookie(&fiber.Cookie{
			Name: "crm_impersonator", Value: originalAccess, Path: "/api/auth/impersonation",
			HTTPOnly: true, Secure: secure, SameSite: fiber.CookieSameSiteLaxMode,
			MaxAge: int((2 * time.Hour).Seconds()),
		})
		c.Cookie(&fiber.Cookie{
			Name: "crm_access", Value: access, Path: "/", HTTPOnly: true,
			Secure: secure, SameSite: fiber.CookieSameSiteLaxMode,
			MaxAge: int((2 * time.Hour).Seconds()),
		})
		logAuditEvent(svc.DB, c, "admin.tenant.impersonate", "company", tenantID, fiber.Map{"reason": body.Reason, "target_user_id": id, "target_email": email})
		return c.JSON(fiber.Map{"impersonating": true, "user": fiber.Map{"id": id, "company_id": companyID, "name": name, "email": email, "role_slug": roleSlug, "role_name": roleName, "is_online": isOnline, "is_super_admin": false}})
	}
}

func hashAdminToken(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func ExportTenantData(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := c.Params("id")
		var name, slug string
		if err := svc.DB.QueryRow("SELECT name, slug FROM companies WHERE id = $1", tenantID).Scan(&name, &slug); err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Tenant não encontrado"})
		}
		queries := map[string]string{
			"users":         "SELECT id, name, email, phone, role_id, is_active, created_at, updated_at FROM users WHERE company_id = $1 ORDER BY created_at",
			"contacts":      "SELECT * FROM contacts WHERE company_id = $1 ORDER BY created_at",
			"channels":      "SELECT id, name, type, status, team_id, business_hours, is_active, created_at, updated_at FROM channels WHERE company_id = $1 ORDER BY created_at",
			"conversations": "SELECT * FROM conversations WHERE company_id = $1 ORDER BY created_at",
			"messages":      "SELECT * FROM messages WHERE company_id = $1 ORDER BY created_at",
			"audit_logs":    "SELECT * FROM audit_logs WHERE company_id = $1 ORDER BY created_at",
		}
		data := fiber.Map{}
		counts := fiber.Map{}
		for key, query := range queries {
			items, err := exportTenantRows(svc.DB, query, tenantID)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "Falha ao exportar " + key + ": " + err.Error()})
			}
			data[key] = items
			counts[key] = len(items)
		}
		logAuditEvent(svc.DB, c, "admin.tenant.export", "company", tenantID, nil)
		c.Set("Content-Disposition", "attachment; filename=tenant-"+slug+".json")
		return c.JSON(fiber.Map{"tenant": fiber.Map{"id": tenantID, "name": name, "slug": slug}, "counts": counts, "data": data, "exported_at": time.Now()})
	}
}

func exportTenantRows(db *sql.DB, query, tenantID string) ([]interface{}, error) {
	rows, err := db.Query("SELECT to_jsonb(export_row) FROM ("+query+") export_row", tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []interface{}{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var item interface{}
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func ScheduleTenantDeletion(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := c.Params("id")
		var body struct {
			Reason string `json:"reason"`
		}
		_ = c.BodyParser(&body)
		if len(strings.TrimSpace(body.Reason)) < 5 {
			return c.Status(400).JSON(fiber.Map{"error": "Informe o motivo da exclusão"})
		}
		result, err := svc.DB.Exec("UPDATE companies SET is_active = false, suspended_at = NOW(), deletion_scheduled_at = NOW() + INTERVAL '7 days', updated_at = NOW() WHERE id = $1", tenantID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return c.Status(404).JSON(fiber.Map{"error": "Tenant não encontrado"})
		}
		_, _ = svc.DB.Exec("DELETE FROM refresh_tokens WHERE user_id IN (SELECT id FROM users WHERE company_id = $1)", tenantID)
		logAuditEvent(svc.DB, c, "admin.tenant.deletion.schedule", "company", tenantID, fiber.Map{"reason": body.Reason, "recovery_days": 7})
		return c.JSON(fiber.Map{"message": "Exclusão agendada", "recovery_days": 7})
	}
}

func RestoreTenant(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		result, err := svc.DB.Exec("UPDATE companies SET is_active = true, suspended_at = NULL, deletion_scheduled_at = NULL, updated_at = NOW() WHERE id = $1", c.Params("id"))
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return c.Status(404).JSON(fiber.Map{"error": "Tenant não encontrado"})
		}
		logAuditEvent(svc.DB, c, "admin.tenant.restore", "company", c.Params("id"), nil)
		return c.JSON(fiber.Map{"message": "Tenant restaurado"})
	}
}
