package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

var validConsentChannels = map[string]bool{"whatsapp": true, "email": true, "marketing": true}
var validConsentStatuses = map[string]bool{"granted": true, "denied": true, "revoked": true, "opted_out": true, "unknown": true}
var validRequestTypes = map[string]bool{"access": true, "correction": true, "deletion": true}
var validRequestStatuses = map[string]bool{"open": true, "in_progress": true, "completed": true, "rejected": true}

func GetContactChannelConsents(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		contactID := c.Params("id")
		rows, err := svc.DB.Query(`
			SELECT channel, purpose, status, COALESCE(source, ''), COALESCE(consent_text, ''),
			       proof, given_at, revoked_at, updated_at
			FROM contact_channel_consents
			WHERE company_id = $1 AND contact_id = $2
			ORDER BY channel, purpose
		`, companyID, contactID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erro ao consultar consentimentos"})
		}
		defer rows.Close()
		items := []fiber.Map{}
		for rows.Next() {
			var channel, purpose, status, source, text string
			var proof json.RawMessage
			var givenAt, revokedAt sql.NullTime
			var updatedAt time.Time
			if err := rows.Scan(&channel, &purpose, &status, &source, &text, &proof, &givenAt, &revokedAt, &updatedAt); err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "Erro ao ler consentimentos"})
			}
			items = append(items, fiber.Map{
				"channel": channel, "purpose": purpose, "status": status, "source": source,
				"consent_text": text, "proof": proof, "given_at": nullableTimeValue(givenAt),
				"revoked_at": nullableTimeValue(revokedAt), "updated_at": updatedAt,
			})
		}
		return c.JSON(fiber.Map{"consents": items})
	}
}

func UpsertContactChannelConsent(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		userID := c.Locals("user_id").(string)
		contactID := c.Params("id")
		var body struct {
			Channel     string                 `json:"channel"`
			Purpose     string                 `json:"purpose"`
			Status      string                 `json:"status"`
			Source      string                 `json:"source"`
			ConsentText string                 `json:"consent_text"`
			Proof       map[string]interface{} `json:"proof"`
		}
		if c.BodyParser(&body) != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Dados inválidos"})
		}
		body.Channel = strings.ToLower(strings.TrimSpace(body.Channel))
		body.Status = strings.ToLower(strings.TrimSpace(body.Status))
		body.Purpose = strings.ToLower(strings.TrimSpace(body.Purpose))
		if body.Purpose == "" {
			body.Purpose = "marketing"
		}
		if !validConsentChannels[body.Channel] || !validConsentStatuses[body.Status] {
			return c.Status(400).JSON(fiber.Map{"error": "Canal ou status de consentimento inválido"})
		}
		proof, _ := json.Marshal(body.Proof)
		result, err := svc.DB.Exec(`
			INSERT INTO contact_channel_consents
				(id, company_id, contact_id, channel, purpose, status, source, consent_text,
				 proof, given_at, revoked_at, updated_by)
			SELECT $1, $2, id, $4, $5, $6, NULLIF($7, ''), NULLIF($8, ''), $9::jsonb,
			       CASE WHEN $6 = 'granted' THEN NOW() END,
			       CASE WHEN $6 IN ('revoked', 'denied', 'opted_out') THEN NOW() END,
			       NULLIF($10, '')::uuid
			FROM contacts WHERE id = $3 AND company_id = $2
			ON CONFLICT (contact_id, channel, purpose) DO UPDATE SET
				status = EXCLUDED.status, source = EXCLUDED.source,
				consent_text = EXCLUDED.consent_text, proof = EXCLUDED.proof,
				given_at = CASE WHEN EXCLUDED.status = 'granted' THEN NOW() ELSE contact_channel_consents.given_at END,
				revoked_at = CASE WHEN EXCLUDED.status IN ('revoked', 'denied', 'opted_out') THEN NOW()
					WHEN EXCLUDED.status = 'granted' THEN NULL ELSE contact_channel_consents.revoked_at END,
				updated_by = EXCLUDED.updated_by, updated_at = NOW()
		`, uuid.New().String(), companyID, contactID, body.Channel, body.Purpose, body.Status,
			body.Source, body.ConsentText, proof, userID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erro ao salvar consentimento"})
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return c.Status(404).JSON(fiber.Map{"error": "Contato não encontrado"})
		}
		logAuditEvent(svc.DB, c, "contact.channel_consent.update", "contact", contactID, fiber.Map{
			"channel": body.Channel, "purpose": body.Purpose, "status": body.Status,
		})
		return c.JSON(fiber.Map{"message": "Consentimento atualizado"})
	}
}

