package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/evocrm/backend/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

type Claims struct {
	UserID    string `json:"user_id"`
	CompanyID string `json:"company_id"`
	RoleSlug  string `json:"role_slug"`
	Email     string `json:"email"`
	jwt.RegisteredClaims
}

func AuthMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authorization header required",
			})
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization format",
			})
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		// Set user context
		c.Locals("user_id", claims.UserID)
		c.Locals("company_id", claims.CompanyID)
		c.Locals("role_slug", claims.RoleSlug)
		c.Locals("email", claims.Email)

		return c.Next()
	}
}

// GenerateTokens creates access and refresh tokens
func GenerateTokens(userID, companyID, roleSlug, email string, cfg *config.Config) (string, string, error) {
	// Access token
	claims := &Claims{
		UserID:    userID,
		CompanyID: companyID,
		RoleSlug:  roleSlug,
		Email:     email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.JWTExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessTokenString, err := accessToken.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return "", "", err
	}

	// Refresh token
	refreshClaims := &Claims{
		UserID:    userID,
		CompanyID: companyID,
		RoleSlug:  roleSlug,
		Email:     email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.JWTRefreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(cfg.JWTRefreshSecret))
	if err != nil {
		return "", "", err
	}

	return accessTokenString, refreshTokenString, nil
}

// SuperAdminMiddleware checks if the authenticated user has is_super_admin = true
func SuperAdminMiddleware(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(string)

		var isSuperAdmin bool
		err := db.QueryRow("SELECT COALESCE(is_super_admin, false) FROM users WHERE id = $1", userID).Scan(&isSuperAdmin)
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied",
			})
		}

		if !isSuperAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Super admin access required",
			})
		}

		return c.Next()
	}
}

// ActiveTenantMiddleware blocks active access tokens after a tenant or user is suspended.
func ActiveTenantMiddleware(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, _ := c.Locals("user_id").(string)
		companyID, _ := c.Locals("company_id").(string)
		var allowed bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM users u
				JOIN companies co ON co.id = u.company_id
				WHERE u.id = $1 AND u.company_id = $2 AND u.is_active = true AND co.is_active = true
			)
		`, userID, companyID).Scan(&allowed)
		if err != nil || !allowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Tenant ou usuário suspenso"})
		}
		return c.Next()
	}
}

// RBACMiddleware checks if user has required permission
func RBACMiddleware(requiredPermissions ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleSlug := c.Locals("role_slug").(string)

		// Super admin has all permissions
		if roleSlug == "super-admin" || roleSlug == "admin" {
			return c.Next()
		}

		// For now, allow all authenticated users
		// TODO: Implement granular permission checking from DB
		return c.Next()
	}
}

// DenyRoles blocks authenticated users whose role is explicitly denied.
func DenyRoles(deniedRoles ...string) fiber.Handler {
	denied := make(map[string]struct{}, len(deniedRoles))
	for _, role := range deniedRoles {
		denied[role] = struct{}{}
	}
	return func(c *fiber.Ctx) error {
		roleSlug, _ := c.Locals("role_slug").(string)
		if _, blocked := denied[roleSlug]; blocked {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Você não tem permissão para acessar este recurso"})
		}
		return c.Next()
	}
}

// ConversationAccess limits agents and supervisors to their allowed conversations.
func ConversationAccess(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleSlug, _ := c.Locals("role_slug").(string)
		if roleSlug != "agent" && roleSlug != "supervisor" {
			return c.Next()
		}
		companyID, _ := c.Locals("company_id").(string)
		userID, _ := c.Locals("user_id").(string)
		conversationID := c.Params("id")
		var allowed bool
		query := `
			SELECT EXISTS (
				SELECT 1 FROM conversations conv
				WHERE conv.id = $1 AND conv.company_id = $2
				  AND `
		if roleSlug == "supervisor" {
			query += `(
				conv.team_id IS NOT NULL AND EXISTS (
					SELECT 1 FROM team_users tu WHERE tu.team_id = conv.team_id AND tu.user_id = $3 AND COALESCE(tu.is_supervisor, false) = true
				)
			)`
		} else {
			query += `(
					conv.assigned_to = $3
					OR (conv.assigned_to IS NULL AND conv.team_id IS NOT NULL AND EXISTS (
						SELECT 1 FROM team_users tu WHERE tu.team_id = conv.team_id AND tu.user_id = $3
					))
					OR (conv.assigned_to IS NULL AND conv.team_id IS NULL)
				  )`
		}
		query += `)`
		err := db.QueryRow(query, conversationID, companyID, userID).Scan(&allowed)
		if err != nil || !allowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Você não tem acesso a esta conversa"})
		}
		return c.Next()
	}
}

// SupervisorTeamAccess allows supervisors to add members only to teams they supervise.
func SupervisorTeamAccess(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleSlug, _ := c.Locals("role_slug").(string)
		if roleSlug != "supervisor" {
			return c.Next()
		}
		userID, _ := c.Locals("user_id").(string)
		companyID, _ := c.Locals("company_id").(string)
		teamID := c.Params("id")
		var allowed bool
		_ = db.QueryRow(`SELECT EXISTS (SELECT 1 FROM team_users tu JOIN teams t ON t.id = tu.team_id WHERE tu.team_id = $1 AND tu.user_id = $2 AND t.company_id = $3 AND COALESCE(tu.is_supervisor, false) = true)`, teamID, userID, companyID).Scan(&allowed)
		if !allowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Você não supervisiona este time"})
		}
		return c.Next()
	}
}

// RateLimiter creates a rate limiting middleware using Redis
func RateLimiter(rdb *redis.Client, maxRequests int, window time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()
		if c.Method() == fiber.MethodOptions ||
			path == "/ws" ||
			strings.HasPrefix(path, "/api/webhooks/") {
			return c.Next()
		}

		ip := c.IP()
		if forwardedFor := strings.TrimSpace(c.Get("X-Forwarded-For")); forwardedFor != "" {
			ip = strings.TrimSpace(strings.Split(forwardedFor, ",")[0])
		} else if realIP := strings.TrimSpace(c.Get("X-Real-IP")); realIP != "" {
			ip = realIP
		}

		// Stricter rate limit for auth endpoints (brute-force protection)
		limit := maxRequests
		dur := window
		if strings.HasPrefix(path, "/api/auth/login") || strings.HasPrefix(path, "/api/auth/register") {
			limit = 5
			dur = time.Minute
		} else if strings.HasPrefix(path, "/api/auth/") {
			limit = 10
			dur = time.Minute
		}

		key := fmt.Sprintf("rate_limit:%s:%s:%s", ip, c.Method(), path)

		ctx := context.Background()
		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			return c.Next()
		}

		if count == 1 {
			rdb.Expire(ctx, key, dur)
		}

		if count > int64(limit) {
			c.Set("Retry-After", fmt.Sprintf("%.0f", dur.Seconds()))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded",
			})
		}

		return c.Next()
	}
}
