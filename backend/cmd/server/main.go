package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

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
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, DELETE, PATCH, OPTIONS",
		AllowCredentials: false,
	}))

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
