package handlers

import (
	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func SaveTelephonyProvider(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body struct {
			Name             string `json:"name"`
			ProviderType     string `json:"provider_type"`
			SipHost          string `json:"sip_host"`
			SipPort          int    `json:"sip_port"`
			SipUser          string `json:"sip_user"`
			SipPassword      string `json:"sip_password"`
			SipDomain        string `json:"sip_domain"`
			Transport        string `json:"transport"`
			CallerID         string `json:"caller_id"`
			StunServer       string `json:"stun_server"`
			RecordingEnabled bool   `json:"recording_enabled"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		// Check if provider already exists for this company
		var existingID string
		err := svc.DB.QueryRow("SELECT id FROM telephony_providers WHERE company_id = $1 LIMIT 1", companyID).Scan(&existingID)

		if err == nil {
			// Update existing
			svc.DB.Exec(`
				UPDATE telephony_providers SET name=$1, provider_type=$2, sip_host=$3, sip_port=$4, sip_user=$5, sip_password=$6, sip_domain=$7, transport=$8, caller_id=$9, stun_server=$10, recording_enabled=$11, updated_at=NOW()
				WHERE id=$12
			`, body.Name, body.ProviderType, body.SipHost, body.SipPort, body.SipUser, body.SipPassword, body.SipDomain, body.Transport, body.CallerID, body.StunServer, body.RecordingEnabled, existingID)
			return c.JSON(fiber.Map{"message": "Provider updated", "id": existingID})
		}

		// Create new
		id := uuid.New().String()
		svc.DB.Exec(`
			INSERT INTO telephony_providers (id, company_id, name, provider_type, sip_host, sip_port, sip_user, sip_password, sip_domain, transport, caller_id, stun_server, recording_enabled)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`, id, companyID, body.Name, body.ProviderType, body.SipHost, body.SipPort, body.SipUser, body.SipPassword, body.SipDomain, body.Transport, body.CallerID, body.StunServer, body.RecordingEnabled)

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Provider created", "id": id})
	}
}

func GetTelephonyProvider(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var id, name, providerType, sipHost, sipUser, sipDomain, transport, callerID, stunServer, status string
		var sipPort int
		var recordingEnabled bool

		err := svc.DB.QueryRow(`
			SELECT id, name, provider_type, sip_host, sip_port, sip_user, sip_domain, transport, caller_id, COALESCE(stun_server, ''), recording_enabled, status
			FROM telephony_providers WHERE company_id = $1 LIMIT 1
		`, companyID).Scan(&id, &name, &providerType, &sipHost, &sipPort, &sipUser, &sipDomain, &transport, &callerID, &stunServer, &recordingEnabled, &status)

		if err != nil {
			return c.JSON(fiber.Map{"provider": nil})
		}

		return c.JSON(fiber.Map{"provider": fiber.Map{
			"id": id, "name": name, "provider_type": providerType,
			"sip_host": sipHost, "sip_port": sipPort, "sip_user": sipUser,
			"sip_domain": sipDomain, "transport": transport, "caller_id": callerID,
			"stun_server": stunServer, "recording_enabled": recordingEnabled, "status": status,
		}})
	}
}
