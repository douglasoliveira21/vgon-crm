package handlers

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

func VerifyAuditChain(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		rows, err := svc.DB.Query(`
			SELECT id, user_id, action, entity_type, entity_id, ip_address,
			       metadata, previous_hash, entry_hash, created_at
			FROM audit_logs
			WHERE company_id = $1 AND entry_hash IS NOT NULL
			ORDER BY created_at, id
		`, companyID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erro ao verificar auditoria"})
		}
		defer rows.Close()

		expectedPrevious := ""
		checked := 0
		for rows.Next() {
			var id, action, entityType, entryHash string
			var userID, entityID, ipAddress, previousHash sql.NullString
			var metadata json.RawMessage
			var createdAt time.Time
			if err := rows.Scan(&id, &userID, &action, &entityType, &entityID, &ipAddress,
				&metadata, &previousHash, &entryHash, &createdAt); err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "Erro ao ler auditoria"})
			}
			var canonical interface{}
			if json.Unmarshal(metadata, &canonical) == nil {
				metadata, _ = json.Marshal(canonical)
			}
			expectedHash := hashAuditEntry(expectedPrevious, id, companyID, userID.String, action,
				entityType, entityID.String, ipAddress.String, createdAt.UTC(), metadata)
			if previousHash.String != expectedPrevious || entryHash != expectedHash {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"valid": false, "checked": checked, "invalid_entry_id": id,
					"message": "A cadeia de auditoria foi alterada ou está incompleta",
				})
			}
			expectedPrevious = entryHash
			checked++
		}
		return c.JSON(fiber.Map{"valid": true, "checked": checked})
	}
}
