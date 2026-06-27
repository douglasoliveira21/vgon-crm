package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// GET /api/telephony/webrtc/config - Browser-safe WebRTC registration config
func GetWebRTCConfig(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		userID := c.Locals("user_id").(string)

		var extensionID, extensionNumber, extensionPassword, displayName, sipUsername, webRTCDomain, webRTCWSURL, stunServer string
		err := svc.DB.QueryRow(`
			SELECT id, extension_number, extension_password, display_name,
			       COALESCE(NULLIF(sip_username, ''), extension_number),
			       COALESCE(webrtc_domain, 'voip.vgon.com.br'),
			       COALESCE(webrtc_ws_url, 'wss://voip.vgon.com.br:8089/ws'),
			       COALESCE(stun_server, 'stun:stun.l.google.com:19302')
			FROM phone_extensions
			WHERE company_id = $1 AND (user_id = $2 OR user_id IS NULL)
			ORDER BY CASE WHEN user_id = $2 THEN 0 ELSE 1 END, extension_number
			LIMIT 1
		`, companyID, userID).Scan(&extensionID, &extensionNumber, &extensionPassword, &displayName, &sipUsername, &webRTCDomain, &webRTCWSURL, &stunServer)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No WebRTC extension configured for this user"})
		}

		return c.JSON(fiber.Map{
			"extension_id":     extensionID,
			"extension_number": extensionNumber,
			"display_name":     displayName,
			"sip_uri":          fmt.Sprintf("sip:%s@%s", sipUsername, webRTCDomain),
			"sip_host":         webRTCDomain,
			"sip_port":         5060,
			"sip_domain":       webRTCDomain,
			"webrtc_domain":    webRTCDomain,
			"webrtc_ws_url":    webRTCWSURL,
			"transport":        "WSS",
			"stun_server":      stunServer,
			"username":         sipUsername,
			"password":         services.DecryptSecret(extensionPassword, svc.Config.JWTSecret),
		})
	}
}

// POST /api/telephony/calls/log-start - Save SIP.js call start in call_records
func StartWebRTCCallLog(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		userID := c.Locals("user_id").(string)

		var body struct {
			CallID       string `json:"call_id"`
			ChannelID    string `json:"channel_id"`
			Extension    string `json:"extension"`
			FromNumber   string `json:"from_number"`
			ToNumber     string `json:"to_number"`
			Direction    string `json:"direction"`
			Status       string `json:"status"`
			RecordingURL string `json:"recording_url"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}
		if body.Direction == "" {
			body.Direction = "outbound"
		}
		if body.Status == "" {
			body.Status = "initiated"
		}
		if body.CallID == "" {
			body.CallID = uuid.New().String()
		}

		contactID := findContactByPhone(svc, companyID, firstNonEmpty(body.FromNumber, body.ToNumber))
		extensionID := findExtensionID(svc, companyID, body.Extension)

		_, err := svc.DB.Exec(`
			INSERT INTO call_records (id, company_id, extension_id, user_id, contact_id, channel_id, call_direction, call_status, from_number, to_number, started_at, recording_url)
			VALUES ($1, $2, NULLIF($3, '')::uuid, $4, NULLIF($5, '')::uuid, $6, $7, $8, $9, $10, NOW(), NULLIF($11, ''))
			ON CONFLICT (id) DO UPDATE SET call_status = EXCLUDED.call_status, channel_id = EXCLUDED.channel_id, updated_at = NOW()
		`, body.CallID, companyID, extensionID, userID, contactID, body.ChannelID, body.Direction, body.Status, body.FromNumber, body.ToNumber, body.RecordingURL)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		svc.WSHub.BroadcastToCompany(companyID, "call_event", fiber.Map{
			"call_id": body.CallID, "direction": body.Direction, "status": body.Status,
			"from_number": body.FromNumber, "to_number": body.ToNumber,
		})
		return c.JSON(fiber.Map{"call_id": body.CallID})
	}
}

// POST /api/telephony/calls/log-end - Save SIP.js call completion in call_records
func EndWebRTCCallLog(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body struct {
			CallID       string `json:"call_id"`
			Status       string `json:"status"`
			Duration     int    `json:"duration"`
			RecordingURL string `json:"recording_url"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}
		if body.CallID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "call_id is required"})
		}
		if body.Status == "" {
			body.Status = "completed"
		}

		_, err := svc.DB.Exec(`
			UPDATE call_records
			SET call_status = $1,
			    ended_at = NOW(),
			    duration_seconds = CASE WHEN $2 > 0 THEN $2 ELSE EXTRACT(EPOCH FROM (NOW() - started_at))::int END,
			    recording_url = COALESCE(NULLIF($3, ''), recording_url),
			    updated_at = NOW()
			WHERE id = $4 AND company_id = $5
		`, body.Status, body.Duration, body.RecordingURL, body.CallID, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		svc.WSHub.BroadcastToCompany(companyID, "call_event", fiber.Map{
			"call_id": body.CallID, "status": body.Status, "duration": body.Duration,
		})
		return c.JSON(fiber.Map{"status": body.Status})
	}
}

