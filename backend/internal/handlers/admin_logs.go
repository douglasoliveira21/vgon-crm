package handlers

import (
	"database/sql"
	"time"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

func GetAuditLogs(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		limit := c.QueryInt("limit", 100)
		if limit <= 0 || limit > 300 {
			limit = 100
		}
		action := c.Query("action")

		filter := ""
		args := []interface{}{companyID, limit}
		if action != "" {
			filter = " AND al.action ILIKE $3"
			args = append(args, "%"+action+"%")
		}

		rows, err := svc.DB.Query(`
			SELECT al.id, al.action,
			       COALESCE(al.entity_type, al.resource_type, ''),
			       COALESCE(al.entity_id, al.resource_id),
			       COALESCE(u.name, 'Sistema'),
			       COALESCE(u.email, ''),
			       COALESCE(al.ip_address, ''),
			       COALESCE(al.metadata, al.details, '{}'::jsonb),
			       al.created_at
			FROM audit_logs al
			LEFT JOIN users u ON u.id = al.user_id
			WHERE al.company_id = $1`+filter+`
			ORDER BY al.created_at DESC
			LIMIT $2
		`, args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		logs := []fiber.Map{}
		for rows.Next() {
			var id, actionName, entityType, userName, userEmail, ipAddress string
			var entityID sql.NullString
			var metadata []byte
			var createdAt time.Time
			if err := rows.Scan(&id, &actionName, &entityType, &entityID, &userName, &userEmail, &ipAddress, &metadata, &createdAt); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
			logs = append(logs, fiber.Map{
				"id":          id,
				"action":      actionName,
				"entity_type": entityType,
				"entity_id":   nullableStringMapValue(entityID),
				"user_name":   userName,
				"user_email":  userEmail,
				"ip_address":  ipAddress,
				"metadata":    string(metadata),
				"created_at":  createdAt,
			})
		}
		return c.JSON(fiber.Map{"logs": logs})
	}
}
