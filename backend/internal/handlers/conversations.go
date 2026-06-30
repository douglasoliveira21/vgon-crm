package handlers

import (
	"database/sql"
	"strings"
	"time"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func StartConversation(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		userID := c.Locals("user_id").(string)

		var body struct {
			Phone   string `json:"phone"`
			Message string `json:"message"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if body.Phone == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Phone is required"})
		}

		// Find or create contact
		var contactID string
		err := svc.DB.QueryRow("SELECT id FROM contacts WHERE company_id = $1 AND phone = $2", companyID, body.Phone).Scan(&contactID)
		if err != nil {
			contactID = uuid.New().String()
			svc.DB.Exec("INSERT INTO contacts (id, company_id, name, phone, origin) VALUES ($1, $2, $3, $4, 'manual')", contactID, companyID, body.Phone, body.Phone)
		}

		// Find or create conversation
		var conversationID string
		err = svc.DB.QueryRow("SELECT id FROM conversations WHERE company_id = $1 AND contact_id = $2 AND status != 'resolved' ORDER BY created_at DESC LIMIT 1", companyID, contactID).Scan(&conversationID)
		if err != nil {
			// Get first active channel
			var channelID string
			svc.DB.QueryRow("SELECT id FROM channels WHERE company_id = $1 AND status = 'connected' LIMIT 1", companyID).Scan(&channelID)

			conversationID = uuid.New().String()
			svc.DB.Exec("INSERT INTO conversations (id, company_id, contact_id, channel_id, assigned_to, status, last_message_at, customer_company_id) VALUES ($1, $2, $3, $4, $5, 'in_progress', NOW(), (SELECT customer_company_id FROM contacts WHERE id = $3))",
				conversationID, companyID, contactID, channelID, userID)
			applyConversationSLA(svc.DB, companyID, conversationID)
		}

		// If message provided, send it
		if body.Message != "" {
			// Get instance for sending
			var phone, instanceName string
			svc.DB.QueryRow(`
				SELECT co.phone, wi.instance_name
				FROM conversations conv
				JOIN contacts co ON conv.contact_id = co.id
				JOIN channels ch ON conv.channel_id = ch.id
				JOIN whatsapp_instances wi ON wi.channel_id = ch.id
				WHERE conv.id = $1
			`, conversationID).Scan(&phone, &instanceName)

			if phone != "" && instanceName != "" {
				externalID, _ := svc.Evolution.SendTextMessage(instanceName, phone, body.Message)
				msgID := uuid.New().String()
				svc.DB.Exec(`INSERT INTO messages (id, conversation_id, company_id, sender_type, sender_id, content, message_type, external_id, status) VALUES ($1, $2, $3, 'user', $4, $5, 'text', $6, 'sent')`,
					msgID, conversationID, companyID, userID, body.Message, externalID)
			}
		}

		return c.JSON(fiber.Map{"conversation_id": conversationID, "contact_id": contactID})
	}
}

func applyConversationSLA(db *sql.DB, companyID, conversationID string) {
	db.Exec(`
		UPDATE conversations conv
		SET customer_company_id = COALESCE(conv.customer_company_id, ct.customer_company_id),
		    first_response_due_at = CASE
		      WHEN cc.id IS NULL THEN NULL
		      ELSE conv.created_at + (cc.initial_response_sla_minutes || ' minutes')::interval
		    END,
		    resolution_due_at = CASE
		      WHEN cc.id IS NULL THEN NULL
		      ELSE conv.created_at + (cc.resolution_sla_minutes || ' minutes')::interval
		    END
		FROM contacts ct
		LEFT JOIN customer_companies cc ON COALESCE(conv.customer_company_id, ct.customer_company_id) = cc.id
		WHERE conv.id = $1 AND conv.company_id = $2 AND conv.contact_id = ct.id
	`, conversationID, companyID)
}

func GetConversations(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		status := c.Query("status")
		teamID := c.Query("team_id")
		channelID := c.Query("channel")
		assignedTo := c.Query("assigned_to")
		unassigned := c.Query("unassigned")
		limit := c.QueryInt("limit", 50)
		offset := c.QueryInt("offset", 0)

		conversations, err := svc.Message.GetConversations(companyID, status, assignedTo, teamID, channelID, unassigned == "true", limit, offset)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"conversations": conversations})
	}
}

func GetMyConversations(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		userID := c.Locals("user_id").(string)
		limit := c.QueryInt("limit", 50)
		offset := c.QueryInt("offset", 0)

		conversations, err := svc.Message.GetConversations(companyID, "", userID, "", "", false, limit, offset)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"conversations": conversations})
	}
}

func GetConversation(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		conversationID := c.Params("id")

		conversation, err := svc.Message.GetConversationByID(companyID, conversationID)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Conversation not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(conversation)
	}
}

func AssignConversation(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		conversationID := c.Params("id")

		var body struct {
			UserID string `json:"user_id"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if body.UserID == "" {
			body.UserID = c.Locals("user_id").(string)
		}

		if err := svc.Message.AssignConversation(conversationID, body.UserID, companyID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Conversation assigned"})
	}
}

func UnassignConversation(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		conversationID := c.Params("id")

		_, err := svc.DB.Exec(`UPDATE conversations SET assigned_to = NULL, status = 'open', updated_at = NOW() WHERE id = $1 AND company_id = $2`, conversationID, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Conversation unassigned"})
	}
}

func TransferConversation(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		conversationID := c.Params("id")

		var body struct {
			UserID    *string `json:"user_id"`
			TeamID    *string `json:"team_id"`
			ClearTeam bool    `json:"clear_team"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if err := svc.Message.TransferConversation(conversationID, companyID, body.UserID, body.TeamID, body.ClearTeam); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Conversation transferred"})
	}
}

func CloseConversation(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		conversationID := c.Params("id")

		if err := svc.Message.CloseConversation(conversationID, companyID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Conversation closed"})
	}
}

func LinkConversationCustomerCompany(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		conversationID := c.Params("id")
		var body struct {
			CustomerCompanyID *string `json:"customer_company_id"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}
		customerCompanyID := ""
		if body.CustomerCompanyID != nil {
			customerCompanyID = *body.CustomerCompanyID
		}
		_, err := svc.DB.Exec(`
			UPDATE contacts SET customer_company_id = NULLIF($1, '')::uuid, updated_at = NOW()
			WHERE id = (SELECT contact_id FROM conversations WHERE id = $2 AND company_id = $3)
			  AND company_id = $3
		`, customerCompanyID, conversationID, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		_, err = svc.DB.Exec(`
			UPDATE conversations SET customer_company_id = NULLIF($1, '')::uuid, updated_at = NOW()
			WHERE id = $2 AND company_id = $3
		`, customerCompanyID, conversationID, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		applyConversationSLA(svc.DB, companyID, conversationID)
		return c.JSON(fiber.Map{"message": "Empresa vinculada"})
	}
}

func ReopenConversation(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		conversationID := c.Params("id")

		if err := svc.Message.ReopenConversation(conversationID, companyID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Conversation reopened"})
	}
}

func MarkConversationRead(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		conversationID := c.Params("id")

		svc.DB.Exec("UPDATE conversations SET unread_count = 0 WHERE id = $1 AND company_id = $2", conversationID, companyID)

		return c.JSON(fiber.Map{"message": "Marked as read"})
	}
}

func MarkConversationUnread(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		conversationID := c.Params("id")

		_, err := svc.DB.Exec(`
			UPDATE conversations
			SET unread_count = (
				SELECT COUNT(*) FROM messages
				WHERE conversation_id = $1 AND company_id = $2 AND sender_type = 'contact' AND is_private = false
			), updated_at = NOW()
			WHERE id = $1 AND company_id = $2
		`, conversationID, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		var unreadCount int
		_ = svc.DB.QueryRow("SELECT unread_count FROM conversations WHERE id = $1 AND company_id = $2", conversationID, companyID).Scan(&unreadCount)

		return c.JSON(fiber.Map{"message": "Marked as unread", "unread_count": unreadCount})
	}
}

func GetConversationMessages(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		conversationID := c.Params("id")
		limit := c.QueryInt("limit", 50)
		offset := c.QueryInt("offset", 0)

		messages, err := svc.Message.GetConversationMessages(conversationID, companyID, limit, offset)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"messages": messages})
	}
}

