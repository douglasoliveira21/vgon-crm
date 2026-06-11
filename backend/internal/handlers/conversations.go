package handlers

import (
	"time"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func GetConversations(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		status := c.Query("status")
		teamID := c.Query("team_id")
		assignedTo := c.Query("assigned_to")
		limit := c.QueryInt("limit", 50)
		offset := c.QueryInt("offset", 0)

		conversations, err := svc.Message.GetConversations(companyID, status, assignedTo, teamID, limit, offset)
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

		conversations, err := svc.Message.GetConversations(companyID, "", userID, "", limit, offset)
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

		conversations, err := svc.Message.GetConversations(companyID, "", "", "", 1, 0)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		for _, conv := range conversations {
			if conv.ID == conversationID {
				return c.JSON(conv)
			}
		}

		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Conversation not found"})
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

func TransferConversation(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		conversationID := c.Params("id")

		var body struct {
			UserID *string `json:"user_id"`
			TeamID *string `json:"team_id"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if err := svc.Message.TransferConversation(conversationID, companyID, body.UserID, body.TeamID); err != nil {
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
			Content   string `json:"content"`
			IsPrivate bool   `json:"is_private"`
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

		// If not private, send via WhatsApp
		if !body.IsPrivate {
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

			if err == nil && phone != "" && instanceName != "" {
				externalID, _ := svc.Evolution.SendTextMessage(instanceName, phone, body.Content)
				if externalID != "" {
					svc.DB.Exec("UPDATE messages SET external_id = $1 WHERE id = $2", externalID, msg.ID)
				}
			}
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

		// Send via WhatsApp
		var externalID string
		mediaToSend := body.MediaURL
		if body.MediaBase64 != "" {
			mediaToSend = body.MediaBase64
		}

		if phone != "" && instanceName != "" && mediaToSend != "" {
			externalID, _ = svc.Evolution.SendMediaMessage(instanceName, phone, body.MediaType, mediaToSend, body.Caption, body.FileName)
		}

		// Save message to DB
		msgID := uuid.New().String()
		content := body.Caption
		if content == "" {
			content = body.FileName
		}

		svc.DB.Exec(`
			INSERT INTO messages (id, conversation_id, company_id, sender_type, sender_id, content, message_type, media_filename, external_id, status)
			VALUES ($1, $2, $3, 'user', $4, $5, $6, $7, $8, 'sent')
		`, msgID, conversationID, companyID, userID, content, body.MediaType, body.FileName, externalID)

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

		// Send audio via Evolution API
		var externalID string
		if phone != "" && instanceName != "" {
			if body.AudioBase64 != "" {
				externalID, _ = svc.Evolution.SendAudioBase64(instanceName, phone, body.AudioBase64)
			} else if body.AudioURL != "" {
				externalID, _ = svc.Evolution.SendAudioMessage(instanceName, phone, body.AudioURL)
			}
		}

		// Save message to DB
		msgID := uuid.New().String()
		svc.DB.Exec(`
			INSERT INTO messages (id, conversation_id, company_id, sender_type, sender_id, content, message_type, external_id, status)
			VALUES ($1, $2, $3, 'user', $4, '🎵 Áudio', 'audio', $5, 'sent')
		`, msgID, conversationID, companyID, userID, externalID)

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
