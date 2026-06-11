package handlers

import (
	"encoding/json"
	"log"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

func GetWhatsAppInstances(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		instances, err := svc.Evolution.GetInstances(companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"instances": instances})
	}
}

func CreateWhatsAppInstance(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body struct {
			InstanceName string `json:"instance_name"`
			ChannelName  string `json:"channel_name"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		if body.InstanceName == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Instance name is required"})
		}

		if body.ChannelName == "" {
			body.ChannelName = "WhatsApp - " + body.InstanceName
		}

		req := &services.CreateInstanceRequest{
			InstanceName: body.InstanceName,
			CompanyID:    companyID,
			ChannelName:  body.ChannelName,
		}

		instance, err := svc.Evolution.CreateInstance(req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(instance)
	}
}

func GetWhatsAppQRCode(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		instanceID := c.Params("id")

		// Get instance name
		var instanceName string
		err := svc.DB.QueryRow("SELECT instance_name FROM whatsapp_instances WHERE id = $1 AND company_id = $2", instanceID, companyID).Scan(&instanceName)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Instance not found"})
		}

		qrcode, err := svc.Evolution.GetQRCode(instanceName)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"qrcode": qrcode})
	}
}

func GetWhatsAppStatus(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		instanceID := c.Params("id")

		var instanceName string
		err := svc.DB.QueryRow("SELECT instance_name FROM whatsapp_instances WHERE id = $1 AND company_id = $2", instanceID, companyID).Scan(&instanceName)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Instance not found"})
		}

		status, err := svc.Evolution.GetConnectionStatus(instanceName)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"status": status, "instance_name": instanceName})
	}
}

func DisconnectWhatsAppInstance(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		instanceID := c.Params("id")

		var instanceName string
		err := svc.DB.QueryRow("SELECT instance_name FROM whatsapp_instances WHERE id = $1 AND company_id = $2", instanceID, companyID).Scan(&instanceName)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Instance not found"})
		}

		if err := svc.Evolution.DisconnectInstance(instanceName); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Instance disconnected"})
	}
}

func DeleteWhatsAppInstance(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		instanceID := c.Params("id")

		var instanceName string
		err := svc.DB.QueryRow("SELECT instance_name FROM whatsapp_instances WHERE id = $1 AND company_id = $2", instanceID, companyID).Scan(&instanceName)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Instance not found"})
		}

		if err := svc.Evolution.DeleteInstance(instanceName); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Instance deleted"})
	}
}

func SyncWhatsAppContacts(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		instanceID := c.Params("id")

		var instanceName string
		err := svc.DB.QueryRow("SELECT instance_name FROM whatsapp_instances WHERE id = $1 AND company_id = $2", instanceID, companyID).Scan(&instanceName)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Instance not found"})
		}

		count, err := svc.Evolution.SyncContacts(instanceName, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		// Sync photos in background
		go svc.Evolution.SyncAllContactPhotos(instanceName, companyID)

		return c.JSON(fiber.Map{"message": "Contacts synced", "count": count})
	}
}

func SyncWhatsAppPhotos(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		instanceID := c.Params("id")

		var instanceName string
		err := svc.DB.QueryRow("SELECT instance_name FROM whatsapp_instances WHERE id = $1 AND company_id = $2", instanceID, companyID).Scan(&instanceName)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Instance not found"})
		}

		go svc.Evolution.SyncAllContactPhotos(instanceName, companyID)

		return c.JSON(fiber.Map{"message": "Photo sync started in background"})
	}
}

func HandleEvolutionWebhook(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		instanceName := c.Params("instanceName")

		var event map[string]interface{}
		if err := json.Unmarshal(c.Body(), &event); err != nil {
			log.Printf("[WEBHOOK] Failed to parse payload from %s: %v", instanceName, err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
		}

		log.Printf("[WEBHOOK] Received event from instance '%s': %v", instanceName, getEventType(event))

		// Process webhook asynchronously
		go svc.Evolution.HandleWebhook(instanceName, event)

		return c.JSON(fiber.Map{"status": "ok"})
	}
}

func getEventType(event map[string]interface{}) string {
	if e, ok := event["event"].(string); ok {
		return e
	}
	// v2.4.0 may use different field names
	for key := range event {
		return "keys:" + key
	}
	return "unknown"
}