func SendTextMessage(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		userID := c.Locals("user_id").(string)
		conversationID := c.Params("id")

		var body struct {
			Content   string  `json:"content"`
			IsPrivate bool    `json:"is_private"`
			ReplyToID *string `json:"reply_to_id"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if body.Content == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Content is required"})
		}

		req := &services.SendTextMessageRequest{
			ConversationID: conversationID,
			Content:        body.Content,
			IsPrivate:      body.IsPrivate,
		}

		msg, err := svc.Message.SaveAndSendMessage(companyID, userID, req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		svc.DB.Exec(`
			UPDATE conversations SET first_response_at = COALESCE(first_response_at, NOW())
			WHERE id = $1 AND company_id = $2
		`, conversationID, companyID)

		// If not private, send through the conversation channel
		if !body.IsPrivate {
			var channelType string
			_ = svc.DB.QueryRow(`
				SELECT COALESCE(ch.type, '')
				FROM conversations c
				LEFT JOIN channels ch ON c.channel_id = ch.id
				WHERE c.id = $1 AND c.company_id = $2
			`, conversationID, companyID).Scan(&channelType)

			if channelType == "email" {
				externalID, err := svc.Email.SendReply(companyID, conversationID, body.Content)
				if err != nil {
					svc.DB.Exec("UPDATE messages SET status = 'failed' WHERE id = $1 AND company_id = $2", msg.ID, companyID)
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
				}
				if externalID != "" {
					svc.DB.Exec("UPDATE messages SET external_id = $1, status = 'sent' WHERE id = $2 AND company_id = $3", externalID, msg.ID, companyID)
				}
			}

			// Get contact phone and instance
			var phone, instanceName string
			err := svc.DB.QueryRow(`
				SELECT co.phone, wi.instance_name
				FROM conversations c
				JOIN contacts co ON c.contact_id = co.id
				JOIN channels ch ON c.channel_id = ch.id
				JOIN whatsapp_instances wi ON wi.channel_id = ch.id
				WHERE c.id = $1
			`, conversationID).Scan(&phone, &instanceName)

			if channelType != "email" && err == nil && phone != "" && instanceName != "" {
				// Get agent name to prefix message
				var agentName string
				svc.DB.QueryRow("SELECT name FROM users WHERE id = $1", userID).Scan(&agentName)

				// Build message with agent name prefix
				messageToSend := body.Content
				if agentName != "" {
					messageToSend = "*" + agentName + ":*\n" + body.Content
				}

				// Get quoted message external_id if replying
				var quotedExternalID string
				if body.ReplyToID != nil && *body.ReplyToID != "" {
					svc.DB.QueryRow("SELECT COALESCE(external_id, '') FROM messages WHERE id = $1", *body.ReplyToID).Scan(&quotedExternalID)
				}

				externalID, _ := svc.Evolution.SendTextMessageWithQuote(instanceName, phone, messageToSend, quotedExternalID)
				if externalID != "" {
					svc.DB.Exec("UPDATE messages SET external_id = $1 WHERE id = $2", externalID, msg.ID)
				}
			}
		}

		// Save reply-to info if replying
		if body.ReplyToID != nil && *body.ReplyToID != "" {
			var replyContent, replySender string
			svc.DB.QueryRow(`
				SELECT COALESCE(m.content, '📎 Mídia'),
					   CASE WHEN m.sender_type = 'user' THEN COALESCE(u.name, 'Atendente') ELSE COALESCE(c.name, 'Contato') END
				FROM messages m
				LEFT JOIN users u ON m.sender_type = 'user' AND m.sender_id = u.id
				LEFT JOIN contacts c ON m.sender_type = 'contact' AND m.sender_id = c.id
				WHERE m.id = $1
			`, *body.ReplyToID).Scan(&replyContent, &replySender)

			svc.DB.Exec("UPDATE messages SET reply_to_id = $1, reply_to_content = $2, reply_to_sender = $3 WHERE id = $4",
				*body.ReplyToID, replyContent, replySender, msg.ID)
		}

		return c.Status(fiber.StatusCreated).JSON(msg)
	}
}

func SendMediaMessage(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		userID := c.Locals("user_id").(string)
		conversationID := c.Params("id")

		var body struct {
			MediaURL    string `json:"media_url"`
			MediaBase64 string `json:"media_base64"`
			MediaType   string `json:"media_type"`
			Caption     string `json:"caption"`
			FileName    string `json:"file_name"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}
		if err := svc.Message.PauseAutomationForConversation(conversationID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		// Get contact phone and instance
		var phone, instanceName string
		svc.DB.QueryRow(`
			SELECT co.phone, wi.instance_name
			FROM conversations c
			JOIN contacts co ON c.contact_id = co.id
			JOIN channels ch ON c.channel_id = ch.id
			JOIN whatsapp_instances wi ON wi.channel_id = ch.id
			WHERE c.id = $1
		`, conversationID).Scan(&phone, &instanceName)

		// Save file to disk and send via WhatsApp
		var savedFileName string
		var externalID string
		if body.MediaBase64 != "" {
			ext := services.GetExtensionFromBase64(body.MediaBase64)
			if ext == "" {
				ext = services.GetExtensionFromType(body.MediaType)
			}
			// Try to get extension from filename
			if body.FileName != "" {
				if dotIdx := strings.LastIndex(body.FileName, "."); dotIdx != -1 {
					ext = body.FileName[dotIdx:]
				}
			}
			var err error
			savedFileName, err = services.SaveBase64File(body.MediaBase64, ext)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save file"})
			}
		}

		if phone != "" && instanceName != "" && savedFileName != "" {
			// Send using public URL of the saved file
			publicURL := svc.Config.EvolutionWebhookURL
			baseURL := strings.TrimSuffix(publicURL, "/api/webhooks/evolution")
			mediaPublicURL := baseURL + "/uploads/" + savedFileName
			externalID, _ = svc.Evolution.SendMediaMessage(instanceName, phone, body.MediaType, mediaPublicURL, body.Caption, body.FileName)
		}

		// Save message to DB with local file URL
		msgID := uuid.New().String()
		content := body.Caption
		if content == "" {
			content = body.FileName
		}
		mediaURL := "/uploads/" + savedFileName

		svc.DB.Exec(`
			INSERT INTO messages (id, conversation_id, company_id, sender_type, sender_id, content, message_type, media_url, media_filename, external_id, status)
			VALUES ($1, $2, $3, 'user', $4, $5, $6, $7, $8, $9, 'sent')
		`, msgID, conversationID, companyID, userID, content, body.MediaType, mediaURL, body.FileName, externalID)

		svc.DB.Exec(`UPDATE conversations SET last_message_at = NOW(), last_message_preview = $1, updated_at = NOW() WHERE id = $2`,
			"📎 "+body.FileName, conversationID)

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"id":              msgID,
			"conversation_id": conversationID,
			"sender_type":     "user",
			"content":         content,
			"message_type":    body.MediaType,
			"media_filename":  body.FileName,
			"status":          "sent",
			"created_at":      time.Now(),
		})
	}
}

