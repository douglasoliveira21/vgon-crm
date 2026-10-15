package handlers

import (
	"log"
	"strconv"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

// GLPITenantGuard restricts the GLPI routes to the single company that owns
// the integration.
//
// GLPI is reached with one global account from the environment, and the
// handlers address tickets by sequential numeric id, so without this any
// tenant could read (and open) tickets in that one helpdesk just by counting
// upwards — and since signup is open and a new company's owner is created as
// an admin with "*", that meant anyone who registered.
func GLPITenantGuard(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID, _ := c.Locals("company_id").(string)

		if svc.Config.GLPICompanyID != "" {
			if companyID == svc.Config.GLPICompanyID {
				return c.Next()
			}
			log.Printf("[GLPI] Blocked access from company %s (integration belongs to %s)", companyID, svc.Config.GLPICompanyID)
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Integração GLPI não disponível para esta empresa"})
		}

		// Not configured: fall back to super admins rather than everyone.
		userID, _ := c.Locals("user_id").(string)
		var isSuperAdmin bool
		if err := svc.DB.QueryRow("SELECT COALESCE(is_super_admin, false) FROM users WHERE id = $1", userID).Scan(&isSuperAdmin); err != nil || !isSuperAdmin {
			log.Printf("[GLPI] Blocked access from company %s: GLPI_COMPANY_ID is not set", companyID)
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Integração GLPI não liberada para esta empresa"})
		}
		return c.Next()
	}
}

// POST /api/glpi/tickets - Create a GLPI ticket
func GLPICreateTicket(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if svc.Config.GLPIBaseURL == "" || svc.Config.GLPIUserToken == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "GLPI não configurado. Configure GLPI_BASE_URL, GLPI_APP_TOKEN e GLPI_USER_TOKEN."})
		}

		var body struct {
			Title      string `json:"title"`
			Content    string `json:"content"`
			EntityID   int    `json:"entity_id"`
			TicketType int    `json:"ticket_type"` // 1=Incident, 2=Request
			Priority   int    `json:"priority"`    // 1-6
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if body.Title == "" || body.Content == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Title and content are required"})
		}
		if body.TicketType == 0 {
			body.TicketType = 1 // Default: Incident
		}
		if body.Priority == 0 {
			body.Priority = 3 // Default: Medium
		}

		// Init GLPI session
		sessionToken, err := svc.GLPI.InitSession(svc.Config.GLPIUserToken)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Falha ao conectar no GLPI: " + err.Error()})
		}
		defer svc.GLPI.KillSession(sessionToken)

		// Create ticket
		ticket, err := svc.GLPI.CreateTicket(sessionToken, body.Title, body.Content, body.EntityID, body.TicketType, body.Priority)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Falha ao criar chamado: " + err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"ticket_id":  ticket.ID,
			"title":      ticket.Name,
			"status":     ticket.Status,
			"entity_id":  ticket.EntityID,
		})
	}
}

// GET /api/glpi/tickets/:id - Get a GLPI ticket
func GLPIGetTicket(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if svc.Config.GLPIBaseURL == "" || svc.Config.GLPIUserToken == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "GLPI não configurado"})
		}

		ticketID, err := strconv.Atoi(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ticket ID"})
		}

		sessionToken, err := svc.GLPI.InitSession(svc.Config.GLPIUserToken)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Falha ao conectar no GLPI: " + err.Error()})
		}
		defer svc.GLPI.KillSession(sessionToken)

		ticket, err := svc.GLPI.GetTicket(sessionToken, ticketID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Falha ao buscar chamado: " + err.Error()})
		}

		statusLabels := map[int]string{
			1: "Novo",
			2: "Em atendimento (atribuído)",
			3: "Em atendimento (planejado)",
			4: "Pendente",
			5: "Solucionado",
			6: "Fechado",
		}

		return c.JSON(fiber.Map{
			"id":           ticket.ID,
			"title":        ticket.Name,
			"content":      ticket.Content,
			"status":       ticket.Status,
			"status_label": statusLabels[ticket.Status],
			"entity_id":    ticket.EntityID,
			"date":         ticket.Date,
			"date_mod":     ticket.DateMod,
			"type":         ticket.Type,
			"priority":     ticket.Priority,
		})
	}
}

// GET /api/glpi/entities - List GLPI entities
func GLPIGetEntities(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if svc.Config.GLPIBaseURL == "" || svc.Config.GLPIUserToken == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "GLPI não configurado"})
		}

		sessionToken, err := svc.GLPI.InitSession(svc.Config.GLPIUserToken)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Falha ao conectar no GLPI: " + err.Error()})
		}
		defer svc.GLPI.KillSession(sessionToken)

		entities, err := svc.GLPI.GetEntities(sessionToken)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Falha ao buscar entidades: " + err.Error()})
		}

		return c.JSON(fiber.Map{"entities": entities})
	}
}

// GET /api/glpi/entities/:id - Get a specific GLPI entity
func GLPIGetEntity(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if svc.Config.GLPIBaseURL == "" || svc.Config.GLPIUserToken == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "GLPI não configurado"})
		}

		entityID, err := strconv.Atoi(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid entity ID"})
		}

		sessionToken, err := svc.GLPI.InitSession(svc.Config.GLPIUserToken)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Falha ao conectar no GLPI: " + err.Error()})
		}
		defer svc.GLPI.KillSession(sessionToken)

		entity, err := svc.GLPI.GetEntity(sessionToken, entityID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Falha ao buscar entidade: " + err.Error()})
		}

		return c.JSON(fiber.Map{
			"id":            entity.ID,
			"name":          entity.Name,
			"complete_name": entity.CompleteName,
		})
	}
}


// POST /api/glpi/flow/start - Start a GLPI flow for a conversation
func GLPIStartFlow(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body struct {
			ConversationID string `json:"conversation_id"`
			Mode           string `json:"mode"` // "open_ticket" or "check_status"
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if body.ConversationID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "conversation_id is required"})
		}
		if body.Mode == "" {
			body.Mode = "open_ticket"
		}

		// Get conversation details
		var contactID, phone, instanceName string
		svc.DB.QueryRow(`
			SELECT c.contact_id, co.phone, COALESCE(wi.instance_name, '')
			FROM conversations c
			JOIN contacts co ON c.contact_id = co.id
			LEFT JOIN channels ch ON c.channel_id = ch.id
			LEFT JOIN whatsapp_instances wi ON wi.channel_id = ch.id
			WHERE c.id = $1 AND c.company_id = $2
		`, body.ConversationID, companyID).Scan(&contactID, &phone, &instanceName)

		if phone == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Conversation not found or no phone"})
		}

		// Start GLPI flow
		go svc.GLPIFlow.StartGLPIFlow(companyID, body.ConversationID, contactID, instanceName, phone, body.Mode)

		return c.JSON(fiber.Map{"message": "GLPI flow started", "mode": body.Mode})
	}
}
