package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

func AuthLogin(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req services.LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		if req.Email == "" || req.Password == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email and password are required"})
		}

		resp, err := svc.Auth.Login(&req)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(resp)
	}
}

func AuthRegister(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req services.RegisterRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		if req.Email == "" || req.Password == "" || req.Name == "" || req.CompanyName == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "All fields are required"})
		}

		if len(req.Password) < 8 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password must be at least 8 characters"})
		}

		resp, err := svc.Auth.Register(&req)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(resp)
	}
}

func AuthRefresh(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		resp, err := svc.Auth.RefreshToken(body.RefreshToken)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(resp)
	}
}

func GetCurrentUser(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(string)
		user, err := svc.Auth.GetUserByID(userID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
		return c.JSON(user)
	}
}

func UpdateCurrentUser(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(string)
		companyID := c.Locals("company_id").(string)

		var body struct {
			Name  string `json:"name"`
			Phone string `json:"phone"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		body.Name = strings.TrimSpace(body.Name)
		body.Phone = strings.TrimSpace(body.Phone)
		if body.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Name is required"})
		}

		_, err := svc.DB.Exec(`
			UPDATE users SET name = $1, phone = NULLIF($2, ''), updated_at = NOW()
			WHERE id = $3 AND company_id = $4
		`, body.Name, body.Phone, userID, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		user, err := svc.Auth.GetUserByID(userID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
		return c.JSON(user)
	}
}

func UpdateCurrentUserStatus(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(string)
		companyID := c.Locals("company_id").(string)

		var body struct {
			Status string `json:"status"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		status := strings.ToLower(strings.TrimSpace(body.Status))
		if status != "online" && status != "offline" && status != "busy" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid status"})
		}
		isOnline := status != "offline"

		_, err := svc.DB.Exec(`
			UPDATE users SET availability_status = $1, is_online = $2, last_seen_at = NOW(), updated_at = NOW()
			WHERE id = $3 AND company_id = $4
		`, status, isOnline, userID, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		user, err := svc.Auth.GetUserByID(userID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
		return c.JSON(user)
	}
}

func UploadCurrentUserAvatar(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(string)
		companyID := c.Locals("company_id").(string)

		file, err := c.FormFile("avatar")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Avatar file is required"})
		}
		if file.Size > 3*1024*1024 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Avatar must be up to 3MB"})
		}

		ext := strings.ToLower(filepath.Ext(file.Filename))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Only JPG, PNG or WEBP images are allowed"})
		}

		dir := "/app/uploads/avatars"
		if err := os.MkdirAll(dir, 0755); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create upload directory"})
		}

		filename := fmt.Sprintf("%s-%d%s", userID, time.Now().UnixNano(), ext)
		dst := filepath.Join(dir, filename)
		if err := c.SaveFile(file, dst); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save avatar"})
		}

		avatarURL := "/uploads/avatars/" + filename
		_, err = svc.DB.Exec(`
			UPDATE users SET avatar_url = $1, updated_at = NOW()
			WHERE id = $2 AND company_id = $3
		`, avatarURL, userID, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		user, err := svc.Auth.GetUserByID(userID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
		return c.JSON(user)
	}
}