func SendAudioMessage(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		userID := c.Locals("user_id").(string)
		conversationID := c.Params("id")

		var body struct {
			AudioURL    string `json:"audio_url"`
			AudioBase64 string `json:"audio_base64"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}
		if err := svc.Message.PauseAutomationForConversation(conversationID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		// Save audio file to disk
		var savedFileName string
		if body.AudioBase64 != "" {
			ext := services.GetExtensionFromBase64(body.AudioBase64)
			if ext == "" {
				ext = ".ogg"
			}
			var err error
			savedFileName, err = services.SaveBase64File(body.AudioBase64, ext)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save audio"})
			}
		}

		// Get contact phone and instance
		var phone, instanceName string
		svc.DB.QueryRow(`
			SELECT co.phone, wi.instance_name
			FROM conversations c
			JOIN contacts co ON c.contact_id = co.id
			JOIN channels ch ON c.channel_id = ch.id
			JOIN whatsapp_instances wi ON wi.channel_id = ch.id
			WHERE c.id = $1
		`, conversationID).Scan(&phone, &instanceName)

		// Send audio via Evolution API using the public URL of the saved file
		var externalID string
		if phone != "" && instanceName != "" && savedFileName != "" {
			publicURL := svc.Config.EvolutionWebhookURL
			// Build public URL from the backend domain
			// Extract base URL (remove /api/webhooks/evolution)
			baseURL := strings.TrimSuffix(publicURL, "/api/webhooks/evolution")
			audioPublicURL := baseURL + "/uploads/" + savedFileName
			externalID, _ = svc.Evolution.SendAudioMessage(instanceName, phone, audioPublicURL)
		}

		// Save message to DB with local file URL
		msgID := uuid.New().String()
		mediaURL := "/uploads/" + savedFileName
		svc.DB.Exec(`
			INSERT INTO messages (id, conversation_id, company_id, sender_type, sender_id, content, message_type, media_url, external_id, status)
			VALUES ($1, $2, $3, 'user', $4, '🎵 Áudio', 'audio', $5, $6, 'sent')
		`, msgID, conversationID, companyID, userID, mediaURL, externalID)

		// Update conversation
		svc.DB.Exec(`UPDATE conversations SET last_message_at = NOW(), last_message_preview = '🎵 Áudio', updated_at = NOW() WHERE id = $1`, conversationID)

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"id":              msgID,
			"conversation_id": conversationID,
			"sender_type":     "user",
			"content":         "🎵 Áudio",
			"message_type":    "audio",
			"status":          "sent",
			"created_at":      time.Now(),
		})
	}
}

func DeleteMessage(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		msgID := c.Params("msgId")

		_, err := svc.DB.Exec("DELETE FROM messages WHERE id = $1 AND company_id = $2", msgID, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete message"})
		}

		return c.JSON(fiber.Map{"message": "Message deleted"})
	}
}
