package handlers

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/evocrm/backend/internal/config"
	"github.com/evocrm/backend/internal/middleware"
	ws "github.com/evocrm/backend/internal/websocket"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func WebSocketHandler(hub *ws.Hub, cfg *config.Config) fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		// Authenticate via query param token
		token := c.Query("token")
		if token == "" {
			c.Close()
			return
		}

		// Validate token
		claims := &middleware.Claims{}
		jwtToken, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !jwtToken.Valid {
			c.Close()
			return
		}

		// Create client
		clientID := uuid.New().String()
		client := &ws.Client{
			ID:        clientID,
			UserID:    claims.UserID,
			CompanyID: claims.CompanyID,
			Send:      make(chan []byte, 256),
			Hub:       hub,
		}

		hub.Register(client)
		defer hub.Unregister(client)

		// Write pump
		go func() {
			for msg := range client.Send {
				if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
					break
				}
			}
		}()

		// Read pump
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				break
			}

			// Handle incoming WebSocket messages
			var wsMsg struct {
				Event string          `json:"event"`
				Data  json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(message, &wsMsg); err != nil {
				continue
			}

			switch wsMsg.Event {
			case "join_conversation":
				var data struct {
					ConversationID string `json:"conversation_id"`
				}
				json.Unmarshal(wsMsg.Data, &data)
				hub.JoinRoom(clientID, "conversation:"+data.ConversationID)

			case "leave_conversation":
				var data struct {
					ConversationID string `json:"conversation_id"`
				}
				json.Unmarshal(wsMsg.Data, &data)
				hub.LeaveRoom(clientID, "conversation:"+data.ConversationID)

			case "typing":
				var data struct {
					ConversationID string `json:"conversation_id"`
					IsTyping       bool   `json:"is_typing"`
				}
				json.Unmarshal(wsMsg.Data, &data)
				hub.BroadcastToRoom("conversation:"+data.ConversationID, "typing", map[string]interface{}{
					"user_id":         claims.UserID,
					"conversation_id": data.ConversationID,
					"is_typing":       data.IsTyping,
				})

			default:
				log.Printf("Unknown WebSocket event: %s", wsMsg.Event)
			}
		}
	})
}
