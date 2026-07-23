package handlers

import (
	"database/sql"
	"fmt"
	"log"
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

// DeleteTenant permanently deletes a tenant and all company data.
func DeleteTenant(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := c.Params("id")

		tx, err := svc.DB.Begin()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to start delete transaction"})
		}
		defer tx.Rollback()

		var companyName string
		err = tx.QueryRow("SELECT name FROM companies WHERE id = $1", tenantID).Scan(&companyName)
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tenant not found"})
		}
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load tenant"})
		}

		if err = cleanupForeignKeyReferences(tx, "companies", tenantID); err != nil {
			log.Printf("[ADMIN] failed to clean references for tenant %s: %v", tenantID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to clean tenant references",
				"details": err.Error(),
			})
		}

		if _, err = tx.Exec("DELETE FROM companies WHERE id = $1", tenantID); err != nil {
			log.Printf("[ADMIN] failed to delete tenant %s: %v", tenantID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to delete tenant data",
				"details": err.Error(),
			})
		}

		if err = tx.Commit(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to commit tenant deletion"})
		}

		return c.JSON(fiber.Map{"message": "Tenant permanently deleted", "name": companyName})
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

func execIfTableExists(tx *sql.Tx, tableName string, query string, args ...interface{}) error {
	var exists bool
	if err := tx.QueryRow("SELECT to_regclass($1) IS NOT NULL", "public."+tableName).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return nil
	}
	_, err := tx.Exec(query, args...)
	return err
}

func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

func cleanupForeignKeyReferences(tx *sql.Tx, referencedTable string, referencedID string) error {
	rows, err := tx.Query(`
		SELECT
			kcu.table_schema,
			kcu.table_name,
			kcu.column_name,
			COALESCE(c.is_nullable = 'YES', false) AS is_nullable
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		JOIN information_schema.columns c
			ON c.table_schema = kcu.table_schema
			AND c.table_name = kcu.table_name
			AND c.column_name = kcu.column_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND ccu.table_schema = 'public'
			AND ccu.table_name = $1
			AND ccu.column_name = 'id'
			AND kcu.table_schema = 'public'
			AND NOT (kcu.table_schema = 'public' AND kcu.table_name = $1)
	`, referencedTable)
	if err != nil {
		return err
	}

	type foreignKeyReference struct {
		schemaName string
		tableName  string
		columnName string
		isNullable bool
	}
	var references []foreignKeyReference
	for rows.Next() {
		var reference foreignKeyReference
		if err := rows.Scan(&reference.schemaName, &reference.tableName, &reference.columnName, &reference.isNullable); err != nil {
			rows.Close()
			return err
		}
		references = append(references, reference)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	for _, reference := range references {
		qualifiedTable := quoteIdentifier(reference.schemaName) + "." + quoteIdentifier(reference.tableName)
		quotedColumn := quoteIdentifier(reference.columnName)
		if reference.isNullable {
			_, err = tx.Exec("UPDATE "+qualifiedTable+" SET "+quotedColumn+" = NULL WHERE "+quotedColumn+" = $1", referencedID)
		} else {
			_, err = tx.Exec("DELETE FROM "+qualifiedTable+" WHERE "+quotedColumn+" = $1", referencedID)
		}
		if err != nil {
			return fmt.Errorf("%s.%s.%s: %w", reference.schemaName, reference.tableName, reference.columnName, err)
		}
	}

	return nil
}

func cleanupUserReferences(tx *sql.Tx, userID string) error {
	return cleanupForeignKeyReferences(tx, "users", userID)
}

// AdminGetTenantUsers returns all users for a specific tenant
func AdminGetTenantUsers(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := c.Params("id")

		rows, err := svc.DB.Query(`
			SELECT u.id, u.name, u.email, u.is_active, u.is_online, COALESCE(u.is_super_admin, false),
			       COALESCE(r.name, '') as role_name, COALESCE(r.slug, '') as role_slug, u.created_at
			FROM users u
			LEFT JOIN roles r ON u.role_id = r.id
			WHERE u.company_id = $1
			ORDER BY u.created_at DESC
		`, tenantID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch users"})
		}
		defer rows.Close()

		type UserRow struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			Email        string `json:"email"`
			IsActive     bool   `json:"is_active"`
			IsOnline     bool   `json:"is_online"`
			IsSuperAdmin bool   `json:"is_super_admin"`
			RoleName     string `json:"role_name"`
			RoleSlug     string `json:"role_slug"`
			CreatedAt    string `json:"created_at"`
		}

		users := []UserRow{}
		for rows.Next() {
			var u UserRow
			rows.Scan(&u.ID, &u.Name, &u.Email, &u.IsActive, &u.IsOnline, &u.IsSuperAdmin, &u.RoleName, &u.RoleSlug, &u.CreatedAt)
			users = append(users, u)
		}

		return c.JSON(fiber.Map{"users": users})
	}
}