func findExtensionID(svc *services.Container, companyID, extension string) string {
	if extension == "" {
		return ""
	}
	var id string
	_ = svc.DB.QueryRow("SELECT id FROM phone_extensions WHERE company_id = $1 AND extension_number = $2 LIMIT 1", companyID, extension).Scan(&id)
	return id
}

func findContactByPhone(svc *services.Container, companyID, phone string) string {
	normalized := onlyDialable(phone)
	if normalized == "" {
		return ""
	}
	var id string
	_ = svc.DB.QueryRow(`
		SELECT id FROM contacts
		WHERE company_id = $1 AND regexp_replace(COALESCE(phone, ''), '[^0-9+]', '', 'g') ILIKE $2
		LIMIT 1
	`, companyID, "%"+normalized+"%").Scan(&id)
	return id
}

func onlyDialable(value string) string {
	return strings.Map(func(r rune) rune {
		if (r >= '0' && r <= '9') || r == '+' {
			return r
		}
		return -1
	}, value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
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

		recordingRef := strings.TrimSpace(*recordingURL)
		if strings.HasPrefix(recordingRef, "http://") || strings.HasPrefix(recordingRef, "https://") {
			return c.Redirect(recordingRef, fiber.StatusTemporaryRedirect)
		}

		// Try to serve only from the CRM-mounted recordings storage. Do not expose
		// Asterisk physical paths such as /var/spool/asterisk/monitor to clients.
		filePath := filepath.Join("/app/storage/voip-recordings", filepath.Clean("/"+recordingRef))
		if _, err := os.Stat(filePath); err == nil {
			c.Set("Content-Type", "audio/wav")
			c.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s.wav\"", id))
			return c.SendFile(filePath)
		}

		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Recording file is not available in CRM storage"})
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
			filePath := filepath.Join("/app/storage/voip-recordings", filepath.Clean("/"+*recordingURL))
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

// ============================================
// Conference Calls
// ============================================

// POST /api/telephony/conference - Create a conference room
func CreateConference(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body struct {
			Name string `json:"name"`
		}
		c.BodyParser(&body)

		if body.Name == "" {
			body.Name = "Conference"
		}

		cfg, err := svc.Asterisk.GetAsteriskConfig(companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Telephony provider not configured"})
		}

		conferenceID := fmt.Sprintf("conf-%s", uuid.New().String()[:8])

		// Create a mixing bridge via ARI
		payload := map[string]interface{}{
			"type":     "mixing",
			"bridgeId": conferenceID,
			"name":     body.Name,
		}

		_, err = svc.Asterisk.ARIRequest(cfg, "POST", "/bridges", payload)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Failed to create conference: %v", err)})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"conference_id": conferenceID,
			"name":          body.Name,
			"status":        "created",
		})
	}
}

