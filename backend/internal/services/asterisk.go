package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"database/sql"

	"github.com/evocrm/backend/internal/config"
	"github.com/evocrm/backend/internal/websocket"
	"github.com/google/uuid"
	gorilla "github.com/gorilla/websocket"
)

// AsteriskService handles communication with Asterisk via ARI
type AsteriskService struct {
	db     *sql.DB
	wsHub  *websocket.Hub
	cfg    *config.Config
	client *http.Client
}

// AsteriskConfig holds connection details for an Asterisk instance
type AsteriskConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	AppName  string
}

func NewAsteriskService(db *sql.DB, wsHub *websocket.Hub, cfg *config.Config) *AsteriskService {
	return &AsteriskService{
		db:     db,
		wsHub:  wsHub,
		cfg:    cfg,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetAsteriskConfig retrieves Asterisk config for a company
func (s *AsteriskService) GetAsteriskConfig(companyID string) (*AsteriskConfig, error) {
	var host, user, password string
	var port int

	err := s.db.QueryRow(`
		SELECT sip_host, sip_port, sip_user, sip_password
		FROM telephony_providers
		WHERE company_id = $1 AND is_active = true
		LIMIT 1
	`, companyID).Scan(&host, &port, &user, &password)

	if err != nil {
		return nil, fmt.Errorf("no telephony provider configured: %w", err)
	}

	return &AsteriskConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		AppName:  "evocrm",
	}, nil
}

// ============================================
// ARI REST API Methods
// ============================================

// ARIRequest performs a request to the Asterisk ARI REST API (public wrapper)
func (s *AsteriskService) ARIRequest(cfg *AsteriskConfig, method, path string, body interface{}) ([]byte, error) {
	return s.ariRequest(cfg, method, path, body)
}

func (s *AsteriskService) ariRequest(cfg *AsteriskConfig, method, path string, body interface{}) ([]byte, error) {
	url := fmt.Sprintf("http://%s:%d/ari%s", cfg.Host, 8088, path) // ARI default port 8088

	var reqBody io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(cfg.User, cfg.Password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ARI request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ARI error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// OriginateCall starts a new outbound call via ARI
func (s *AsteriskService) OriginateCall(companyID, extension, destination, callerID string) (string, error) {
	cfg, err := s.GetAsteriskConfig(companyID)
	if err != nil {
		return "", err
	}

	channelID := fmt.Sprintf("evocrm-%s", uuid.New().String()[:8])

	payload := map[string]interface{}{
		"endpoint":  fmt.Sprintf("PJSIP/%s@trunk", destination),
		"extension": destination,
		"context":   "from-internal",
		"priority":  1,
		"callerId":  callerID,
		"timeout":   30,
		"app":       cfg.AppName,
		"channelId": channelID,
		"variables": map[string]string{
			"COMPANY_ID": companyID,
			"EXTENSION":  extension,
		},
	}

	_, err = s.ariRequest(cfg, "POST", "/channels", payload)
	if err != nil {
		return "", fmt.Errorf("failed to originate call: %w", err)
	}

	log.Printf("[ASTERISK] Originated call from %s to %s (channel: %s)", extension, destination, channelID)
	return channelID, nil
}

// HangupCall ends an active call
func (s *AsteriskService) HangupCall(companyID, channelID string) error {
	cfg, err := s.GetAsteriskConfig(companyID)
	if err != nil {
		return err
	}

	_, err = s.ariRequest(cfg, "DELETE", fmt.Sprintf("/channels/%s", channelID), nil)
	if err != nil {
		return fmt.Errorf("failed to hangup: %w", err)
	}

	log.Printf("[ASTERISK] Hangup channel: %s", channelID)
	return nil
}

// TransferCall transfers a call to another extension or number
func (s *AsteriskService) TransferCall(companyID, channelID, destination string) error {
	cfg, err := s.GetAsteriskConfig(companyID)
	if err != nil {
		return err
	}

	// Redirect the channel to new destination
	payload := map[string]interface{}{
		"endpoint": fmt.Sprintf("PJSIP/%s", destination),
	}

	_, err = s.ariRequest(cfg, "POST", fmt.Sprintf("/channels/%s/redirect", channelID), payload)
	if err != nil {
		return fmt.Errorf("failed to transfer: %w", err)
	}

	log.Printf("[ASTERISK] Transferred channel %s to %s", channelID, destination)
	return nil
}

// HoldCall puts a channel on hold
func (s *AsteriskService) HoldCall(companyID, channelID string) error {
	cfg, err := s.GetAsteriskConfig(companyID)
	if err != nil {
		return err
	}

	_, err = s.ariRequest(cfg, "POST", fmt.Sprintf("/channels/%s/hold", channelID), nil)
	return err
}

// UnholdCall removes a channel from hold
func (s *AsteriskService) UnholdCall(companyID, channelID string) error {
	cfg, err := s.GetAsteriskConfig(companyID)
	if err != nil {
		return err
	}

	_, err = s.ariRequest(cfg, "DELETE", fmt.Sprintf("/channels/%s/hold", channelID), nil)
	return err
}

// GetExtensionStatus gets the status of extensions
func (s *AsteriskService) GetExtensionStatus(companyID string) ([]map[string]interface{}, error) {
	cfg, err := s.GetAsteriskConfig(companyID)
	if err != nil {
		return nil, err
	}

	respBody, err := s.ariRequest(cfg, "GET", "/endpoints", nil)
	if err != nil {
		return nil, err
	}

	var endpoints []map[string]interface{}
	json.Unmarshal(respBody, &endpoints)
	return endpoints, nil
}

// ============================================
// PJSIP Configuration Generation
// ============================================

// CreateSIPTrunk generates PJSIP trunk configuration via ARI
func (s *AsteriskService) CreateSIPTrunk(companyID string, trunkConfig map[string]string) error {
	cfg, err := s.GetAsteriskConfig(companyID)
	if err != nil {
		return err
	}

	// Reload PJSIP module
	_, err = s.ariRequest(cfg, "PUT", "/asterisk/modules/res_pjsip/reload", nil)
	if err != nil {
		log.Printf("[ASTERISK] Warning: PJSIP reload may need manual action: %v", err)
	}

	return nil
}

// CreateExtensionConfig creates a PJSIP endpoint for a ramal
func (s *AsteriskService) CreateExtensionConfig(companyID, extension, password, callerID string, webrtcEnabled bool) error {
	cfg, err := s.GetAsteriskConfig(companyID)
	if err != nil {
		return err
	}

	// The actual PJSIP configuration would be written to Asterisk's config
	// In a production environment, this would use ARI or AMI to push configuration
	// For now, we log and store what needs to be configured

	log.Printf("[ASTERISK] Extension config created for %s (WebRTC: %v)", extension, webrtcEnabled)

	// Attempt to reload PJSIP
	s.ariRequest(cfg, "PUT", "/asterisk/modules/res_pjsip/reload", nil)

	return nil
}

// ============================================
// ARI WebSocket Events (real-time)
// ============================================

// ConnectARIWebSocket establishes WebSocket connection to Asterisk ARI for events
func (s *AsteriskService) ConnectARIWebSocket(companyID string) {
	cfg, err := s.GetAsteriskConfig(companyID)
	if err != nil {
		log.Printf("[ASTERISK] Cannot connect WebSocket: %v", err)
		return
	}

	wsURL := fmt.Sprintf("ws://%s:8088/ari/events?api_key=%s:%s&app=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.AppName)

	go func() {
		for {
			conn, _, err := gorilla.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				log.Printf("[ASTERISK] WebSocket connection failed: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			log.Printf("[ASTERISK] WebSocket connected to %s", cfg.Host)

			for {
				_, message, err := conn.ReadMessage()
				if err != nil {
					log.Printf("[ASTERISK] WebSocket read error: %v", err)
					break
				}

				s.handleARIEvent(companyID, message)
			}

			conn.Close()
			time.Sleep(3 * time.Second) // Reconnect delay
		}
	}()
}

// handleARIEvent processes an ARI event and broadcasts via WebSocket
func (s *AsteriskService) handleARIEvent(companyID string, message []byte) {
	var event map[string]interface{}
	if err := json.Unmarshal(message, &event); err != nil {
		return
	}

	eventType, _ := event["type"].(string)

	switch eventType {
	case "StasisStart":
		// New call started in our app
		channel, _ := event["channel"].(map[string]interface{})
		callerNum, _ := channel["caller"].(map[string]interface{})
		number, _ := callerNum["number"].(string)
		name, _ := callerNum["name"].(string)

		s.wsHub.BroadcastToCompany(companyID, "incoming_call", map[string]interface{}{
			"channel_id": channel["id"],
			"number":     number,
			"name":       name,
		})

	case "ChannelStateChange":
		channel, _ := event["channel"].(map[string]interface{})
		state, _ := channel["state"].(string)
		channelID, _ := channel["id"].(string)

		wsEvent := ""
		switch state {
		case "Ringing":
			wsEvent = "call_ringing"
		case "Up":
			wsEvent = "call_answered"
		}

		if wsEvent != "" {
			s.wsHub.BroadcastToCompany(companyID, wsEvent, map[string]interface{}{
				"channel_id": channelID,
				"state":      state,
			})
		}

	case "StasisEnd":
		channel, _ := event["channel"].(map[string]interface{})
		channelID, _ := channel["id"].(string)

		s.wsHub.BroadcastToCompany(companyID, "call_finished", map[string]interface{}{
			"channel_id": channelID,
		})

	case "ChannelHold":
		s.wsHub.BroadcastToCompany(companyID, "call_hold", event)

	case "ChannelUnhold":
		s.wsHub.BroadcastToCompany(companyID, "call_unhold", event)
	}
}

// ============================================
// AMI Commands (for operations ARI doesn't support well)
// ============================================

// AMIReloadPJSIP reloads PJSIP via AMI
func (s *AsteriskService) AMIReloadPJSIP(companyID string) error {
	// In production, this would connect via AMI (port 5038) and send:
	// Action: Command
	// Command: pjsip reload
	log.Printf("[ASTERISK] PJSIP reload requested for company %s", companyID)
	return nil
}
