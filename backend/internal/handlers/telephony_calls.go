package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// POST /api/telephony/call - Originate a call via Asterisk
func TelephonyCall(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		_ = c.Locals("user_id").(string)

		var body struct {
			Destination string `json:"destination"`
			Extension   string `json:"extension"`
			CallerID    string `json:"caller_id"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if body.Destination == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Destination is required"})
		}

		channelID, err := svc.Asterisk.OriginateCall(companyID, body.Extension, body.Destination, body.CallerID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		// Register in call history
		callID := uuid.New().String()
		svc.DB.Exec(`
			INSERT INTO call_records (id, company_id, extension_id, call_direction, call_status, from_number, to_number, started_at)
			VALUES ($1, $2, NULL, 'outbound', 'initiated', $3, $4, $5)
		`, callID, companyID, body.Extension, body.Destination, time.Now())

		return c.JSON(fiber.Map{
			"call_id":    callID,
			"channel_id": channelID,
			"status":     "initiated",
		})
	}
}

// POST /api/telephony/hangup - End a call
func TelephonyHangup(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body struct {
			ChannelID string `json:"channel_id"`
			CallID    string `json:"call_id"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if body.ChannelID != "" {
			if err := svc.Asterisk.HangupCall(companyID, body.ChannelID); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
		}

		// Update call record
		if body.CallID != "" {
			svc.DB.Exec("UPDATE call_records SET call_status = 'ended', ended_at = NOW(), duration_seconds = EXTRACT(EPOCH FROM (NOW() - started_at))::int WHERE id = $1", body.CallID)
		}

		return c.JSON(fiber.Map{"status": "ended"})
	}
}

