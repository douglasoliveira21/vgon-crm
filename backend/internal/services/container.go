package services

import (
	"database/sql"

	"github.com/evocrm/backend/internal/config"
	"github.com/evocrm/backend/internal/websocket"
	"github.com/redis/go-redis/v9"
)

// Container holds all service dependencies
type Container struct {
	DB        *sql.DB
	Redis     *redis.Client
	Config    *config.Config
	WSHub     *websocket.Hub
	Auth      *AuthService
	Evolution *EvolutionService
	Message   *MessageService
	Email     *EmailService
	Contact   *ContactService
	Bot       *BotEngine
	GLPI      *GLPIService
	GLPIFlow  *GLPIFlowEngine
	Jobs      *JobQueue
}

// NewContainer creates a new service container
func NewContainer(db *sql.DB, rdb *redis.Client, cfg *config.Config, wsHub *websocket.Hub) *Container {
	container := &Container{
		DB:     db,
		Redis:  rdb,
		Config: cfg,
		WSHub:  wsHub,
	}

	container.Auth = NewAuthService(db, cfg)
	container.Jobs = NewJobQueue(db)
	container.Evolution = NewEvolutionService(cfg, db, wsHub)
	container.Message = NewMessageService(db, rdb, wsHub)
	container.Email = NewEmailService(db, wsHub, cfg)
	container.Contact = NewContactService(db)
	container.Bot = NewBotEngine(db, wsHub, container.Evolution)
	container.Bot.EnableDurableJobs(container.Jobs)
	container.GLPI = NewGLPIService(cfg.GLPIBaseURL, cfg.GLPIAppToken)
	container.GLPIFlow = NewGLPIFlowEngine(db, wsHub, container.Evolution, container.GLPI, cfg.GLPIUserToken)

	// Link bot engine to evolution service
	container.Evolution.botEngine = container.Bot
	container.Evolution.glpiFlow = container.GLPIFlow

	// Link GLPI flow to bot engine
	container.Bot.glpiFlow = container.GLPIFlow
	container.GLPIFlow.SetBotEngine(container.Bot)

	return container
}
