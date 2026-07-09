package handlers

import (
	"database/sql"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func logAuditEvent(db *sql.DB, c *fiber.Ctx, action, entityType, entityID string, metadata interface{}) {
	companyID, _ := c.Locals("company_id").(string)
	userID, _ := c.Locals("user_id").(string)
	if companyID == "" {
		return
	}
	rawMetadata := []byte("{}")
	if metadata != nil {
		if marshaled, err := json.Marshal(metadata); err == nil {
			rawMetadata = marshaled
		}
	}
	_, _ = db.Exec(`
		INSERT INTO audit_logs (id, company_id, user_id, action, entity_type, entity_id, ip_address, user_agent, metadata)
		VALUES ($1, $2, NULLIF($3, '')::uuid, $4, $5, NULLIF($6, '')::uuid, $7, $8, $9::jsonb)
	`, uuid.New().String(), companyID, userID, action, entityType, entityID, c.IP(), c.Get("User-Agent"), string(rawMetadata))
}
