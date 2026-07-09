package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	_ "time/tzdata"

	"github.com/evocrm/backend/internal/config"
	"github.com/evocrm/backend/internal/database"
	"github.com/evocrm/backend/internal/handlers"
	"github.com/evocrm/backend/internal/middleware"
	"github.com/evocrm/backend/internal/redis"
	"github.com/evocrm/backend/internal/services"
	"github.com/evocrm/backend/internal/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize Redis
	rdb, err := redis.Connect(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer rdb.Close()

	// Initialize WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Initialize services
	svc := services.NewContainer(db, rdb, cfg, wsHub)

	// Start periodic background tasks
	svc.Evolution.StartPeriodicPhotoSync()
	svc.Evolution.StartWhatsAppInstanceMonitor()
	svc.Email.StartPeriodicSync()
	svc.Bot.StartClientInactivityMonitor()

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		BodyLimit:    50 * 1024 * 1024, // 50MB
		ServerHeader: "EvoCRM Pro",
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} ${latency}\n",
	}))

	// Open CORS for widget routes (must be before the restrictive CORS)
	app.Use(func(c *fiber.Ctx) error {
		if strings.Contains(c.Path(), "/widget/") {
			c.Set("Access-Control-Allow-Origin", "*")
			c.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			c.Set("Access-Control-Allow-Headers", "Content-Type")
			if c.Method() == "OPTIONS" {
				return c.SendStatus(204)
			}
		}
		return c.Next()
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins:     buildAllowedOrigins(cfg.FrontendURL),
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Requested-With",
		AllowMethods:     "GET, POST, PUT, DELETE, PATCH, OPTIONS",
		AllowCredentials: false,
		Next: func(c *fiber.Ctx) bool {
			// Widget routes already handled above.
			return strings.Contains(c.Path(), "/widget/")
		},
	}))
	app.Options("*", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	// Security headers
	app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "camera=(), microphone=(self), geolocation=()")
		return c.Next()
	})

	// Rate limiting
	app.Use(middleware.RateLimiter(rdb, cfg.RateLimitMax, cfg.RateLimitWindow))

	// Setup routes
	handlers.SetupRoutes(app, svc, wsHub)

	// Start server
	port := cfg.AppPort
	if port == "" {
		port = "3001"
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	log.Printf("🚀 EvoCRM Pro API running on port %s", port)

	<-quit
	log.Println("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Server stopped gracefully")
}

func buildAllowedOrigins(frontendURL string) string {
	origins := []string{
		"https://crm.vgon.com.br",
		"https://www.crm.vgon.com.br",
		"http://localhost:3000",
		"http://localhost:3001",
		"http://127.0.0.1:3000",
	}

	frontendURL = strings.TrimSpace(frontendURL)
	if frontendURL != "" {
		origins = append(origins, strings.TrimRight(frontendURL, "/"))
	}

	seen := make(map[string]bool, len(origins))
	unique := make([]string, 0, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(strings.TrimRight(origin, "/"))
		if origin == "" || seen[origin] {
			continue
		}
		seen[origin] = true
		unique = append(unique, origin)
	}

	return strings.Join(unique, ",")
}
