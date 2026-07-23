package websocket

import (
	"encoding/json"
	"log"
	"sync"
)

// Message types for WebSocket communication
const (
	EventNewMessage       = "new_message"
	EventMessageStatus    = "message_status"
	EventConversationNew  = "conversation_new"
	EventConversationUpdate = "conversation_update"
	EventTyping           = "typing"
	EventPresence         = "presence"
	EventNotification     = "notification"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Event     string          `json:"event"`
	Data      json.RawMessage `json:"data"`
	CompanyID string          `json:"company_id,omitempty"`
	UserID    string          `json:"user_id,omitempty"`
	Room      string          `json:"room,omitempty"`
}

// Client represents a WebSocket client
type Client struct {
	ID        string
	UserID    string
	CompanyID string
	Send      chan []byte
	Hub       *Hub
	mu        sync.Mutex
}

// Hub manages WebSocket connections
type Hub struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *WSMessage
	rooms      map[string]map[string]*Client // room -> clientID -> client
	mu         sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *WSMessage, 256),
		rooms:      make(map[string]map[string]*Client),
	}
}

// Run starts the hub event loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			// Auto-join company room
			companyRoom := "company:" + client.CompanyID
			if h.rooms[companyRoom] == nil {
				h.rooms[companyRoom] = make(map[string]*Client)
			}
			h.rooms[companyRoom][client.ID] = client
			h.mu.Unlock()
			log.Printf("WebSocket client connected: %s (user: %s, company: %s)", client.ID, client.UserID, client.CompanyID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Send)
				// Remove from all rooms
				for room := range h.rooms {
					delete(h.rooms[room], client.ID)
					if len(h.rooms[room]) == 0 {
						delete(h.rooms, room)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("WebSocket client disconnected: %s", client.ID)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// Register adds a client to the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// HasOtherUserConnection reports whether a user still has another active tab or device.
func (h *Hub) HasOtherUserConnection(userID, excludedClientID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for clientID, client := range h.clients {
		if clientID != excludedClientID && client.UserID == userID {
			return true
		}
	}
	return false
}

// JoinRoom adds a client to a room
func (h *Hub) JoinRoom(clientID, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[room] == nil {
		h.rooms[room] = make(map[string]*Client)
	}
	if client, ok := h.clients[clientID]; ok {
		h.rooms[room][clientID] = client
	}
}

// LeaveRoom removes a client from a room
func (h *Hub) LeaveRoom(clientID, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[room] != nil {
		delete(h.rooms[room], clientID)
		if len(h.rooms[room]) == 0 {
			delete(h.rooms, room)
		}
	}
}

// BroadcastToCompany sends a message to all clients in a company
func (h *Hub) BroadcastToCompany(companyID string, event string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal broadcast data: %v", err)
		return
	}

	msg := &WSMessage{
		Event:     event,
		Data:      jsonData,
		CompanyID: companyID,
		Room:      "company:" + companyID,
	}
	h.broadcast <- msg
}

// BroadcastToRoom sends a message to all clients in a room
func (h *Hub) BroadcastToRoom(room string, event string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal broadcast data: %v", err)
		return
	}

	msg := &WSMessage{
		Event: event,
		Data:  jsonData,
		Room:  room,
	}
	h.broadcast <- msg
}

// SendToUser sends a message to a specific user
func (h *Hub) SendToUser(userID string, event string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal user data: %v", err)
		return
	}

	msg := &WSMessage{
		Event:  event,
		Data:   jsonData,
		UserID: userID,
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		if client.UserID == userID {
			select {
			case client.Send <- mustMarshal(msg):
			default:
			}
		}
	}
}

func (h *Hub) broadcastMessage(msg *WSMessage) {
	data := mustMarshal(msg)

	h.mu.RLock()
	defer h.mu.RUnlock()

	// If message has a specific user target
	if msg.UserID != "" {
		for _, client := range h.clients {
			if client.UserID == msg.UserID {
				select {
				case client.Send <- data:
				default:
				}
			}
		}
		return
	}

	// If message targets a room
	if msg.Room != "" {
		if clients, ok := h.rooms[msg.Room]; ok {
			for _, client := range clients {
				select {
				case client.Send <- data:
				default:
				}
			}
		}
		return
	}

	// If message targets a company
	if msg.CompanyID != "" {
		companyRoom := "company:" + msg.CompanyID
		if clients, ok := h.rooms[companyRoom]; ok {
			for _, client := range clients {
				select {
				case client.Send <- data:
				default:
				}
			}
		}
	}
}

func mustMarshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}
