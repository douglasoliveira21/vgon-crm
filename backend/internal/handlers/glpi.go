package handlers

import (
	"strconv"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

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