// AdminCreateTenantUser creates a new user for a specific tenant
func AdminCreateTenantUser(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := c.Params("id")

		var req struct {
			Name         string `json:"name"`
			Email        string `json:"email"`
			Password     string `json:"password"`
			Role         string `json:"role"` // super-admin, admin, agent, supervisor
			IsSuperAdmin bool   `json:"is_super_admin"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if req.Name == "" || req.Email == "" || req.Password == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Name, email and password are required"})
		}

		// Check max users limit
		var currentUsers, maxUsers int
		svc.DB.QueryRow("SELECT COUNT(*) FROM users WHERE company_id = $1", tenantID).Scan(&currentUsers)
		svc.DB.QueryRow("SELECT max_users FROM companies WHERE id = $1", tenantID).Scan(&maxUsers)
		if currentUsers >= maxUsers {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("Limite de usuários atingido (%d/%d). Aumente o plano.", currentUsers, maxUsers)})
		}

		// Check if email already exists
		var exists bool
		svc.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", strings.ToLower(req.Email)).Scan(&exists)
		if exists {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email já está em uso"})
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to hash password"})
		}

		// Get role
		roleSlug := req.Role
		if roleSlug == "" {
			roleSlug = "agent"
		}
		isSuperAdmin := roleSlug == "super-admin" || req.IsSuperAdmin
		if isSuperAdmin {
			roleSlug = "super-admin"
		}
		var roleID string
		err = svc.DB.QueryRow("SELECT id FROM roles WHERE slug = $1 AND (company_id = $2 OR is_system = true) LIMIT 1", roleSlug, tenantID).Scan(&roleID)
		if err != nil {
			// Fallback to any system role
			svc.DB.QueryRow("SELECT id FROM roles WHERE is_system = true LIMIT 1").Scan(&roleID)
		}

		userID := uuid.New().String()
		_, err = svc.DB.Exec(`
			INSERT INTO users (id, company_id, role_id, name, email, password_hash, is_super_admin)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, userID, tenantID, roleID, req.Name, strings.ToLower(req.Email), string(hashedPassword), isSuperAdmin)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create user: " + err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"id":    userID,
			"name":  req.Name,
			"email": req.Email,
			"role":  roleSlug,
			"is_super_admin": isSuperAdmin,
		})
	}
}