func ListCampaignSuppressions(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		rows, err := svc.DB.Query(`
			SELECT id, channel, destination, COALESCE(reason, ''), COALESCE(source, ''), created_at
			FROM campaign_suppressions WHERE company_id = $1 ORDER BY created_at DESC
		`, companyID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erro ao listar supressões"})
		}
		defer rows.Close()
		items := []fiber.Map{}
		for rows.Next() {
			var id, channel, destination, reason, source string
			var createdAt time.Time
			if rows.Scan(&id, &channel, &destination, &reason, &source, &createdAt) == nil {
				items = append(items, fiber.Map{"id": id, "channel": channel, "destination": destination, "reason": reason, "source": source, "created_at": createdAt})
			}
		}
		return c.JSON(fiber.Map{"suppressions": items})
	}
}

func CreateCampaignSuppression(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		var body struct {
			Channel     string `json:"channel"`
			Destination string `json:"destination"`
			Reason      string `json:"reason"`
		}
		if c.BodyParser(&body) != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Dados inválidos"})
		}
		body.Channel = strings.ToLower(strings.TrimSpace(body.Channel))
		body.Destination = normalizeSuppressionDestination(body.Channel, body.Destination)
		if (body.Channel != "email" && body.Channel != "whatsapp") || body.Destination == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Canal ou destino inválido"})
		}
		var id string
		err := svc.DB.QueryRow(`
			INSERT INTO campaign_suppressions(id, company_id, channel, destination, reason, source)
			VALUES ($1, $2, $3, $4, NULLIF($5, ''), 'manual')
			ON CONFLICT (company_id, channel, destination) DO UPDATE SET reason = EXCLUDED.reason
			RETURNING id
		`, uuid.New().String(), companyID, body.Channel, body.Destination, body.Reason).Scan(&id)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erro ao adicionar supressão"})
		}
		logAuditEvent(svc.DB, c, "campaign.suppression.create", "campaign", "", fiber.Map{"channel": body.Channel, "destination": body.Destination})
		return c.Status(201).JSON(fiber.Map{"id": id})
	}
}

func DeleteCampaignSuppression(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		result, err := svc.DB.Exec(`DELETE FROM campaign_suppressions WHERE id = $1 AND company_id = $2`, c.Params("id"), companyID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erro ao remover supressão"})
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return c.Status(404).JSON(fiber.Map{"error": "Supressão não encontrada"})
		}
		logAuditEvent(svc.DB, c, "campaign.suppression.delete", "campaign", "", fiber.Map{"suppression_id": c.Params("id")})
		return c.SendStatus(204)
	}
}

func ListDataSubjectRequests(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		rows, err := svc.DB.Query(`
			SELECT r.id, r.contact_id, r.request_type, r.status, COALESCE(r.requester_name, ''),
			       COALESCE(r.requester_email, ''), COALESCE(r.notes, ''), r.due_at,
			       r.completed_at, r.created_at, COALESCE(c.name, '')
			FROM data_subject_requests r
			LEFT JOIN contacts c ON c.id = r.contact_id AND c.company_id = r.company_id
			WHERE r.company_id = $1 ORDER BY r.created_at DESC
		`, companyID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erro ao listar solicitações"})
		}
		defer rows.Close()
		items := []fiber.Map{}
		for rows.Next() {
			var id, requestType, status, requesterName, requesterEmail, notes, contactName string
			var contactID sql.NullString
			var dueAt, completedAt sql.NullTime
			var createdAt time.Time
			if err := rows.Scan(&id, &contactID, &requestType, &status, &requesterName, &requesterEmail,
				&notes, &dueAt, &completedAt, &createdAt, &contactName); err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "Erro ao ler solicitações"})
			}
			items = append(items, fiber.Map{
				"id": id, "contact_id": nullableStringMapValue(contactID), "contact_name": contactName,
				"request_type": requestType, "status": status, "requester_name": requesterName,
				"requester_email": requesterEmail, "notes": notes, "due_at": nullableTimeValue(dueAt),
				"completed_at": nullableTimeValue(completedAt), "created_at": createdAt,
			})
		}
		return c.JSON(fiber.Map{"requests": items})
	}
}

