package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

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
	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()
	// One chain per tenant prevents concurrent administrative events from forking.
	if _, err = tx.Exec(`SELECT pg_advisory_xact_lock(hashtext($1))`, companyID); err != nil {
		return
	}
	var previousHash sql.NullString
	if err = tx.QueryRow(`
		SELECT entry_hash FROM audit_logs
		WHERE company_id = $1 AND entry_hash IS NOT NULL
		ORDER BY created_at DESC, id DESC LIMIT 1
	`, companyID).Scan(&previousHash); err != nil && err != sql.ErrNoRows {
		return
	}
	id := uuid.New().String()
	createdAt := time.Now().UTC().Truncate(time.Microsecond)
	entryHash := hashAuditEntry(previousHash.String, id, companyID, userID, action, entityType, entityID, c.IP(), createdAt, rawMetadata)
	if _, err = tx.Exec(`
		INSERT INTO audit_logs
			(id, company_id, user_id, action, entity_type, entity_id, ip_address, user_agent,
			 metadata, previous_hash, entry_hash, created_at)
		VALUES
			($1, $2, NULLIF($3, '')::uuid, $4, $5, NULLIF($6, '')::uuid, $7, $8,
			 $9::jsonb, NULLIF($10, ''), $11, $12)
	`, id, companyID, userID, action, entityType, entityID, c.IP(), c.Get("User-Agent"),
		string(rawMetadata), previousHash.String, entryHash, createdAt); err != nil {
		return
	}
	_ = tx.Commit()
}

func hashAuditEntry(previousHash, id, companyID, userID, action, entityType, entityID, ipAddress string, createdAt time.Time, metadata []byte) string {
	hashInput := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%s",
		previousHash, id, companyID, userID, action, entityType, entityID,
		ipAddress, createdAt.Format(time.RFC3339Nano), string(metadata))
	return fmt.Sprintf("%x", sha256.Sum256([]byte(hashInput)))
}
