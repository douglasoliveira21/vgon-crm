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
	Contact   *ContactService
	Bot       *BotEngine
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
	container.Evolution = NewEvolutionService(cfg, db, wsHub)
	container.Message = NewMessageService(db, rdb, wsHub)
	container.Contact = NewContactService(db)
	container.Bot = NewBotEngine(db, wsHub, container.Evolution)

	// Link bot engine to evolution service
	container.Evolution.botEngine = container.Bot

	return container
}
