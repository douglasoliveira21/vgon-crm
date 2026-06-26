package handlers

import (
	"encoding/json"
	"time"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// IVROption represents a single menu option in the IVR
type IVROption struct {
	Digit       string `json:"digit"`
	Label       string `json:"label"`
	Action      string `json:"action"`      // extension, queue, external, voicemail, ivr
	Destination string `json:"destination"` // extension number, queue name, phone, or IVR id
}

// IVR represents a complete IVR/URA configuration
type IVR struct {
	ID                 string      `json:"id"`
	CompanyID          string      `json:"company_id"`
	Name               string      `json:"name"`
	WelcomeMessage     string      `json:"welcome_message"`
	BusinessHoursStart string      `json:"business_hours_start"`
	BusinessHoursEnd   string      `json:"business_hours_end"`
	OffHoursMessage    string      `json:"off_hours_message"`
	TimeoutMessage     string      `json:"timeout_message"`
	InvalidMessage     string      `json:"invalid_message"`
	MaxRetries         int         `json:"max_retries"`
	Options            []IVROption `json:"options"`
	IsActive           bool        `json:"is_active"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}

// GET /api/telephony/ivr - List IVRs for company
func GetIVRList(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		rows, err := svc.DB.Query(`
			SELECT id, name, welcome_message, business_hours_start, business_hours_end,
				   off_hours_message, timeout_message, invalid_message, max_retries,
				   options, is_active, created_at, updated_at
			FROM telephony_ivr
			WHERE company_id = $1
			ORDER BY created_at DESC
		`, companyID)
		if err != nil {
			return c.JSON(fiber.Map{"ivrs": []interface{}{}})
		}
		defer rows.Close()

		var ivrs []map[string]interface{}
		for rows.Next() {
			var id, name, welcomeMsg, hoursStart, hoursEnd, offHoursMsg, timeoutMsg, invalidMsg string
			var maxRetries int
			var optionsJSON []byte
			var isActive bool
			var createdAt, updatedAt time.Time

			err := rows.Scan(&id, &name, &welcomeMsg, &hoursStart, &hoursEnd,
				&offHoursMsg, &timeoutMsg, &invalidMsg, &maxRetries,
				&optionsJSON, &isActive, &createdAt, &updatedAt)
			if err != nil {
				continue
			}

			var options []IVROption
			json.Unmarshal(optionsJSON, &options)

			ivrs = append(ivrs, map[string]interface{}{
				"id":                   id,
				"name":                 name,
				"welcome_message":      welcomeMsg,
				"business_hours_start": hoursStart,
				"business_hours_end":   hoursEnd,
				"off_hours_message":    offHoursMsg,
				"timeout_message":      timeoutMsg,
				"invalid_message":      invalidMsg,
				"max_retries":          maxRetries,
				"options":              options,
				"is_active":            isActive,
				"created_at":           createdAt,
				"updated_at":           updatedAt,
			})
		}

		if ivrs == nil {
			ivrs = []map[string]interface{}{}
		}

		return c.JSON(fiber.Map{"ivrs": ivrs})
	}
}

// POST /api/telephony/ivr - Create IVR
func CreateIVR(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var body struct {
			Name               string      `json:"name"`
			WelcomeMessage     string      `json:"welcome_message"`
			BusinessHoursStart string      `json:"business_hours_start"`
			BusinessHoursEnd   string      `json:"business_hours_end"`
			OffHoursMessage    string      `json:"off_hours_message"`
			TimeoutMessage     string      `json:"timeout_message"`
			InvalidMessage     string      `json:"invalid_message"`
			MaxRetries         int         `json:"max_retries"`
			Options            []IVROption `json:"options"`
			IsActive           bool        `json:"is_active"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if body.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Name is required"})
		}

		// Defaults
		if body.WelcomeMessage == "" {
			body.WelcomeMessage = "Bem-vindo. Escolha uma opção."
		}
		if body.BusinessHoursStart == "" {
			body.BusinessHoursStart = "08:00"
		}
		if body.BusinessHoursEnd == "" {
			body.BusinessHoursEnd = "18:00"
		}
		if body.OffHoursMessage == "" {
			body.OffHoursMessage = "Estamos fora do horário de atendimento."
		}
		if body.TimeoutMessage == "" {
			body.TimeoutMessage = "Não entendi sua opção. Tente novamente."
		}
		if body.InvalidMessage == "" {
			body.InvalidMessage = "Opção inválida. Tente novamente."
		}
		if body.MaxRetries == 0 {
			body.MaxRetries = 3
		}
		if body.Options == nil {
			body.Options = []IVROption{}
		}

		id := uuid.New().String()
		optionsJSON, _ := json.Marshal(body.Options)

		_, err := svc.DB.Exec(`
			INSERT INTO telephony_ivr (id, company_id, name, welcome_message, business_hours_start, business_hours_end, off_hours_message, timeout_message, invalid_message, max_retries, options, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`, id, companyID, body.Name, body.WelcomeMessage, body.BusinessHoursStart, body.BusinessHoursEnd,
			body.OffHoursMessage, body.TimeoutMessage, body.InvalidMessage, body.MaxRetries, optionsJSON, body.IsActive)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id, "message": "IVR created"})
	}
}

// PUT /api/telephony/ivr/:id - Update IVR
func UpdateIVR(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		id := c.Params("id")

		var body struct {
			Name               string      `json:"name"`
			WelcomeMessage     string      `json:"welcome_message"`
			BusinessHoursStart string      `json:"business_hours_start"`
			BusinessHoursEnd   string      `json:"business_hours_end"`
			OffHoursMessage    string      `json:"off_hours_message"`
			TimeoutMessage     string      `json:"timeout_message"`
			InvalidMessage     string      `json:"invalid_message"`
			MaxRetries         int         `json:"max_retries"`
			Options            []IVROption `json:"options"`
			IsActive           bool        `json:"is_active"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if body.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Name is required"})
		}

		if body.Options == nil {
			body.Options = []IVROption{}
		}

		optionsJSON, _ := json.Marshal(body.Options)

		result, err := svc.DB.Exec(`
			UPDATE telephony_ivr
			SET name = $1, welcome_message = $2, business_hours_start = $3, business_hours_end = $4,
				off_hours_message = $5, timeout_message = $6, invalid_message = $7, max_retries = $8,
				options = $9, is_active = $10, updated_at = NOW()
			WHERE id = $11 AND company_id = $12
		`, body.Name, body.WelcomeMessage, body.BusinessHoursStart, body.BusinessHoursEnd,
			body.OffHoursMessage, body.TimeoutMessage, body.InvalidMessage, body.MaxRetries,
			optionsJSON, body.IsActive, id, companyID)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "IVR not found"})
		}

		return c.JSON(fiber.Map{"message": "IVR updated"})
	}
}

// DELETE /api/telephony/ivr/:id - Delete IVR
func DeleteIVR(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		id := c.Params("id")

		result, err := svc.DB.Exec("DELETE FROM telephony_ivr WHERE id = $1 AND company_id = $2", id, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "IVR not found"})
		}

		return c.JSON(fiber.Map{"message": "IVR deleted"})
	}
}
