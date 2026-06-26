package handlers

import (
	"fmt"
	"strings"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// GetTenants returns all companies/tenants with aggregated stats
func GetTenants(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		search := c.Query("search", "")
		status := c.Query("status", "")

		query := `
			SELECT 
				c.id, c.name, c.slug, c.plan, c.max_users, c.max_channels, c.is_active, c.created_at,
				COALESCE(uc.user_count, 0) as user_count,
				COALESCE(cc.conversation_count, 0) as conversation_count
			FROM companies c
			LEFT JOIN (
				SELECT company_id, COUNT(*) as user_count FROM users GROUP BY company_id
			) uc ON uc.company_id = c.id
			LEFT JOIN (
				SELECT company_id, COUNT(*) as conversation_count FROM conversations GROUP BY company_id
			) cc ON cc.company_id = c.id
			WHERE 1=1
		`

		args := []interface{}{}
		argIdx := 1

		if search != "" {
			query += ` AND (LOWER(c.name) LIKE $` + itoa(argIdx) + ` OR LOWER(c.slug) LIKE $` + itoa(argIdx) + `)`
			args = append(args, "%"+strings.ToLower(search)+"%")
			argIdx++
		}

		if status == "active" {
			query += ` AND c.is_active = true`
		} else if status == "inactive" {
			query += ` AND c.is_active = false`
		}

		query += ` ORDER BY c.created_at DESC`

		rows, err := svc.DB.Query(query, args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch tenants",
			})
		}
		defer rows.Close()

		type TenantRow struct {
			ID                string `json:"id"`
			Name              string `json:"name"`
			Slug              string `json:"slug"`
			Plan              string `json:"plan"`
			MaxUsers          int    `json:"max_users"`
			MaxChannels       int    `json:"max_channels"`
			IsActive          bool   `json:"is_active"`
			CreatedAt         string `json:"created_at"`
			UserCount         int    `json:"user_count"`
			ConversationCount int    `json:"conversation_count"`
		}

		tenants := []TenantRow{}
		for rows.Next() {
			var t TenantRow
			err := rows.Scan(
				&t.ID, &t.Name, &t.Slug, &t.Plan, &t.MaxUsers, &t.MaxChannels,
				&t.IsActive, &t.CreatedAt, &t.UserCount, &t.ConversationCount,
			)
			if err != nil {
				continue
			}
			tenants = append(tenants, t)
		}

		return c.JSON(fiber.Map{
			"tenants": tenants,
			"total":   len(tenants),
		})
	}
}