// AdminResetUserPassword resets a user's password
func AdminResetUserPassword(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Params("userId")

		var req struct {
			NewPassword string `json:"new_password"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if req.NewPassword == "" || len(req.NewPassword) < 6 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password must be at least 6 characters"})
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to hash password"})
		}

		result, err := svc.DB.Exec("UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2", string(hashedPassword), userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update password"})
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}

		return c.JSON(fiber.Map{"message": "Password updated successfully"})
	}
}

// AdminUpdateUser updates a user's details
func AdminUpdateUser(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Params("userId")

		var req struct {
			Name         string `json:"name"`
			Email        string `json:"email"`
			Role         *string `json:"role"`
			IsActive     *bool  `json:"is_active"`
			IsSuperAdmin *bool  `json:"is_super_admin"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		updates := []string{}
		args := []interface{}{}
		argIdx := 1

		if req.Name != "" {
			updates = append(updates, "name = $"+itoa(argIdx))
			args = append(args, req.Name)
			argIdx++
		}
		if req.Email != "" {
			updates = append(updates, "email = $"+itoa(argIdx))
			args = append(args, strings.ToLower(req.Email))
			argIdx++
		}
		if req.IsActive != nil {
			updates = append(updates, "is_active = $"+itoa(argIdx))
			args = append(args, *req.IsActive)
			argIdx++
		}
		if req.Role != nil {
			roleSlug := *req.Role
			isSuperAdmin := roleSlug == "super-admin"
			var roleID string
			err := svc.DB.QueryRow(
				"SELECT id FROM roles WHERE slug = $1 AND is_system = true LIMIT 1",
				roleSlug,
			).Scan(&roleID)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid role"})
			}
			updates = append(updates, "role_id = $"+itoa(argIdx))
			args = append(args, roleID)
			argIdx++
			updates = append(updates, "is_super_admin = $"+itoa(argIdx))
			args = append(args, isSuperAdmin)
			argIdx++
		}
		if req.Role == nil && req.IsSuperAdmin != nil {
			updates = append(updates, "is_super_admin = $"+itoa(argIdx))
			args = append(args, *req.IsSuperAdmin)
			argIdx++
		}

		if len(updates) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No fields to update"})
		}

		updates = append(updates, "updated_at = NOW()")
		query := "UPDATE users SET " + strings.Join(updates, ", ") + " WHERE id = $" + itoa(argIdx)
		args = append(args, userID)

		_, err := svc.DB.Exec(query, args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update user"})
		}

		return c.JSON(fiber.Map{"message": "User updated successfully"})
	}
}

// AdminDeleteUser permanently deletes a user and user-owned history.
func AdminDeleteUser(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Params("userId")

		var companyID string
		var isSuperAdmin bool
		err := svc.DB.QueryRow(`
			SELECT company_id, COALESCE(is_super_admin, false)
			FROM users
			WHERE id = $1
		`, userID).Scan(&companyID, &isSuperAdmin)
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load user"})
		}
		if isSuperAdmin {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Super admin users cannot be deleted from tenant management"})
		}

		tx, err := svc.DB.Begin()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to start delete transaction"})
		}
		defer tx.Rollback()

		cleanupStatements := []string{
			`DELETE FROM message_attachments WHERE message_id IN (SELECT id FROM messages WHERE company_id = $2 AND sender_type = 'user' AND sender_id = $1::uuid)`,
			`DELETE FROM messages WHERE company_id = $2 AND sender_type = 'user' AND sender_id = $1::uuid`,
		}

		for _, stmt := range cleanupStatements {
			var execErr error
			if strings.Contains(stmt, "$2") {
				_, execErr = tx.Exec(stmt, userID, companyID)
			} else {
				_, execErr = tx.Exec(stmt, userID)
			}
			if execErr != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Failed to clean user history",
					"details": execErr.Error(),
				})
			}
		}

		if err = cleanupUserReferences(tx, userID); err != nil {
			log.Printf("[ADMIN] failed to clean references for user %s in company %s: %v", userID, companyID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to clean user references",
				"details": err.Error(),
			})
		}

		result, err := tx.Exec("DELETE FROM users WHERE id = $1", userID)
		if err != nil {
			log.Printf("[ADMIN] failed to delete user %s in company %s: %v", userID, companyID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to delete user",
				"details": err.Error(),
			})
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}

		if err = tx.Commit(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to commit user deletion"})
		}

		return c.JSON(fiber.Map{"message": "User permanently deleted"})
	}
}
