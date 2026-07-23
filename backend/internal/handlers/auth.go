package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
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
			recordLoginEvent(svc, c, req.Email, false, err.Error(), "", "")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}
		recordLoginEvent(svc, c, req.Email, true, "", resp.User.ID, resp.User.CompanyID)
		_, _ = svc.DB.Exec(`
			UPDATE refresh_tokens SET ip_address = $1, user_agent = $2, last_used_at = NOW()
			WHERE id = (
				SELECT id FROM refresh_tokens WHERE user_id = $3 ORDER BY created_at DESC LIMIT 1
			)
		`, c.IP(), c.Get("User-Agent"), resp.User.ID)

		return c.JSON(resp)
	}
}

func recordLoginEvent(svc *services.Container, c *fiber.Ctx, email string, success bool, reason, userID, companyID string) {
	if userID == "" {
		_ = svc.DB.QueryRow(`
			SELECT COALESCE(id::text, ''), COALESCE(company_id::text, '')
			FROM users WHERE LOWER(email) = LOWER($1) ORDER BY created_at DESC LIMIT 1
		`, email).Scan(&userID, &companyID)
	}
	_, _ = svc.DB.Exec(`
		INSERT INTO login_events (id, company_id, user_id, email, success, failure_reason, ip_address, user_agent)
		VALUES ($1, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid, LOWER($4), $5, NULLIF($6, ''), $7, $8)
	`, uuid.New().String(), companyID, userID, email, success, reason, c.IP(), c.Get("User-Agent"))
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

func AuthForgotPassword(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req services.ForgotPasswordRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		req.Email = strings.TrimSpace(req.Email)
		if req.Email == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email is required"})
		}

		if err := svc.Auth.RequestPasswordReset(&req); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{
			"message": "If this email exists, a password reset link was sent",
		})
	}
}

func AuthResetPassword(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req services.ResetPasswordRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		if err := svc.Auth.ResetPassword(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Password updated"})
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
		_, err := svc.DB.Exec(`
			UPDATE users SET availability_status = $1, last_seen_at = NOW(), updated_at = NOW()
			WHERE id = $2 AND company_id = $3
		`, status, userID, companyID)
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

func UpdateCurrentUserPassword(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(string)
		companyID := c.Locals("company_id").(string)

		var body struct {
			CurrentPassword string `json:"current_password"`
			NewPassword     string `json:"new_password"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		if body.CurrentPassword == "" || body.NewPassword == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Current and new password are required"})
		}
		if len(body.NewPassword) < 8 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "New password must be at least 8 characters"})
		}

		var passwordHash string
		err := svc.DB.QueryRow(`
			SELECT password_hash FROM users WHERE id = $1 AND company_id = $2
		`, userID, companyID).Scan(&passwordHash)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
		if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(body.CurrentPassword)); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Current password is incorrect"})
		}

		newHash, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to hash password"})
		}

		_, err = svc.DB.Exec(`
			UPDATE users SET password_hash = $1, updated_at = NOW()
			WHERE id = $2 AND company_id = $3
		`, string(newHash), userID, companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Password updated"})
	}
}