// POST /api/telephony/transfer - Transfer a call
func TelephonyTransfer(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body struct {
			ChannelID   string `json:"channel_id"`
			Destination string `json:"destination"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if err := svc.Asterisk.TransferCall(companyID, body.ChannelID, body.Destination); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"status": "transferred"})
	}
}

// POST /api/telephony/hold - Put call on hold
func TelephonyHold(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body struct {
			ChannelID string `json:"channel_id"`
		}
		c.BodyParser(&body)

		if err := svc.Asterisk.HoldCall(companyID, body.ChannelID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"status": "on_hold"})
	}
}

// POST /api/telephony/unhold - Remove from hold
func TelephonyUnhold(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body struct {
			ChannelID string `json:"channel_id"`
		}
		c.BodyParser(&body)

		if err := svc.Asterisk.UnholdCall(companyID, body.ChannelID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"status": "active"})
	}
}

// GET /api/telephony/status - Get extension statuses
func TelephonyStatus(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		endpoints, err := svc.Asterisk.GetExtensionStatus(companyID)
		if err != nil {
			return c.JSON(fiber.Map{"endpoints": []interface{}{}, "error": err.Error()})
		}

		return c.JSON(fiber.Map{"endpoints": endpoints})
	}
}

// GET /api/telephony/recordings - List recordings for company
func GetRecordings(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		limit := c.QueryInt("limit", 50)
		search := c.Query("search", "")
		dateFrom := c.Query("date_from", "")
		dateTo := c.Query("date_to", "")

		query := `
			SELECT cr.id, cr.from_number, cr.to_number, cr.call_direction, cr.duration_seconds,
				   cr.recording_url, cr.created_at, COALESCE(co.name, '') as contact_name
			FROM call_records cr
			LEFT JOIN contacts co ON cr.contact_id = co.id
			WHERE cr.company_id = $1 AND cr.recording_url IS NOT NULL AND cr.recording_url != ''
		`
		args := []interface{}{companyID}
		argIdx := 2

		if search != "" {
			query += fmt.Sprintf(" AND (cr.from_number ILIKE $%d OR cr.to_number ILIKE $%d OR co.name ILIKE $%d)", argIdx, argIdx, argIdx)
			args = append(args, "%"+search+"%")
			argIdx++
		}

		if dateFrom != "" {
			query += fmt.Sprintf(" AND cr.created_at >= $%d", argIdx)
			args = append(args, dateFrom)
			argIdx++
		}

		if dateTo != "" {
			query += fmt.Sprintf(" AND cr.created_at <= $%d", argIdx)
			args = append(args, dateTo+"T23:59:59Z")
			argIdx++
		}

		query += " ORDER BY cr.created_at DESC LIMIT $" + fmt.Sprintf("%d", argIdx)
		args = append(args, limit)

		rows, err := svc.DB.Query(query, args...)
		if err != nil {
			return c.JSON(fiber.Map{"recordings": []interface{}{}})
		}
		defer rows.Close()

		var recordings []map[string]interface{}
		for rows.Next() {
			var id, fromNum, toNum, direction, contactName string
			var duration int
			var recordingURL *string
			var createdAt time.Time
			rows.Scan(&id, &fromNum, &toNum, &direction, &duration, &recordingURL, &createdAt, &contactName)

			rec := map[string]interface{}{
				"id":            id,
				"from_number":   fromNum,
				"to_number":     toNum,
				"direction":     direction,
				"duration":      duration,
				"recording_url": recordingURL,
				"created_at":    createdAt,
				"contact_name":  contactName,
			}
			recordings = append(recordings, rec)
		}

		if recordings == nil {
			recordings = []map[string]interface{}{}
		}

		return c.JSON(fiber.Map{"recordings": recordings})
	}
}

// GET /api/telephony/recordings/:id/play - Stream a recording file
func PlayRecording(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		id := c.Params("id")

		var recordingURL *string
		err := svc.DB.QueryRow("SELECT recording_url FROM call_records WHERE id = $1 AND company_id = $2", id, companyID).Scan(&recordingURL)
		if err != nil || recordingURL == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Recording not found"})
		}

		// Try to serve the file from disk
		// File path pattern: /app/storage/voip-recordings/{company_id}/{year}/{month}/{day}/{call_id}.wav
		filePath := fmt.Sprintf("/app/storage/voip-recordings/%s", *recordingURL)
		if _, err := os.Stat(filePath); err == nil {
			c.Set("Content-Type", "audio/wav")
			c.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s.wav\"", id))
			return c.SendFile(filePath)
		}

		// If not a local file, redirect to the URL
		return c.Redirect(*recordingURL, fiber.StatusTemporaryRedirect)
	}
}

// DELETE /api/telephony/recordings/:id - Delete a recording
func DeleteRecording(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		id := c.Params("id")

		var recordingURL *string
		err := svc.DB.QueryRow("SELECT recording_url FROM call_records WHERE id = $1 AND company_id = $2", id, companyID).Scan(&recordingURL)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Recording not found"})
		}

		// Try to delete file from disk
		if recordingURL != nil {
			filePath := fmt.Sprintf("/app/storage/voip-recordings/%s", *recordingURL)
			if _, err := os.Stat(filePath); err == nil {
				os.Remove(filePath)
				// Also try to remove empty parent directories
				dir := filepath.Dir(filePath)
				os.Remove(dir) // Only removes if empty
			}
		}

		// Update DB - clear recording_url
		svc.DB.Exec("UPDATE call_records SET recording_url = NULL WHERE id = $1 AND company_id = $2", id, companyID)

		return c.JSON(fiber.Map{"message": "Recording deleted"})
	}
}

// GET /api/calls/history - Get call history
func GetCallHistory(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		limit := c.QueryInt("limit", 50)

		rows, err := svc.DB.Query(`
			SELECT cr.id, cr.call_direction, cr.call_status, cr.from_number, cr.to_number,
				   cr.duration_seconds, cr.recording_url, cr.notes, cr.call_result, cr.created_at,
				   COALESCE(co.name, '') as contact_name
			FROM call_records cr
			LEFT JOIN contacts co ON cr.contact_id = co.id
			WHERE cr.company_id = $1
			ORDER BY cr.created_at DESC
			LIMIT $2
		`, companyID, limit)
		if err != nil {
			return c.JSON(fiber.Map{"calls": []interface{}{}})
		}
		defer rows.Close()

		var calls []map[string]interface{}
		for rows.Next() {
			var id, direction, status, fromNum, toNum, contactName string
			var duration int
			var recordingURL, notes, callResult *string
			var createdAt time.Time
			rows.Scan(&id, &direction, &status, &fromNum, &toNum, &duration, &recordingURL, &notes, &callResult, &createdAt, &contactName)

			calls = append(calls, map[string]interface{}{
				"id": id, "direction": direction, "status": status,
				"from_number": fromNum, "to_number": toNum,
				"duration": duration, "recording_url": recordingURL,
				"notes": notes, "call_result": callResult,
				"contact_name": contactName, "created_at": createdAt,
			})
		}

		return c.JSON(fiber.Map{"calls": calls})
	}
}