func CreateDataSubjectRequest(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		userID := c.Locals("user_id").(string)
		var body struct {
			ContactID      string `json:"contact_id"`
			RequestType    string `json:"request_type"`
			RequesterName  string `json:"requester_name"`
			RequesterEmail string `json:"requester_email"`
			Notes          string `json:"notes"`
		}
		if c.BodyParser(&body) != nil || !validRequestTypes[body.RequestType] {
			return c.Status(400).JSON(fiber.Map{"error": "Tipo de solicitação inválido"})
		}
		if body.ContactID != "" {
			var exists bool
			_ = svc.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM contacts WHERE id = $1 AND company_id = $2)`, body.ContactID, companyID).Scan(&exists)
			if !exists {
				return c.Status(404).JSON(fiber.Map{"error": "Contato não encontrado"})
			}
		}
		id := uuid.New().String()
		_, err := svc.DB.Exec(`
			INSERT INTO data_subject_requests
				(id, company_id, contact_id, request_type, requester_name, requester_email,
				 notes, due_at, created_by)
			VALUES ($1, $2, NULLIF($3, '')::uuid, $4, NULLIF($5, ''), NULLIF($6, ''),
			        NULLIF($7, ''), NOW() + INTERVAL '15 days', NULLIF($8, '')::uuid)
		`, id, companyID, body.ContactID, body.RequestType, body.RequesterName, body.RequesterEmail, body.Notes, userID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erro ao criar solicitação"})
		}
		logAuditEvent(svc.DB, c, "privacy.request.create", "contact", body.ContactID, fiber.Map{"request_id": id, "type": body.RequestType})
		return c.Status(201).JSON(fiber.Map{"id": id, "due_in_days": 15})
	}
}

func UpdateDataSubjectRequest(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		userID := c.Locals("user_id").(string)
		var body struct {
			Status     string `json:"status"`
			Notes      string `json:"notes"`
			AssignedTo string `json:"assigned_to"`
		}
		if c.BodyParser(&body) != nil || !validRequestStatuses[body.Status] {
			return c.Status(400).JSON(fiber.Map{"error": "Status inválido"})
		}
		result, err := svc.DB.Exec(`
			UPDATE data_subject_requests SET status = $1, notes = NULLIF($2, ''),
				assigned_to = COALESCE(NULLIF($3, '')::uuid, NULLIF($4, '')::uuid),
				completed_at = CASE WHEN $1 IN ('completed', 'rejected') THEN NOW() ELSE NULL END,
				updated_at = NOW()
			WHERE id = $5 AND company_id = $6
		`, body.Status, body.Notes, body.AssignedTo, userID, c.Params("id"), companyID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erro ao atualizar solicitação"})
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return c.Status(404).JSON(fiber.Map{"error": "Solicitação não encontrada"})
		}
		logAuditEvent(svc.DB, c, "privacy.request.update", "contact", "", fiber.Map{"request_id": c.Params("id"), "status": body.Status})
		return c.JSON(fiber.Map{"message": "Solicitação atualizada"})
	}
}

func GetRetentionPolicies(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		rows, err := svc.DB.Query(`
			SELECT id, resource_type, retention_days, action, is_active, updated_at
			FROM data_retention_policies WHERE company_id = $1 ORDER BY resource_type
		`, companyID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erro ao consultar retenção"})
		}
		defer rows.Close()
		items := []fiber.Map{}
		for rows.Next() {
			var id, resourceType, action string
			var days int
			var active bool
			var updatedAt time.Time
			if rows.Scan(&id, &resourceType, &days, &action, &active, &updatedAt) == nil {
				items = append(items, fiber.Map{"id": id, "resource_type": resourceType, "retention_days": days, "action": action, "is_active": active, "updated_at": updatedAt})
			}
		}
		return c.JSON(fiber.Map{"policies": items})
	}
}

func UpsertRetentionPolicy(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		userID := c.Locals("user_id").(string)
		var body struct {
			ResourceType  string `json:"resource_type"`
			RetentionDays int    `json:"retention_days"`
			IsActive      bool   `json:"is_active"`
		}
		if c.BodyParser(&body) != nil || body.ResourceType != "inactive_contacts" || body.RetentionDays < 30 {
			return c.Status(400).JSON(fiber.Map{"error": "Política inválida; o mínimo é 30 dias"})
		}
		_, err := svc.DB.Exec(`
			INSERT INTO data_retention_policies
				(id, company_id, resource_type, retention_days, action, is_active, created_by)
			VALUES ($1, $2, $3, $4, 'anonymize', $5, NULLIF($6, '')::uuid)
			ON CONFLICT (company_id, resource_type) DO UPDATE SET
				retention_days = EXCLUDED.retention_days, is_active = EXCLUDED.is_active,
				updated_at = NOW()
		`, uuid.New().String(), companyID, body.ResourceType, body.RetentionDays, body.IsActive, userID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erro ao salvar política"})
		}
		logAuditEvent(svc.DB, c, "privacy.retention.update", "company", companyID, fiber.Map{"days": body.RetentionDays, "active": body.IsActive})
		return c.JSON(fiber.Map{"message": "Política de retenção atualizada"})
	}
}

func RegisterPrivacyJobs(svc *services.Container) {
	svc.Jobs.Register("privacy.retention", func(ctx context.Context, raw json.RawMessage) error {
		var payload struct {
			CompanyID string `json:"company_id"`
		}
		if json.Unmarshal(raw, &payload) != nil || payload.CompanyID == "" {
			return fmt.Errorf("invalid retention payload")
		}
		return runRetentionPolicy(ctx, svc.DB, payload.CompanyID)
	})
}

func ScheduleRetentionJobs(svc *services.Container) error {
	rows, err := svc.DB.Query(`SELECT DISTINCT company_id FROM data_retention_policies WHERE is_active = true`)
	if err != nil {
		return err
	}
	defer rows.Close()
	today := time.Now().UTC().Format("2006-01-02")
	for rows.Next() {
		var companyID string
		if err := rows.Scan(&companyID); err != nil {
			return err
		}
		if _, err := svc.Jobs.Enqueue(companyID, "privacy.retention", companyID+":"+today,
			fiber.Map{"company_id": companyID}, time.Now()); err != nil {
			return err
		}
	}
	return rows.Err()
}

func runRetentionPolicy(ctx context.Context, db *sql.DB, companyID string) error {
	_, err := db.ExecContext(ctx, `
		WITH policy AS (
			SELECT retention_days FROM data_retention_policies
			WHERE company_id = $1 AND resource_type = 'inactive_contacts' AND is_active = true
		), candidates AS (
			SELECT c.id
			FROM contacts c, policy p
			WHERE c.company_id = $1
			  AND c.updated_at < NOW() - (p.retention_days * INTERVAL '1 day')
			  AND NOT EXISTS (
				SELECT 1 FROM conversations conversation
				WHERE conversation.contact_id = c.id
				  AND conversation.updated_at >= NOW() - (p.retention_days * INTERVAL '1 day')
			  )
			  AND NOT EXISTS (
				SELECT 1 FROM data_subject_requests request
				WHERE request.contact_id = c.id AND request.status IN ('open', 'in_progress')
			  )
		)
		UPDATE contacts c SET
			name = 'Contato anonimizado',
			phone = NULL,
			email = NULL,
			company_name = NULL,
			position = NULL,
			city = NULL,
			state = NULL,
			avatar_url = NULL,
			notes = NULL,
			custom_fields = '{}'::jsonb,
			updated_at = NOW()
		FROM candidates WHERE c.id = candidates.id
	`, companyID)
	return err
}

func normalizeSuppressionDestination(channel, value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if channel == "whatsapp" {
		var digits strings.Builder
		for _, r := range value {
			if r >= '0' && r <= '9' {
				digits.WriteRune(r)
			}
		}
		return digits.String()
	}
	return value
}

type unsubscribeClaims struct {
	CompanyID string `json:"company_id"`
	ContactID string `json:"contact_id"`
	Channel   string `json:"channel"`
	ExpiresAt int64  `json:"expires_at"`
}

func createUnsubscribeToken(companyID, contactID, channel, secret string, expiresAt time.Time) string {
	raw, _ := json.Marshal(unsubscribeClaims{
		CompanyID: companyID, ContactID: contactID, Channel: channel, ExpiresAt: expiresAt.Unix(),
	})
	payload := base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func parseUnsubscribeToken(token, secret string) (unsubscribeClaims, bool) {
	var claims unsubscribeClaims
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return claims, false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(parts[0]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return claims, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || json.Unmarshal(raw, &claims) != nil || claims.ExpiresAt < time.Now().Unix() {
		return claims, false
	}
	return claims, claims.CompanyID != "" && claims.ContactID != "" && claims.Channel == "email"
}

func unsubscribeURL(svc *services.Container, companyID, contactID string) string {
	baseURL := strings.TrimSuffix(svc.Config.EvolutionWebhookURL, "/api/webhooks/evolution")
	token := createUnsubscribeToken(companyID, contactID, "email", svc.Config.JWTSecret, time.Now().Add(180*24*time.Hour))
	return strings.TrimRight(baseURL, "/") + "/api/marketing/unsubscribe?token=" + url.QueryEscape(token)
}

func HandleMarketingUnsubscribe(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, valid := parseUnsubscribeToken(c.Query("token"), svc.Config.JWTSecret)
		if !valid {
			return c.Status(fiber.StatusBadRequest).Type("html", "utf-8").SendString(
				"<!doctype html><html lang=\"pt-BR\"><body><h1>Link inválido ou expirado</h1></body></html>",
			)
		}
		var email string
		err := svc.DB.QueryRow(`
			SELECT LOWER(TRIM(email)) FROM contacts
			WHERE id = $1 AND company_id = $2 AND NULLIF(TRIM(email), '') IS NOT NULL
		`, claims.ContactID, claims.CompanyID).Scan(&email)
		if err != nil {
			return c.Status(404).Type("html", "utf-8").SendString(
				"<!doctype html><html lang=\"pt-BR\"><body><h1>Contato não encontrado</h1></body></html>",
			)
		}
		tx, err := svc.DB.Begin()
		if err != nil {
			return c.SendStatus(500)
		}
		defer tx.Rollback()
		_, err = tx.Exec(`
			INSERT INTO contact_channel_consents
				(id, company_id, contact_id, channel, purpose, status, source, revoked_at)
			VALUES ($1, $2, $3, 'email', 'marketing', 'opted_out', 'unsubscribe_link', NOW())
			ON CONFLICT (contact_id, channel, purpose) DO UPDATE SET
				status = 'opted_out', source = 'unsubscribe_link', revoked_at = NOW(), updated_at = NOW()
		`, uuid.New().String(), claims.CompanyID, claims.ContactID)
		if err == nil {
			_, err = tx.Exec(`
				INSERT INTO campaign_suppressions(id, company_id, channel, destination, reason, source)
				VALUES ($1, $2, 'email', $3, 'opted_out', 'unsubscribe_link')
				ON CONFLICT (company_id, channel, destination) DO UPDATE SET
					reason = 'opted_out', source = 'unsubscribe_link'
			`, uuid.New().String(), claims.CompanyID, email)
		}
		if err != nil || tx.Commit() != nil {
			return c.SendStatus(500)
		}
		return c.Type("html", "utf-8").SendString(
			"<!doctype html><html lang=\"pt-BR\"><head><meta charset=\"utf-8\"><title>Descadastro concluído</title></head><body style=\"font-family:Arial;padding:40px\"><h1>Descadastro concluído</h1><p>Você não receberá mais campanhas por e-mail desta empresa.</p></body></html>",
		)
	}
}
