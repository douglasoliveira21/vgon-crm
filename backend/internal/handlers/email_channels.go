package handlers

import (
	"encoding/json"
	"strings"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type emailChannelRequest struct {
	Name         string `json:"name"`
	Provider     string `json:"provider"`
	IMAPHost     string `json:"imap_host"`
	IMAPPort     int    `json:"imap_port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	SMTPUseTLS   bool   `json:"smtp_use_tls"`
	Mailbox      string `json:"mailbox"`
	UseTLS       bool   `json:"use_tls"`
	MaxImport    int    `json:"max_import"`
}

func CreateEmailChannel(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body emailChannelRequest
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}
		if strings.TrimSpace(body.Name) == "" || strings.TrimSpace(body.IMAPHost) == "" || strings.TrimSpace(body.Username) == "" || strings.TrimSpace(body.Password) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nome, servidor IMAP, usuário e senha são obrigatórios"})
		}
		settings := emailSettingsFromRequest(body, nil)
		settingsJSON, _ := json.Marshal(settings)

		id := uuid.New().String()
		_, err := svc.DB.Exec(`
			INSERT INTO channels (id, company_id, name, type, status, settings, is_active)
			VALUES ($1, $2, $3, 'email', 'connected', $4, true)
		`, id, companyID, body.Name, settingsJSON)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		imported, syncErr := svc.Email.SyncChannel(companyID, id)
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"id":       id,
			"name":     body.Name,
			"type":     "email",
			"status":   "connected",
			"imported": imported,
			"warning":  errorString(syncErr),
		})
	}
}

func UpdateEmailChannel(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		channelID := c.Params("id")

		var currentRaw []byte
		err := svc.DB.QueryRow("SELECT settings FROM channels WHERE id = $1 AND company_id = $2 AND type = 'email'", channelID, companyID).Scan(&currentRaw)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Canal de e-mail não encontrado"})
		}
		current, _ := services.ParseEmailSettingsForHandler(currentRaw)

		var body emailChannelRequest
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}
		if strings.TrimSpace(body.Name) == "" || strings.TrimSpace(body.IMAPHost) == "" || strings.TrimSpace(body.Username) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nome, servidor IMAP e usuário são obrigatórios"})
		}
		settings := emailSettingsFromRequest(body, &current)
		settingsJSON, _ := json.Marshal(settings)

		_, err = svc.DB.Exec(`
			UPDATE channels
			SET name = $1, settings = $2, status = 'connected', updated_at = NOW()
			WHERE id = $3 AND company_id = $4 AND type = 'email'
		`, body.Name, settingsJSON, channelID, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "Canal de e-mail atualizado"})
	}
}

func DeleteEmailChannel(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		channelID := c.Params("id")

		// Remove channel reference from related tables (don't delete the data)
		svc.DB.Exec("UPDATE conversations SET channel_id = NULL WHERE channel_id = $1 AND company_id = $2", channelID, companyID)
		svc.DB.Exec("UPDATE whatsapp_instances SET channel_id = NULL WHERE channel_id = $1", channelID)
		svc.DB.Exec("UPDATE bot_flows SET channel_id = NULL WHERE channel_id = $1", channelID)
		svc.DB.Exec("UPDATE call_queues SET channel_id = NULL WHERE channel_id = $1", channelID)

		// Delete the channel
		_, err := svc.DB.Exec("DELETE FROM channels WHERE id = $1 AND company_id = $2 AND type = 'email'", channelID, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Falha ao remover canal: " + err.Error()})
		}
		return c.JSON(fiber.Map{"message": "Canal de e-mail removido"})
	}
}

func SyncEmailChannel(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		channelID := c.Params("id")

		imported, err := svc.Email.SyncChannel(companyID, channelID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"imported": imported})
	}
}

func emailSettingsFromRequest(body emailChannelRequest, current *services.EmailChannelSettings) services.EmailChannelSettings {
	settings := services.EmailChannelSettings{
		Provider:     strings.TrimSpace(body.Provider),
		IMAPHost:     strings.TrimSpace(body.IMAPHost),
		IMAPPort:     body.IMAPPort,
		Username:     strings.TrimSpace(body.Username),
		Password:     body.Password,
		SMTPHost:     strings.TrimSpace(body.SMTPHost),
		SMTPPort:     body.SMTPPort,
		SMTPUsername: strings.TrimSpace(body.SMTPUsername),
		SMTPPassword: body.SMTPPassword,
		SMTPUseTLS:   body.SMTPUseTLS,
		Mailbox:      strings.TrimSpace(body.Mailbox),
		UseTLS:       body.UseTLS,
		MaxImport:    body.MaxImport,
	}
	if settings.Mailbox == "" {
		settings.Mailbox = "INBOX"
	}
	if settings.Provider == "" {
		settings.Provider = "imap"
	}
	if settings.IMAPPort == 0 {
		if settings.UseTLS {
			settings.IMAPPort = 993
		} else {
			settings.IMAPPort = 143
		}
	}
	if settings.SMTPUsername == "" {
		settings.SMTPUsername = settings.Username
	}
	if settings.SMTPPassword == "" {
		settings.SMTPPassword = settings.Password
	}
	if settings.SMTPPort == 0 {
		if settings.SMTPUseTLS {
			settings.SMTPPort = 465
		} else {
			settings.SMTPPort = 587
		}
	}
	if settings.MaxImport <= 0 {
		settings.MaxImport = 500
	}
	if current != nil {
		settings.LastUID = current.LastUID
		if settings.Password == "" {
			settings.Password = current.Password
		}
		if settings.SMTPPassword == "" {
			settings.SMTPPassword = current.SMTPPassword
		}
	}
	return settings
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
