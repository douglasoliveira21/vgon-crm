package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"strings"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func HandleEmailDeliveryEvent(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		expected := svc.Config.EmailWebhookSecret
		provided := c.Get("X-Webhook-Secret")
		if expected == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) != 1 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Webhook não autorizado"})
		}
		var body struct {
			CompanyID       string                 `json:"company_id"`
			CampaignID      string                 `json:"campaign_id"`
			ContactID       string                 `json:"contact_id"`
			Email           string                 `json:"email"`
			Event           string                 `json:"event"`
			ProviderEventID string                 `json:"provider_event_id"`
			Details         map[string]interface{} `json:"details"`
		}
		if c.BodyParser(&body) != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Evento inválido"})
		}
		body.Email = normalizeSuppressionDestination("email", body.Email)
		body.Event = strings.ToLower(strings.TrimSpace(body.Event))
		valid := map[string]bool{"sent": true, "delivered": true, "opened": true, "clicked": true, "bounced": true, "complained": true}
		if body.CompanyID == "" || body.Email == "" || !valid[body.Event] {
			return c.Status(400).JSON(fiber.Map{"error": "Evento incompleto"})
		}
		details, _ := json.Marshal(body.Details)
		if body.ProviderEventID == "" {
			body.ProviderEventID = uuid.New().String()
		}
		_, err := svc.DB.Exec(`
			INSERT INTO campaign_delivery_events
				(id, company_id, campaign_id, contact_id, channel, destination, event_type,
				 provider_event_id, details)
			VALUES ($1, $2, NULLIF($3, '')::uuid, NULLIF($4, '')::uuid, 'email', $5, $6, $7, $8::jsonb)
			ON CONFLICT (company_id, provider_event_id, event_type) DO NOTHING
		`, uuid.New().String(), body.CompanyID, body.CampaignID, body.ContactID, body.Email,
			body.Event, body.ProviderEventID, details)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erro ao registrar evento"})
		}
		if body.Event == "bounced" || body.Event == "complained" {
			_, err = svc.DB.Exec(`
				INSERT INTO campaign_suppressions(id, company_id, channel, destination, reason, source)
				VALUES ($1, $2, 'email', $3, $4, 'provider_webhook')
				ON CONFLICT (company_id, channel, destination) DO UPDATE SET
					reason = EXCLUDED.reason, source = EXCLUDED.source
			`, uuid.New().String(), body.CompanyID, body.Email, body.Event)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "Erro ao aplicar supressão"})
			}
		}
		return c.SendStatus(204)
	}
}
