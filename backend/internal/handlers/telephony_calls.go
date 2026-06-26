package handlers

import (
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