// CreateTenant creates a new company (tenant) with an admin user
func CreateTenant(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			CompanyName string `json:"company_name"`
			Slug        string `json:"slug"`
			AdminEmail  string `json:"admin_email"`
			AdminPass   string `json:"admin_password"`
			AdminName   string `json:"admin_name"`
			Plan        string `json:"plan"`
			MaxUsers    int    `json:"max_users"`
			MaxChannels int    `json:"max_channels"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		if req.CompanyName == "" || req.AdminEmail == "" || req.AdminPass == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Company name, admin email and password are required"})
		}

		if req.Plan == "" {
			req.Plan = "basic"
		}
		if req.MaxUsers == 0 {
			req.MaxUsers = 5
		}
		if req.MaxChannels == 0 {
			req.MaxChannels = 2
		}
		if req.AdminName == "" {
			req.AdminName = "Admin"
		}

		// Generate slug if not provided
		slug := req.Slug
		if slug == "" {
			slug = strings.ToLower(strings.ReplaceAll(req.CompanyName, " ", "-"))
		}

		// Check if slug exists
		var exists bool
		svc.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM companies WHERE slug = $1)", slug).Scan(&exists)
		if exists {
			slug = slug + "-" + uuid.New().String()[:8]
		}

		// Create company
		companyID := uuid.New().String()
		_, err := svc.DB.Exec(`
			INSERT INTO companies (id, name, slug, plan, max_users, max_channels)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, companyID, req.CompanyName, slug, req.Plan, req.MaxUsers, req.MaxChannels)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create company: " + err.Error()})
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.AdminPass), bcrypt.DefaultCost)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to hash password"})
		}

		// Get admin role
		var roleID string
		err = svc.DB.QueryRow("SELECT id FROM roles WHERE slug = 'admin' AND is_system = true LIMIT 1").Scan(&roleID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get admin role"})
		}

		// Create admin user
		userID := uuid.New().String()
		_, err = svc.DB.Exec(`
			INSERT INTO users (id, company_id, role_id, name, email, password_hash)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, userID, companyID, roleID, req.AdminName, strings.ToLower(req.AdminEmail), string(hashedPassword))
		if err != nil {
			// Rollback company creation
			svc.DB.Exec("DELETE FROM companies WHERE id = $1", companyID)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create admin user: " + err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message":    "Tenant created successfully",
			"company_id": companyID,
			"user_id":    userID,
			"slug":       slug,
		})
	}
}

// GetTenant returns details of a single tenant
func GetTenant(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := c.Params("id")

		var tenant struct {
			ID          string  `json:"id"`
			Name        string  `json:"name"`
			Slug        string  `json:"slug"`
			Document    *string `json:"document"`
			Email       *string `json:"email"`
			Phone       *string `json:"phone"`
			Plan        string  `json:"plan"`
			MaxUsers    int     `json:"max_users"`
			MaxChannels int     `json:"max_channels"`
			IsActive    bool    `json:"is_active"`
			CreatedAt   string  `json:"created_at"`
			UpdatedAt   string  `json:"updated_at"`
		}

		err := svc.DB.QueryRow(`
			SELECT id, name, slug, document, email, phone, plan, max_users, max_channels, is_active, created_at, updated_at
			FROM companies WHERE id = $1
		`, tenantID).Scan(
			&tenant.ID, &tenant.Name, &tenant.Slug, &tenant.Document, &tenant.Email,
			&tenant.Phone, &tenant.Plan, &tenant.MaxUsers, &tenant.MaxChannels,
			&tenant.IsActive, &tenant.CreatedAt, &tenant.UpdatedAt,
		)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tenant not found"})
		}

		// Get users for this tenant
		type UserInfo struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Email     string `json:"email"`
			IsActive  bool   `json:"is_active"`
			IsOnline  bool   `json:"is_online"`
			CreatedAt string `json:"created_at"`
		}

		users := []UserInfo{}
		userRows, err := svc.DB.Query(`
			SELECT id, name, email, is_active, is_online, created_at
			FROM users WHERE company_id = $1 ORDER BY created_at DESC
		`, tenantID)
		if err == nil {
			defer userRows.Close()
			for userRows.Next() {
				var u UserInfo
				userRows.Scan(&u.ID, &u.Name, &u.Email, &u.IsActive, &u.IsOnline, &u.CreatedAt)
				users = append(users, u)
			}
		}

		// Get stats
		var conversationCount, messageCount, channelCount int
		svc.DB.QueryRow("SELECT COUNT(*) FROM conversations WHERE company_id = $1", tenantID).Scan(&conversationCount)
		svc.DB.QueryRow("SELECT COUNT(*) FROM messages WHERE company_id = $1", tenantID).Scan(&messageCount)
		svc.DB.QueryRow("SELECT COUNT(*) FROM channels WHERE company_id = $1", tenantID).Scan(&channelCount)

		return c.JSON(fiber.Map{
			"tenant": tenant,
			"users":  users,
			"stats": fiber.Map{
				"conversations": conversationCount,
				"messages":      messageCount,
				"channels":      channelCount,
				"users":         len(users),
			},
		})
	}
}

// UpdateTenant updates a tenant's information
func UpdateTenant(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := c.Params("id")

		var req struct {
			Name        string `json:"name"`
			Plan        string `json:"plan"`
			MaxUsers    int    `json:"max_users"`
			MaxChannels int    `json:"max_channels"`
			IsActive    *bool  `json:"is_active"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		// Build dynamic update
		updates := []string{}
		args := []interface{}{}
		argIdx := 1

		if req.Name != "" {
			updates = append(updates, "name = $"+itoa(argIdx))
			args = append(args, req.Name)
			argIdx++
		}
		if req.Plan != "" {
			updates = append(updates, "plan = $"+itoa(argIdx))
			args = append(args, req.Plan)
			argIdx++
		}
		if req.MaxUsers > 0 {
			updates = append(updates, "max_users = $"+itoa(argIdx))
			args = append(args, req.MaxUsers)
			argIdx++
		}
		if req.MaxChannels > 0 {
			updates = append(updates, "max_channels = $"+itoa(argIdx))
			args = append(args, req.MaxChannels)
			argIdx++
		}
		if req.IsActive != nil {
			updates = append(updates, "is_active = $"+itoa(argIdx))
			args = append(args, *req.IsActive)
			argIdx++
		}

		if len(updates) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No fields to update"})
		}

		updates = append(updates, "updated_at = NOW()")
		query := "UPDATE companies SET " + strings.Join(updates, ", ") + " WHERE id = $" + itoa(argIdx)
		args = append(args, tenantID)

		result, err := svc.DB.Exec(query, args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update tenant"})
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tenant not found"})
		}

		return c.JSON(fiber.Map{"message": "Tenant updated successfully"})
	}
}

