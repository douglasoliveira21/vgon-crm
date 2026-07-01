package services

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/evocrm/backend/internal/config"
	"github.com/evocrm/backend/internal/models"
	"github.com/evocrm/backend/internal/websocket"
	"github.com/google/uuid"
)

type EvolutionService struct {
	cfg       *config.Config
	db        *sql.DB
	wsHub     *websocket.Hub
	client    *http.Client
	botEngine *BotEngine
	glpiFlow  *GLPIFlowEngine
}

func NewEvolutionService(cfg *config.Config, db *sql.DB, wsHub *websocket.Hub) *EvolutionService {
	return &EvolutionService{
		cfg:   cfg,
		db:    db,
		wsHub: wsHub,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateInstanceRequest represents the request to create a WhatsApp instance
type CreateInstanceRequest struct {
	InstanceName string `json:"instance_name"`
	CompanyID    string `json:"company_id"`
	ChannelName  string `json:"channel_name"`
}

// EvolutionCreateResponse represents the Evolution API response
type EvolutionCreateResponse struct {
	Instance struct {
		InstanceName string `json:"instanceName"`
		InstanceID   string `json:"instanceId"`
		Status       string `json:"status"`
	} `json:"instance"`
	Hash   map[string]string `json:"hash"`
	QRCode struct {
		Code   string `json:"code"`
		Base64 string `json:"base64"`
	} `json:"qrcode"`
	Settings map[string]interface{} `json:"settings"`
}

// CreateInstance creates a new WhatsApp instance via Evolution API
func (s *EvolutionService) CreateInstance(req *CreateInstanceRequest) (*models.WhatsAppInstance, error) {
	// Create the instance via Evolution API
	payload := map[string]interface{}{
		"instanceName": req.InstanceName,
		"qrcode":       true,
		"integration":  "WHATSAPP-BAILEYS",
		"webhook": map[string]interface{}{
			"url":     fmt.Sprintf("%s/%s", s.cfg.EvolutionWebhookURL, req.InstanceName),
			"enabled": true,
			"events": []string{
				"MESSAGES_UPSERT",
				"MESSAGES_UPDATE",
				"MESSAGES_DELETE",
				"SEND_MESSAGE",
				"CONNECTION_UPDATE",
				"CONTACTS_UPSERT",
				"QRCODE_UPDATED",
				"PRESENCE_UPDATE",
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s/instance/create", s.cfg.EvolutionAPIURL), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apikey", s.cfg.EvolutionAPIKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call Evolution API: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Evolution API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Parse response flexibly (Evolution API v2.x format varies)
	var rawResp map[string]interface{}
	if err := json.Unmarshal(respBody, &rawResp); err != nil {
		return nil, fmt.Errorf("failed to parse Evolution API response: %w", err)
	}

	// Extract instance info
	var instanceIDValue string
	var qrcodeBase64 string
	var token string

	if inst, ok := rawResp["instance"].(map[string]interface{}); ok {
		if id, ok := inst["instanceId"].(string); ok {
			instanceIDValue = id
		}
	}

	if qr, ok := rawResp["qrcode"].(map[string]interface{}); ok {
		if b64, ok := qr["base64"].(string); ok {
			qrcodeBase64 = b64
		}
	}

	if hash, ok := rawResp["hash"].(map[string]interface{}); ok {
		if apiKey, ok := hash["apikey"].(string); ok {
			token = apiKey
		}
	}

	// Also check top-level fields (v2.4.0 format)
	if qrcodeBase64 == "" {
		if b64, ok := rawResp["base64"].(string); ok {
			qrcodeBase64 = b64
		}
	}
	if token == "" {
		if t, ok := rawResp["token"].(string); ok {
			token = t
		}
	}

	// Create channel
	channelID := uuid.New().String()
	_, err = s.db.Exec(`
		INSERT INTO channels (id, company_id, name, type, status)
		VALUES ($1, $2, $3, 'whatsapp', 'connecting')
	`, channelID, req.CompanyID, req.ChannelName)
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	// Save instance to database
	instanceID := uuid.New().String()
	webhookURL := fmt.Sprintf("%s/%s", s.cfg.EvolutionWebhookURL, req.InstanceName)

	_, err = s.db.Exec(`
		INSERT INTO whatsapp_instances (id, company_id, channel_id, instance_name, instance_id, token, status, qrcode, webhook_url)
		VALUES ($1, $2, $3, $4, $5, $6, 'qr_code', $7, $8)
	`, instanceID, req.CompanyID, channelID, req.InstanceName, instanceIDValue, token, qrcodeBase64, webhookURL)
	if err != nil {
		return nil, fmt.Errorf("failed to save instance: %w", err)
	}

	instance := &models.WhatsAppInstance{
		ID:           instanceID,
		CompanyID:    req.CompanyID,
		ChannelID:    &channelID,
		InstanceName: req.InstanceName,
		InstanceID:   &instanceIDValue,
		Token:        &token,
		Status:       "qr_code",
		QRCode:       &qrcodeBase64,
		WebhookURL:   &webhookURL,
	}

	return instance, nil
}

// GetQRCode fetches the current QR code for an instance
func (s *EvolutionService) GetQRCode(instanceName string) (string, error) {
	httpReq, err := http.NewRequest("GET", fmt.Sprintf("%s/instance/connect/%s", s.cfg.EvolutionAPIURL, instanceName), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("apikey", s.cfg.EvolutionAPIKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to call Evolution API: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("Evolution API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if base64, ok := result["base64"].(string); ok {
		// Update in database
		s.db.Exec("UPDATE whatsapp_instances SET qrcode = $1, updated_at = NOW() WHERE instance_name = $2", base64, instanceName)
		return base64, nil
	}

	return "", fmt.Errorf("no QR code available")
}

// GetConnectionStatus checks the connection status of an instance
func (s *EvolutionService) GetConnectionStatus(instanceName string) (string, error) {
	httpReq, err := http.NewRequest("GET", fmt.Sprintf("%s/instance/connectionState/%s", s.cfg.EvolutionAPIURL, instanceName), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("apikey", s.cfg.EvolutionAPIKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to call Evolution API: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("evolution send text returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if instance, ok := result["instance"].(map[string]interface{}); ok {
		if state, ok := instance["state"].(string); ok {
			// Update status in database
			status := mapEvolutionStatus(state)
			s.db.Exec("UPDATE whatsapp_instances SET status = $1, updated_at = NOW() WHERE instance_name = $2", status, instanceName)
			s.db.Exec("UPDATE channels SET status = $1, updated_at = NOW() WHERE id = (SELECT channel_id FROM whatsapp_instances WHERE instance_name = $2)", status, instanceName)
			return status, nil
		}
	}

	return "disconnected", nil
}

// GetMediaBase64 fetches media content as base64 from Evolution API
func (s *EvolutionService) GetMediaBase64(instanceName, messageID string) (string, string, error) {
	// Get the external message ID and contact phone from our database
	var externalID string
	var contactPhone string
	err := s.db.QueryRow(`
		SELECT m.external_id, co.phone
		FROM messages m
		JOIN conversations c ON m.conversation_id = c.id
		JOIN contacts co ON c.contact_id = co.id
		WHERE m.id = $1
	`, messageID).Scan(&externalID, &contactPhone)
	if err != nil || externalID == "" {
		return "", "", fmt.Errorf("external message ID not found")
	}

	// Build remoteJid from phone
	remoteJid := contactPhone + "@s.whatsapp.net"

	payload := map[string]interface{}{
		"message": map[string]interface{}{
			"key": map[string]interface{}{
				"id":        externalID,
				"remoteJid": remoteJid,
				"fromMe":    false,
			},
		},
		"convertToMp4": false,
	}

	body, _ := json.Marshal(payload)

	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s/chat/getBase64FromMediaMessage/%s", s.cfg.EvolutionAPIURL, instanceName), bytes.NewBuffer(body))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apikey", s.cfg.EvolutionAPIKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", "", fmt.Errorf("failed to call Evolution API: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Printf("[MEDIA] Evolution API returned %d: %s", resp.StatusCode, string(respBody))
		return "", "", fmt.Errorf("Evolution API error (status %d)", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", "", fmt.Errorf("failed to parse response: %w", err)
	}

	base64Data, _ := result["base64"].(string)
	mimeType, _ := result["mimetype"].(string)

	// Also try "mediaUrl" field (some versions return direct URL)
	if base64Data == "" {
		if mediaURL, ok := result["mediaUrl"].(string); ok && mediaURL != "" {
			return "", "", fmt.Errorf("got mediaUrl instead of base64, fallback needed")
		}
		return "", "", fmt.Errorf("no base64 data in response")
	}

	// Remove data URI prefix if present
	if len(base64Data) > 100 && base64Data[:5] == "data:" {
		// Format: data:audio/ogg;base64,XXXX
		parts := splitOnce(base64Data, ",")
		if len(parts) == 2 {
			base64Data = parts[1]
		}
	}

	return base64Data, mimeType, nil
}

// DisconnectInstance disconnects a WhatsApp instance
func (s *EvolutionService) DisconnectInstance(instanceName string) error {
	httpReq, err := http.NewRequest("DELETE", fmt.Sprintf("%s/instance/logout/%s", s.cfg.EvolutionAPIURL, instanceName), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("apikey", s.cfg.EvolutionAPIKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call Evolution API: %w", err)
	}
	defer resp.Body.Close()

	s.db.Exec("UPDATE whatsapp_instances SET status = 'disconnected', qrcode = NULL, updated_at = NOW() WHERE instance_name = $1", instanceName)
	s.db.Exec("UPDATE channels SET status = 'disconnected', updated_at = NOW() WHERE id = (SELECT channel_id FROM whatsapp_instances WHERE instance_name = $1)", instanceName)

	return nil
}

// DeleteInstance removes a WhatsApp instance
func (s *EvolutionService) DeleteInstance(instanceName string) error {
	httpReq, err := http.NewRequest("DELETE", fmt.Sprintf("%s/instance/delete/%s", s.cfg.EvolutionAPIURL, instanceName), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("apikey", s.cfg.EvolutionAPIKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call Evolution API: %w", err)
	}
	defer resp.Body.Close()

	// Remove from database
	s.db.Exec("DELETE FROM whatsapp_instances WHERE instance_name = $1", instanceName)

	return nil
}

// SendTextMessage sends a text message via WhatsApp
func (s *EvolutionService) SendTextMessage(instanceName, phone, text string) (string, error) {
	return s.SendTextMessageWithQuote(instanceName, phone, text, "")
}

// SendTextMessageWithQuote sends a text message with optional quoted message
func (s *EvolutionService) SendTextMessageWithQuote(instanceName, phone, text, quotedMsgID string) (string, error) {
	payload := map[string]interface{}{
		"number": phone,
		"text":   text,
	}

	if quotedMsgID != "" {
		payload["quoted"] = map[string]interface{}{
			"key": map[string]interface{}{
				"id": quotedMsgID,
			},
		}
	}

	body, _ := json.Marshal(payload)

	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s/message/sendText/%s", s.cfg.EvolutionAPIURL, instanceName), bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apikey", s.cfg.EvolutionAPIKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("evolution send text returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if key, ok := result["key"].(map[string]interface{}); ok {
		if id, ok := key["id"].(string); ok {
			return id, nil
		}
	}

	return "", fmt.Errorf("evolution send text did not return message id: %s", string(respBody))
}

// SendMediaMessage sends a media message via WhatsApp
func (s *EvolutionService) SendMediaMessage(instanceName, phone, mediaType, mediaURL, caption, fileName string) (string, error) {
	payload := map[string]interface{}{
		"number":    phone,
		"mediatype": mediaType,
		"media":     mediaURL,
		"caption":   caption,
		"fileName":  fileName,
	}

	body, _ := json.Marshal(payload)

	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s/message/sendMedia/%s", s.cfg.EvolutionAPIURL, instanceName), bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apikey", s.cfg.EvolutionAPIKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send media: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("evolution send media returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse media response: %w", err)
	}

	if key, ok := result["key"].(map[string]interface{}); ok {
		if id, ok := key["id"].(string); ok {
			return id, nil
		}
	}

	return "", fmt.Errorf("evolution send media did not return message id: %s", string(respBody))
}

// SendAudioMessage sends an audio message via WhatsApp
func (s *EvolutionService) SendAudioMessage(instanceName, phone, audioURL string) (string, error) {
	payload := map[string]interface{}{
		"number": phone,
		"audio":  audioURL,
	}

	body, _ := json.Marshal(payload)

	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s/message/sendWhatsAppAudio/%s", s.cfg.EvolutionAPIURL, instanceName), bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apikey", s.cfg.EvolutionAPIKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send audio: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("evolution send audio returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse audio response: %w", err)
	}

	if key, ok := result["key"].(map[string]interface{}); ok {
		if id, ok := key["id"].(string); ok {
			return id, nil
		}
	}

	return "", fmt.Errorf("evolution send audio did not return message id: %s", string(respBody))
}

// SendAudioBase64 sends a base64 encoded audio via WhatsApp
func (s *EvolutionService) SendAudioBase64(instanceName, phone, audioBase64 string) (string, error) {
	// Remove data URI prefix if present (data:audio/ogg;base64,...)
	base64Data := audioBase64
	if len(base64Data) > 30 {
		parts := splitOnce(base64Data, ",")
		if len(parts) == 2 {
			base64Data = parts[1]
		}
	}

	payload := map[string]interface{}{
		"number": phone,
		"audio":  "data:audio/ogg;base64," + base64Data,
	}

	body, _ := json.Marshal(payload)

	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s/message/sendWhatsAppAudio/%s", s.cfg.EvolutionAPIURL, instanceName), bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apikey", s.cfg.EvolutionAPIKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send audio: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	if key, ok := result["key"].(map[string]interface{}); ok {
		if id, ok := key["id"].(string); ok {
			return id, nil
		}
	}

	log.Printf("[AUDIO] Send audio response: %s", string(respBody))
	return "", nil
}

// HandleWebhook processes incoming webhook events from Evolution API
func (s *EvolutionService) HandleWebhook(instanceName string, event map[string]interface{}) {
	eventType, _ := event["event"].(string)

	// Evolution API v2.4.0 format detection
	if eventType == "" {
		// Try to detect event type from payload structure
		if _, ok := event["data"].(map[string]interface{}); ok {
			if _, hasKey := event["event"]; !hasKey {
				// Check for message structure
				if _, ok := event["key"]; ok {
					eventType = "messages.upsert"
				}
			}
		}
	}

	log.Printf("[WEBHOOK] Processing event '%s' for instance '%s'", eventType, instanceName)

	switch eventType {
	case "connection.update":
		s.handleConnectionUpdate(instanceName, event)
	case "messages.upsert":
		s.handleMessageUpsert(instanceName, event)
	case "messages.update":
		s.handleMessageUpdate(instanceName, event)
	case "qrcode.updated":
		s.handleQRCodeUpdate(instanceName, event)
	case "presence.update":
		s.handlePresenceUpdate(instanceName, event)
	default:
		log.Printf("[WEBHOOK] Unhandled event: %s for instance: %s", eventType, instanceName)
	}
}

func (s *EvolutionService) handleConnectionUpdate(instanceName string, event map[string]interface{}) {
	data, _ := event["data"].(map[string]interface{})
	state, _ := data["state"].(string)

	status := mapEvolutionStatus(state)

	s.db.Exec("UPDATE whatsapp_instances SET status = $1, updated_at = NOW() WHERE instance_name = $2", status, instanceName)
	s.db.Exec("UPDATE channels SET status = $1, updated_at = NOW() WHERE id = (SELECT channel_id FROM whatsapp_instances WHERE instance_name = $2)", status, instanceName)

	if status == "connected" {
		s.db.Exec("UPDATE whatsapp_instances SET connected_at = NOW(), qrcode = NULL WHERE instance_name = $1", instanceName)
	}

	// Get company ID for WebSocket notification
	var companyID string
	s.db.QueryRow("SELECT company_id FROM whatsapp_instances WHERE instance_name = $1", instanceName).Scan(&companyID)

	if companyID != "" {
		s.wsHub.BroadcastToCompany(companyID, "channel_status", map[string]string{
			"instance_name": instanceName,
			"status":        status,
		})
	}
}

func (s *EvolutionService) handleMessageUpsert(instanceName string, event map[string]interface{}) {
	data, _ := event["data"].(map[string]interface{})

	// Get instance info
	var instance models.WhatsAppInstance
	err := s.db.QueryRow(`
		SELECT id, company_id, channel_id FROM whatsapp_instances WHERE instance_name = $1
	`, instanceName).Scan(&instance.ID, &instance.CompanyID, &instance.ChannelID)
	if err != nil {
		log.Printf("Instance not found for webhook: %s", instanceName)
		return
	}

	// Parse message
	key, _ := data["key"].(map[string]interface{})
	fromMe, _ := key["fromMe"].(bool)
	remoteJid, _ := key["remoteJid"].(string)
	messageID, _ := key["id"].(string)

	// Skip if message from us (already tracked)
	if fromMe {
		return
	}
	if s.isKnownOutgoingMessage(messageID) {
		log.Printf("[WEBHOOK] Skipping outgoing message echo %s from instance %s", messageID, instanceName)
		return
	}

	// Skip group messages (groups have @g.us in JID)
	if strings.Contains(remoteJid, "@g.us") || strings.Contains(remoteJid, "@broadcast") {
		log.Printf("[WEBHOOK] Skipping group/broadcast message from %s", remoteJid)
		return
	}

	// Extract phone number from JID
	phone := extractPhoneFromJid(remoteJid)
	if phone == "" {
		return
	}

	// Get or create contact
	contactID := s.getOrCreateContact(instance.CompanyID, phone, data, instanceName)

	// Get or create conversation
	conversationID := s.getOrCreateConversation(instance.CompanyID, contactID, instance.ChannelID)

	// Extract message content
	message, _ := data["message"].(map[string]interface{})
	msgType, content, mediaURL := extractMessageContent(message)
	log.Printf("[WEBHOOK] Incoming message for conversation %s from %s on %s: type=%s content=%q", conversationID, phone, instanceName, msgType, content)

	// Save message
	msgID := uuid.New().String()
	_, err = s.db.Exec(`
		INSERT INTO messages (id, conversation_id, company_id, sender_type, sender_id, content, message_type, media_url, external_id, status)
		VALUES ($1, $2, $3, 'contact', $4, $5, $6, $7, $8, 'delivered')
	`, msgID, conversationID, instance.CompanyID, contactID, content, msgType, mediaURL, messageID)
	if err != nil {
		log.Printf("Failed to save message: %v", err)
		return
	}

	// Update conversation
	preview := content
	if len(preview) > 100 {
		preview = preview[:100]
	}
	s.db.Exec(`
		UPDATE conversations SET last_message_at = NOW(), last_message_preview = $1, unread_count = unread_count + 1, updated_at = NOW()
		WHERE id = $2
	`, preview, conversationID)

	// Broadcast via WebSocket
	s.wsHub.BroadcastToCompany(instance.CompanyID, websocket.EventNewMessage, map[string]interface{}{
		"id":              msgID,
		"conversation_id": conversationID,
		"sender_type":     "contact",
		"content":         content,
		"message_type":    msgType,
		"media_url":       mediaURL,
		"created_at":      time.Now(),
	})

	s.wsHub.BroadcastToCompany(instance.CompanyID, websocket.EventConversationUpdate, map[string]interface{}{
		"id":                   conversationID,
		"last_message_preview": preview,
		"last_message_at":      time.Now(),
	})

	// Trigger bot if applicable (only for contact messages)
	if s.glpiFlow != nil {
		glpiConversationID := s.glpiFlow.ActiveConversationForMessage(instance.CompanyID, conversationID, contactID)
		if glpiConversationID != "" {
			go s.glpiFlow.HandleGLPIMessage(instance.CompanyID, glpiConversationID, contactID, instanceName, phone, content)
			return
		}
	}

	if s.botEngine != nil {
		channelID := ""
		if instance.ChannelID != nil {
			channelID = *instance.ChannelID
		}
		go s.botEngine.TriggerBot(instance.CompanyID, conversationID, contactID, channelID, content, instanceName, phone)
	}
}

func (s *EvolutionService) handleMessageUpdate(instanceName string, event map[string]interface{}) {
	data, _ := event["data"].(map[string]interface{})

	// v2.4.0 format: data may contain key directly or nested
	var messageID string
	var status string

	// Try nested format
	if key, ok := data["key"].(map[string]interface{}); ok {
		messageID, _ = key["id"].(string)
	}

	// Try direct format (v2.4.0)
	if messageID == "" {
		if id, ok := data["id"].(string); ok {
			messageID = id
		}
		if keyData, ok := data["keyId"].(string); ok {
			messageID = keyData
		}
	}

	if messageID == "" {
		log.Printf("[WEBHOOK] Message update: no message ID found")
		return
	}

	// Determine status
	if update, ok := data["update"].(map[string]interface{}); ok {
		if s, ok := update["status"].(string); ok {
			switch s {
			case "DELIVERY_ACK", "delivery_ack", "DELIVERED":
				status = "delivered"
			case "READ", "read":
				status = "read"
			case "PLAYED", "played":
				status = "read"
			}
		}
		// Also check numeric status (Evolution v2.4.0)
		if statusNum, ok := update["status"].(float64); ok {
			switch int(statusNum) {
			case 2:
				status = "delivered"
			case 3:
				status = "read"
			case 4:
				status = "read" // played
			}
		}
	}

	// Also check top level status field
	if status == "" {
		if s, ok := data["status"].(string); ok {
			switch s {
			case "DELIVERY_ACK", "delivery_ack", "DELIVERED":
				status = "delivered"
			case "READ", "read":
				status = "read"
			case "PLAYED", "played":
				status = "read"
			}
		}
		if statusNum, ok := data["status"].(float64); ok {
			switch int(statusNum) {
			case 2:
				status = "delivered"
			case 3:
				status = "read"
			case 4:
				status = "read"
			}
		}
	}

	if status == "" {
		return
	}

	log.Printf("[WEBHOOK] Message status update: %s -> %s", messageID, status)

	// Update in database
	s.db.Exec("UPDATE messages SET status = $1 WHERE external_id = $2", status, messageID)

	// Broadcast status update via WebSocket
	var companyID, conversationID string
	s.db.QueryRow("SELECT company_id, conversation_id FROM messages WHERE external_id = $1", messageID).Scan(&companyID, &conversationID)

	if companyID != "" {
		s.wsHub.BroadcastToCompany(companyID, websocket.EventMessageStatus, map[string]interface{}{
			"external_id":     messageID,
			"conversation_id": conversationID,
			"status":          status,
		})
	}
}

func (s *EvolutionService) handleQRCodeUpdate(instanceName string, event map[string]interface{}) {
	data, _ := event["data"].(map[string]interface{})
	qrcode, _ := data["qrcode"].(map[string]interface{})
	base64, _ := qrcode["base64"].(string)

	s.db.Exec("UPDATE whatsapp_instances SET qrcode = $1, status = 'qr_code', updated_at = NOW() WHERE instance_name = $2", base64, instanceName)

	var companyID string
	s.db.QueryRow("SELECT company_id FROM whatsapp_instances WHERE instance_name = $1", instanceName).Scan(&companyID)

	if companyID != "" {
		s.wsHub.BroadcastToCompany(companyID, "qrcode_update", map[string]string{
			"instance_name": instanceName,
			"qrcode":        base64,
		})
	}
}

func (s *EvolutionService) handlePresenceUpdate(instanceName string, event map[string]interface{}) {
	data, _ := event["data"].(map[string]interface{})

	// Extract participant JID
	participant, _ := data["id"].(string)
	if participant == "" {
		participant, _ = data["participant"].(string)
	}

	presence, _ := data["presence"].(string)
	// presence can be: "composing", "recording", "paused", "available", "unavailable"

	phone := extractPhoneFromJid(participant)
	if phone == "" {
		return
	}

	// Get company ID
	var companyID string
	s.db.QueryRow("SELECT company_id FROM whatsapp_instances WHERE instance_name = $1", instanceName).Scan(&companyID)
	if companyID == "" {
		return
	}

	// Find contact and their active conversation
	var contactID, conversationID string
	s.db.QueryRow("SELECT id FROM contacts WHERE company_id = $1 AND phone = $2", companyID, phone).Scan(&contactID)
	if contactID == "" {
		return
	}
	s.db.QueryRow("SELECT id FROM conversations WHERE company_id = $1 AND contact_id = $2 AND status != 'resolved' ORDER BY created_at DESC LIMIT 1", companyID, contactID).Scan(&conversationID)

	isTyping := presence == "composing" || presence == "recording"

	s.wsHub.BroadcastToCompany(companyID, "typing", map[string]interface{}{
		"conversation_id": conversationID,
		"contact_phone":   phone,
		"is_typing":       isTyping,
		"is_recording":    presence == "recording",
	})
}

func (s *EvolutionService) getOrCreateContact(companyID, phone string, data map[string]interface{}, instanceName string) string {
	var contactID string
	err := s.db.QueryRow("SELECT id FROM contacts WHERE company_id = $1 AND phone = $2", companyID, phone).Scan(&contactID)
	if err == nil {
		log.Printf("[CONTACT] Found existing contact %s for phone %s", contactID, phone)
		// If no avatar yet, fetch in background
		var avatarURL *string
		s.db.QueryRow("SELECT avatar_url FROM contacts WHERE id = $1", contactID).Scan(&avatarURL)
		if avatarURL == nil || *avatarURL == "" {
			go s.fetchAndSaveContactPhoto(instanceName, phone, contactID)
		}
		return contactID
	}

	// Extract name from push name
	pushName := phone
	if pn, ok := data["pushName"].(string); ok && pn != "" {
		pushName = pn
	}

	contactID = uuid.New().String()
	s.db.Exec(`
		INSERT INTO contacts (id, company_id, name, phone, origin)
		VALUES ($1, $2, $3, $4, 'whatsapp')
	`, contactID, companyID, pushName, phone)

	log.Printf("[CONTACT] Created new contact %s for phone %s", contactID, phone)

	// Fetch profile photo in background
	go s.fetchAndSaveContactPhoto(instanceName, phone, contactID)

	return contactID
}

func (s *EvolutionService) fetchAndSaveContactPhoto(instanceName, phone, contactID string) {
	if instanceName == "" {
		return
	}
	_, err := s.SyncContactPhoto(instanceName, phone, contactID)
	if err != nil {
		log.Printf("[PHOTO] Could not fetch photo for %s: %v", phone, err)
	}
}

func (s *EvolutionService) getOrCreateConversation(companyID, contactID string, channelID *string) string {
	var conversationID string

	// First, try to find an active conversation for this contact in the same channel.
	// Different WhatsApp instances/channels must not steal each other's waiting bot execution.
	if channelID != nil && *channelID != "" {
		err := s.db.QueryRow(`
			SELECT id FROM conversations
			WHERE company_id = $1
				AND contact_id = $2
				AND channel_id = $3
				AND status IN ('open', 'pending', 'in_progress')
			ORDER BY last_message_at DESC NULLS LAST, created_at DESC
			LIMIT 1
		`, companyID, contactID, *channelID).Scan(&conversationID)
		if err == nil {
			log.Printf("[CONVERSATION] Found existing conversation %s for contact %s on channel %s", conversationID, contactID, *channelID)
			return conversationID
		}
	} else {
		err := s.db.QueryRow(`
			SELECT id FROM conversations
			WHERE company_id = $1
				AND contact_id = $2
				AND channel_id IS NULL
				AND status IN ('open', 'pending', 'in_progress')
			ORDER BY last_message_at DESC NULLS LAST, created_at DESC
			LIMIT 1
		`, companyID, contactID).Scan(&conversationID)
		if err == nil {
			log.Printf("[CONVERSATION] Found existing conversation %s for contact %s without channel", conversationID, contactID)
			return conversationID
		}
	}

	// No active conversation found, create a new one
	conversationID = uuid.New().String()
	if channelID != nil {
		s.db.Exec(`
			INSERT INTO conversations (id, company_id, contact_id, channel_id, status, last_message_at)
			VALUES ($1, $2, $3, $4, 'open', NOW())
		`, conversationID, companyID, contactID, *channelID)
	} else {
		s.db.Exec(`
			INSERT INTO conversations (id, company_id, contact_id, status, last_message_at)
			VALUES ($1, $2, $3, 'open', NOW())
		`, conversationID, companyID, contactID)
	}

	log.Printf("[CONVERSATION] Created new conversation %s for contact %s", conversationID, contactID)
	return conversationID
}

// GetInstances returns all instances for a company
func (s *EvolutionService) GetInstances(companyID string) ([]models.WhatsAppInstance, error) {
	rows, err := s.db.Query(`
		SELECT id, company_id, channel_id, instance_name, instance_id, token,
		       COALESCE(status, 'disconnected'),
		       phone_number, webhook_url, connected_at,
		       COALESCE(created_at, NOW()), COALESCE(updated_at, NOW())
		FROM whatsapp_instances
		WHERE company_id = $1
		ORDER BY COALESCE(created_at, NOW()) DESC
	`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instances []models.WhatsAppInstance
	for rows.Next() {
		var inst models.WhatsAppInstance
		err := rows.Scan(&inst.ID, &inst.CompanyID, &inst.ChannelID, &inst.InstanceName,
			&inst.InstanceID, &inst.Token, &inst.Status, &inst.PhoneNumber,
			&inst.WebhookURL, &inst.ConnectedAt, &inst.CreatedAt, &inst.UpdatedAt)
		if err != nil {
			continue
		}
		instances = append(instances, inst)
	}

	return instances, nil
}

// Helper functions
func mapEvolutionStatus(state string) string {
	switch state {
	case "open":
		return "connected"
	case "connecting":
		return "connecting"
	case "close":
		return "disconnected"
	default:
		return "disconnected"
	}
}

func extractPhoneFromJid(jid string) string {
	// JID format: 5511999999999@s.whatsapp.net
	if len(jid) == 0 {
		return ""
	}
	parts := splitJid(jid)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func splitJid(jid string) []string {
	result := []string{}
	current := ""
	for _, c := range jid {
		if c == '@' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func extractMessageContent(message map[string]interface{}) (msgType, content, mediaURL string) {
	if message == nil {
		return "text", "", ""
	}

	if conv, ok := message["conversation"].(string); ok {
		return "text", conv, ""
	}

	if ext, ok := message["extendedTextMessage"].(map[string]interface{}); ok {
		text, _ := ext["text"].(string)
		return "text", text, ""
	}

	if content := extractInteractiveResponseContent(message); content != "" {
		return "text", content, ""
	}

	if img, ok := message["imageMessage"].(map[string]interface{}); ok {
		caption, _ := img["caption"].(string)
		url, _ := img["url"].(string)
		mimetype, _ := img["mimetype"].(string)
		if strings.Contains(strings.ToLower(mimetype), "gif") {
			return "gif", caption, url
		}
		return "image", caption, url
	}

	if audio, ok := message["audioMessage"].(map[string]interface{}); ok {
		url, _ := audio["url"].(string)
		return "audio", "", url
	}

	if video, ok := message["videoMessage"].(map[string]interface{}); ok {
		caption, _ := video["caption"].(string)
		url, _ := video["url"].(string)
		if gifPlayback, _ := video["gifPlayback"].(bool); gifPlayback {
			return "gif", caption, url
		}
		return "video", caption, url
	}

	if doc, ok := message["documentMessage"].(map[string]interface{}); ok {
		fileName, _ := doc["fileName"].(string)
		url, _ := doc["url"].(string)
		return "document", fileName, url
	}

	if sticker, ok := message["stickerMessage"].(map[string]interface{}); ok {
		url, _ := sticker["url"].(string)
		mimetype, _ := sticker["mimetype"].(string)
		if strings.Contains(strings.ToLower(mimetype), "gif") {
			return "gif", "", url
		}
		return "sticker", "", url
	}

	return "text", "", ""
}

func extractInteractiveResponseContent(message map[string]interface{}) string {
	if response, ok := message["buttonsResponseMessage"].(map[string]interface{}); ok {
		return firstNonEmptyString(response, "selectedButtonId", "selectedDisplayText", "displayText")
	}

	if response, ok := message["templateButtonReplyMessage"].(map[string]interface{}); ok {
		return firstNonEmptyString(response, "selectedId", "selectedDisplayText", "displayText")
	}

	if response, ok := message["listResponseMessage"].(map[string]interface{}); ok {
		if reply, ok := response["singleSelectReply"].(map[string]interface{}); ok {
			if value := firstNonEmptyString(reply, "selectedRowId", "title", "description"); value != "" {
				return value
			}
		}
		return firstNonEmptyString(response, "title", "description")
	}

	if response, ok := message["interactiveResponseMessage"].(map[string]interface{}); ok {
		if body, ok := response["body"].(map[string]interface{}); ok {
			if value := firstNonEmptyString(body, "text"); value != "" {
				return value
			}
		}
		if native, ok := response["nativeFlowResponseMessage"].(map[string]interface{}); ok {
			if value := firstNonEmptyString(native, "name"); value != "" {
				return value
			}
			if rawParams := firstNonEmptyString(native, "paramsJson"); rawParams != "" {
				var params map[string]interface{}
				if err := json.Unmarshal([]byte(rawParams), &params); err == nil {
					if value := firstNonEmptyString(params, "id", "selectedId", "selectedRowId", "display_text", "title"); value != "" {
						return value
					}
				}
			}
		}
	}

	return ""
}

func firstNonEmptyString(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := data[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (s *EvolutionService) isKnownOutgoingMessage(messageID string) bool {
	if strings.TrimSpace(messageID) == "" {
		return false
	}

	var exists bool
	err := s.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM messages
			WHERE external_id = $1
				AND sender_type IN ('bot', 'agent', 'user')
		)
	`, messageID).Scan(&exists)
	return err == nil && exists
}

func splitOnce(s, sep string) []string {
	for i := range s {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			return []string{s[:i], s[i+len(sep):]}
		}
	}
	return []string{s}
}