// POST /api/telephony/conference/:id/add - Add participant to conference
func AddToConference(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		conferenceID := c.Params("id")

		var body struct {
			ChannelID string `json:"channel_id"`
			Extension string `json:"extension"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		cfg, err := svc.Asterisk.GetAsteriskConfig(companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Telephony provider not configured"})
		}

		channelID := body.ChannelID

		// If extension provided instead of channel, originate a call to that extension
		if channelID == "" && body.Extension != "" {
			newChannelID := fmt.Sprintf("conf-leg-%s", uuid.New().String()[:8])
			payload := map[string]interface{}{
				"endpoint":  fmt.Sprintf("PJSIP/%s", body.Extension),
				"app":       "evocrm",
				"channelId": newChannelID,
				"variables": map[string]string{
					"COMPANY_ID":    companyID,
					"CONFERENCE_ID": conferenceID,
				},
			}
			_, err = svc.Asterisk.ARIRequest(cfg, "POST", "/channels", payload)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Failed to call participant: %v", err)})
			}
			channelID = newChannelID
		}

		if channelID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "channel_id or extension is required"})
		}

		// Add channel to the bridge
		_, err = svc.Asterisk.ARIRequest(cfg, "POST", fmt.Sprintf("/bridges/%s/addChannel", conferenceID), map[string]interface{}{
			"channel": channelID,
		})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Failed to add to conference: %v", err)})
		}

		return c.JSON(fiber.Map{
			"status":     "added",
			"channel_id": channelID,
		})
	}
}

// POST /api/telephony/conference/:id/remove - Remove participant from conference
func RemoveFromConference(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		conferenceID := c.Params("id")

		var body struct {
			ChannelID string `json:"channel_id"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if body.ChannelID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "channel_id is required"})
		}

		cfg, err := svc.Asterisk.GetAsteriskConfig(companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Telephony provider not configured"})
		}

		_, err = svc.Asterisk.ARIRequest(cfg, "POST", fmt.Sprintf("/bridges/%s/removeChannel", conferenceID), map[string]interface{}{
			"channel": body.ChannelID,
		})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Failed to remove from conference: %v", err)})
		}

		return c.JSON(fiber.Map{"status": "removed", "channel_id": body.ChannelID})
	}
}

// DELETE /api/telephony/conference/:id - End conference (destroy bridge)
func EndConference(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		conferenceID := c.Params("id")

		cfg, err := svc.Asterisk.GetAsteriskConfig(companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Telephony provider not configured"})
		}

		_, err = svc.Asterisk.ARIRequest(cfg, "DELETE", fmt.Sprintf("/bridges/%s", conferenceID), nil)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Failed to end conference: %v", err)})
		}

		return c.JSON(fiber.Map{"status": "ended", "conference_id": conferenceID})
	}
}

// ============================================
// Recording Control
// ============================================

// POST /api/telephony/recording/start - Start recording a channel
func StartRecording(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body struct {
			ChannelID string `json:"channel_id"`
			CallID    string `json:"call_id"`
			Format    string `json:"format"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if body.ChannelID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "channel_id is required"})
		}

		if body.Format == "" {
			body.Format = "wav"
		}

		cfg, err := svc.Asterisk.GetAsteriskConfig(companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Telephony provider not configured"})
		}

		recordingName := fmt.Sprintf("%s/%s_%s", companyID, body.ChannelID, time.Now().Format("20060102_150405"))

		payload := map[string]interface{}{
			"name":               recordingName,
			"format":             body.Format,
			"maxDurationSeconds": 3600,
			"ifExists":           "overwrite",
		}

		_, err = svc.Asterisk.ARIRequest(cfg, "POST", fmt.Sprintf("/channels/%s/record", body.ChannelID), payload)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Failed to start recording: %v", err)})
		}

		// Update call record with recording path
		if body.CallID != "" {
			recordingURL := fmt.Sprintf("%s/%s.%s", companyID, recordingName, body.Format)
			svc.DB.Exec("UPDATE call_records SET recording_url = $1 WHERE id = $2 AND company_id = $3", recordingURL, body.CallID, companyID)
		}

		return c.JSON(fiber.Map{
			"status":         "recording",
			"recording_name": recordingName,
		})
	}
}

// POST /api/telephony/recording/stop - Stop recording a channel
func StopRecording(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body struct {
			ChannelID     string `json:"channel_id"`
			RecordingName string `json:"recording_name"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if body.RecordingName == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "recording_name is required"})
		}

		cfg, err := svc.Asterisk.GetAsteriskConfig(companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Telephony provider not configured"})
		}

		_, err = svc.Asterisk.ARIRequest(cfg, "POST", fmt.Sprintf("/recordings/live/%s/stop", body.RecordingName), nil)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Failed to stop recording: %v", err)})
		}

		return c.JSON(fiber.Map{"status": "stopped", "recording_name": body.RecordingName})
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
