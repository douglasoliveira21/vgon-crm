package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/evocrm/backend/internal/config"
	"github.com/evocrm/backend/internal/middleware"
	"github.com/evocrm/backend/internal/services"
	ws "github.com/evocrm/backend/internal/websocket"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func WebSocketHandler(hub *ws.Hub, cfg *config.Config, svc *services.Container) fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		if strings.TrimRight(c.Headers("Origin"), "/") != strings.TrimRight(cfg.FrontendURL, "/") {
			c.Close()
			return
		}
		// Browser clients authenticate exclusively with the HttpOnly cookie.
		token := c.Cookies("crm_access")
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
		var active bool
		if claims.SessionID == "" || svc.DB.QueryRow(`SELECT EXISTS (
			SELECT 1 FROM refresh_tokens
			WHERE session_id = $1 AND user_id = $2 AND revoked_at IS NULL AND expires_at > NOW()
		)`, claims.SessionID, claims.UserID).Scan(&active) != nil || !active {
			c.Close()
			return
		}

		// Create client
		clientID := uuid.New().String()
		client := &ws.Client{
			ID:        clientID,
			UserID:    claims.UserID,
			CompanyID: claims.CompanyID,
			RoleSlug:  claims.RoleSlug,
			Send:      make(chan []byte, 256),
			Hub:       hub,
		}

		hub.Register(client)
		if svc != nil && svc.DB != nil {
			svc.DB.Exec("UPDATE users SET is_online = true, last_seen_at = NOW(), updated_at = NOW() WHERE id = $1", claims.UserID)
		}
		defer func() {
			hub.Unregister(client)
			if svc != nil && svc.DB != nil && !hub.HasOtherUserConnection(claims.UserID, clientID) {
				svc.DB.Exec("UPDATE users SET is_online = false, last_seen_at = NOW(), updated_at = NOW() WHERE id = $1", claims.UserID)
			}
		}()

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
				if json.Unmarshal(wsMsg.Data, &data) != nil ||
					!middleware.CanAccessConversation(svc.DB, data.ConversationID, claims.CompanyID, claims.UserID, claims.RoleSlug) {
					continue
				}
				hub.JoinRoom(clientID, "conversation:"+data.ConversationID)

			case "leave_conversation":
				var data struct {
					ConversationID string `json:"conversation_id"`
				}
				if json.Unmarshal(wsMsg.Data, &data) != nil ||
					!middleware.CanAccessConversation(svc.DB, data.ConversationID, claims.CompanyID, claims.UserID, claims.RoleSlug) {
					continue
				}
				hub.LeaveRoom(clientID, "conversation:"+data.ConversationID)

			case "typing":
				var data struct {
					ConversationID string `json:"conversation_id"`
					IsTyping       bool   `json:"is_typing"`
				}
				if json.Unmarshal(wsMsg.Data, &data) != nil ||
					!middleware.CanAccessConversation(svc.DB, data.ConversationID, claims.CompanyID, claims.UserID, claims.RoleSlug) {
					continue
				}
				// Get agent name
				var userName string
				if svc != nil && svc.DB != nil {
					svc.DB.QueryRow("SELECT COALESCE(name, '') FROM users WHERE id = $1", claims.UserID).Scan(&userName)
				}
				typingPayload := map[string]interface{}{
					"user_id":         claims.UserID,
					"user_name":       userName,
					"conversation_id": data.ConversationID,
					"is_typing":       data.IsTyping,
				}
				hub.BroadcastToRoom("conversation:"+data.ConversationID, "typing", typingPayload)
				// Also broadcast to widget visitor
				hub.BroadcastToRoom("widget:"+data.ConversationID, "typing", typingPayload)

			default:
				log.Printf("Unknown WebSocket event: %s", wsMsg.Event)
			}
		}
	})
}

// WidgetWebSocketHandler provides a public (no JWT) WebSocket connection for
// website visitors using the chat widget. The visitor joins a room keyed by
// their conversation_id so they receive agent/bot replies in real-time.
func WidgetWebSocketHandler(hub *ws.Hub, svc *services.Container) fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		conversationID := c.Query("conversation_id")
		visitorID := c.Query("visitor_id")
		sessionToken := c.Query("session_token")
		if !svc.Auth.ValidateWidgetSocket(conversationID, visitorID, sessionToken) {
			c.Close()
			return
		}

		clientID := uuid.New().String()
		client := &ws.Client{
			ID:        clientID,
			UserID:    "visitor",
			CompanyID: "widget",
			Send:      make(chan []byte, 64),
			Hub:       hub,
		}

		hub.Register(client)
		hub.JoinRoom(clientID, "widget:"+conversationID)
		defer hub.Unregister(client)

		// Write pump — sends messages to the visitor browser
		go func() {
			for msg := range client.Send {
				if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
					break
				}
			}
		}()

		// Read pump — keep connection alive (visitor doesn't send meaningful frames)
		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				break
			}
		}
	})
}
