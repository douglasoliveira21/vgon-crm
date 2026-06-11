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

func GetExtensions(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		rows, err := svc.DB.Query(`
			SELECT id, extension_number, display_name, status, can_call_external
			FROM phone_extensions WHERE company_id = $1 ORDER BY extension_number
		`, companyID)
		if err != nil {
			return c.JSON(fiber.Map{"extensions": []interface{}{}})
		}
		defer rows.Close()

		var extensions []map[string]interface{}
		for rows.Next() {
			var id, number, name, status string
			var canCallExt bool
			rows.Scan(&id, &number, &name, &status, &canCallExt)
			extensions = append(extensions, map[string]interface{}{
				"id": id, "extension_number": number, "display_name": name,
				"status": status, "can_call_external": canCallExt,
			})
		}
		return c.JSON(fiber.Map{"extensions": extensions})
	}
}

func CreateExtension(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		var body struct {
			DisplayName       string `json:"display_name"`
			ExtensionNumber   string `json:"extension_number"`
			ExtensionPassword string `json:"extension_password"`
			CanCallExternal   bool   `json:"can_call_external"`
			CanReceiveCalls   bool   `json:"can_receive_calls"`
			CanTransfer       bool   `json:"can_transfer"`
			CanAccessRec      bool   `json:"can_access_recordings"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		id := uuid.New().String()
		_, err := svc.DB.Exec(`
			INSERT INTO phone_extensions (id, company_id, display_name, extension_number, extension_password, can_call_external, can_receive_calls, can_transfer, can_access_recordings)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, id, companyID, body.DisplayName, body.ExtensionNumber, body.ExtensionPassword, body.CanCallExternal, body.CanReceiveCalls, body.CanTransfer, body.CanAccessRec)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
	}
}

func DeleteExtension(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		id := c.Params("id")
		svc.DB.Exec("DELETE FROM phone_extensions WHERE id = $1 AND company_id = $2", id, companyID)
		return c.JSON(fiber.Map{"message": "Extension deleted"})
	}
}

func GetQueues(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		rows, err := svc.DB.Query("SELECT id, name, strategy FROM call_queues WHERE company_id = $1 ORDER BY name", companyID)
		if err != nil {
			return c.JSON(fiber.Map{"queues": []interface{}{}})
		}
		defer rows.Close()

		var queues []map[string]interface{}
		for rows.Next() {
			var id, name, strategy string
			rows.Scan(&id, &name, &strategy)
			queues = append(queues, map[string]interface{}{"id": id, "name": name, "strategy": strategy})
		}
		return c.JSON(fiber.Map{"queues": queues})
	}
}

func CreateQueue(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		var body struct {
			Name        string `json:"name"`
			Strategy    string `json:"strategy"`
			MaxWaitTime int    `json:"max_wait_time"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}
		if body.Strategy == "" {
			body.Strategy = "ringall"
		}
		if body.MaxWaitTime == 0 {
			body.MaxWaitTime = 120
		}

		id := uuid.New().String()
		svc.DB.Exec("INSERT INTO call_queues (id, company_id, name, strategy, max_wait_time) VALUES ($1, $2, $3, $4, $5)",
			id, companyID, body.Name, body.Strategy, body.MaxWaitTime)
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
	}
}

func DeleteQueue(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		id := c.Params("id")
		svc.DB.Exec("DELETE FROM call_queues WHERE id = $1 AND company_id = $2", id, companyID)
		return c.JSON(fiber.Map{"message": "Queue deleted"})
	}
}
