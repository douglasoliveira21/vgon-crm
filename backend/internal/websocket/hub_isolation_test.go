package websocket

import (
	"encoding/json"
	"testing"
)

func TestRoomBroadcastAppliesAuthorization(t *testing.T) {
	hub := NewHub()
	allowed := &Client{ID: "allowed", CompanyID: "tenant-a", RoleSlug: "supervisor", Send: make(chan []byte, 1)}
	denied := &Client{ID: "denied", CompanyID: "tenant-a", RoleSlug: "agent", Send: make(chan []byte, 1)}
	otherTenant := &Client{ID: "other", CompanyID: "tenant-b", RoleSlug: "admin", Send: make(chan []byte, 1)}
	hub.clients[allowed.ID] = allowed
	hub.clients[denied.ID] = denied
	hub.clients[otherTenant.ID] = otherTenant
	hub.rooms["conversation:one"] = map[string]*Client{
		allowed.ID: allowed, denied.ID: denied, otherTenant.ID: otherTenant,
	}
	hub.SetAuthorizer(func(client *Client, _ *WSMessage) bool {
		return client.CompanyID == "tenant-a" && client.RoleSlug == "supervisor"
	})

	hub.broadcastMessage(&WSMessage{
		Event: "new_message", Room: "conversation:one",
		Data: json.RawMessage(`{"conversation_id":"one"}`),
	})
	if len(allowed.Send) != 1 {
		t.Fatal("authorized supervisor did not receive room event")
	}
	if len(denied.Send) != 0 {
		t.Fatal("unauthorized agent received room event")
	}
	if len(otherTenant.Send) != 0 {
		t.Fatal("cross-tenant client received room event")
	}
}
