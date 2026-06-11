package handlers

import (
	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

func GetContacts(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		search := c.Query("search")
		limit := c.QueryInt("limit", 50)
		offset := c.QueryInt("offset", 0)

		contacts, total, err := svc.Contact.GetContacts(companyID, search, limit, offset)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{
			"contacts": contacts,
			"total":    total,
			"limit":    limit,
			"offset":   offset,
		})
	}
}

func GetContact(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		contactID := c.Params("id")

		contact, err := svc.Contact.GetContactByID(contactID, companyID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Contact not found"})
		}

		return c.JSON(contact)
	}
}

func CreateContact(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)

		var req services.CreateContactRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		contact, err := svc.Contact.CreateContact(companyID, &req)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(contact)
	}
}

func UpdateContact(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		contactID := c.Params("id")

		var req services.UpdateContactRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		contact, err := svc.Contact.UpdateContact(contactID, companyID, &req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(contact)
	}
}

func DeleteContact(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := c.Locals("company_id").(string)
		contactID := c.Params("id")

		if err := svc.Contact.DeleteContact(contactID, companyID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Contact deleted"})
	}
}

func AddContactTag(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		contactID := c.Params("id")

		var body struct {
			TagID string `json:"tag_id"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		if err := svc.Contact.AddTagToContact(contactID, body.TagID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Tag added"})
	}
}

func RemoveContactTag(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		contactID := c.Params("id")
		tagID := c.Params("tagId")

		if err := svc.Contact.RemoveTagFromContact(contactID, tagID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Tag removed"})
	}
}