// DeleteTenant deactivates a tenant (soft delete)
func DeleteTenant(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := c.Params("id")

		result, err := svc.DB.Exec("UPDATE companies SET is_active = false, updated_at = NOW() WHERE id = $1", tenantID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to deactivate tenant"})
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tenant not found"})
		}

		// Deactivate all users of this tenant
		svc.DB.Exec("UPDATE users SET is_active = false WHERE company_id = $1", tenantID)

		return c.JSON(fiber.Map{"message": "Tenant deactivated successfully"})
	}
}

// GetAdminStats returns global platform statistics
func GetAdminStats(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var totalTenants, activeTenants, totalUsers, activeUsers int
		var totalConversations, openConversations, totalMessages int

		svc.DB.QueryRow("SELECT COUNT(*) FROM companies").Scan(&totalTenants)
		svc.DB.QueryRow("SELECT COUNT(*) FROM companies WHERE is_active = true").Scan(&activeTenants)
		svc.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)
		svc.DB.QueryRow("SELECT COUNT(*) FROM users WHERE is_active = true").Scan(&activeUsers)
		svc.DB.QueryRow("SELECT COUNT(*) FROM conversations").Scan(&totalConversations)
		svc.DB.QueryRow("SELECT COUNT(*) FROM conversations WHERE status = 'open'").Scan(&openConversations)
		svc.DB.QueryRow("SELECT COUNT(*) FROM messages").Scan(&totalMessages)

		// Tenants created in last 30 days
		var newTenantsMonth int
		svc.DB.QueryRow("SELECT COUNT(*) FROM companies WHERE created_at >= NOW() - INTERVAL '30 days'").Scan(&newTenantsMonth)

		// Plan distribution
		planDistribution := map[string]int{}
		planRows, err := svc.DB.Query("SELECT plan, COUNT(*) FROM companies GROUP BY plan")
		if err == nil {
			defer planRows.Close()
			for planRows.Next() {
				var plan string
				var count int
				planRows.Scan(&plan, &count)
				planDistribution[plan] = count
			}
		}

		return c.JSON(fiber.Map{
			"total_tenants":       totalTenants,
			"active_tenants":      activeTenants,
			"total_users":         totalUsers,
			"active_users":        activeUsers,
			"total_conversations": totalConversations,
			"open_conversations":  openConversations,
			"total_messages":      totalMessages,
			"new_tenants_month":   newTenantsMonth,
			"plan_distribution":   planDistribution,
		})
	}
}

// itoa converts int to string for query parameter placeholders
func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
