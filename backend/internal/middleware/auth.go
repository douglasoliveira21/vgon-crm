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

// RateLimiter creates a rate limiting middleware using Redis
func RateLimiter(rdb *redis.Client, maxRequests int, window time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()
		if c.Method() == fiber.MethodOptions ||
			path == "/ws" ||
			strings.HasPrefix(path, "/api/auth/") ||
			strings.HasPrefix(path, "/api/webhooks/") {
			return c.Next()
		}

		ip := c.IP()
		if forwardedFor := strings.TrimSpace(c.Get("X-Forwarded-For")); forwardedFor != "" {
			ip = strings.TrimSpace(strings.Split(forwardedFor, ",")[0])
		} else if realIP := strings.TrimSpace(c.Get("X-Real-IP")); realIP != "" {
			ip = realIP
		}

		key := fmt.Sprintf("rate_limit:%s:%s:%s", ip, c.Method(), path)

		ctx := context.Background()
		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			return c.Next()
		}

		if count == 1 {
			rdb.Expire(ctx, key, window)
		}

		if count > int64(maxRequests) {
			c.Set("Retry-After", fmt.Sprintf("%.0f", window.Seconds()))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded",
			})
		}

		return c.Next()
	}
}
