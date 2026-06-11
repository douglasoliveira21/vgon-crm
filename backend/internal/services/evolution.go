package services

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/evocrm/backend/internal/config"
	"github.com/evocrm/backend/internal/models"
	"github.com/evocrm/backend/internal/websocket"
	"github.com/google/uuid"
)

type EvolutionService struct {
	cfg    *config.Config
	db     *sql.DB
	wsHub  *websocket.Hub
	client *http.Client
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
	payload := map[string]interface{}{
		"number": phone,
		"text":   text,
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

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if key, ok := result["key"].(map[string]interface{}); ok {
		if id, ok := key["id"].(string); ok {
			return id, nil
		}
	}

	return "", nil
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

	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	if key, ok := result["key"].(map[string]interface{}); ok {
		if id, ok := key["id"].(string); ok {
			return id, nil
		}
	}

	return "", nil
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

	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	if key, ok := result["key"].(map[string]interface{}); ok {
		if id, ok := key["id"].(string); ok {
			return id, nil
		}
	}

	return "", nil
}

// HandleWebhook processes incoming webhook events from Evolution API
func (s *EvolutionService) HandleWebhook(instanceName string, event map[string]interface{}) {
	eventType, _ := event["event"].(string)

	switch eventType {
	case "connection.update":
		s.handleConnectionUpdate(instanceName, event)
	case "messages.upsert":
		s.handleMessageUpsert(instanceName, event)
	case "messages.update":
		s.handleMessageUpdate(instanceName, event)
	case "qrcode.updated":
		s.handleQRCodeUpdate(instanceName, event)
	default:
		log.Printf("Unhandled webhook event: %s for instance: %s", eventType, instanceName)
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

	// Extract phone number from JID
	phone := extractPhoneFromJid(remoteJid)
	if phone == "" {
		return
	}

	// Get or create contact
	contactID := s.getOrCreateContact(instance.CompanyID, phone, data)

	// Get or create conversation
	conversationID := s.getOrCreateConversation(instance.CompanyID, contactID, instance.ChannelID)

	// Extract message content
	message, _ := data["message"].(map[string]interface{})
	msgType, content, mediaURL := extractMessageContent(message)

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
}

func (s *EvolutionService) handleMessageUpdate(instanceName string, event map[string]interface{}) {
	data, _ := event["data"].(map[string]interface{})
	key, _ := data["key"].(map[string]interface{})
	messageID, _ := key["id"].(string)
	
	update, _ := data["update"].(map[string]interface{})
	status := "sent"
	if s, ok := update["status"].(string); ok {
		switch s {
		case "DELIVERY_ACK":
			status = "delivered"
		case "READ":
			status = "read"
		case "PLAYED":
			status = "read"
		}
	}

	s.db.Exec("UPDATE messages SET status = $1 WHERE external_id = $2", status, messageID)
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

func (s *EvolutionService) getOrCreateContact(companyID, phone string, data map[string]interface{}) string {
	var contactID string
	err := s.db.QueryRow("SELECT id FROM contacts WHERE company_id = $1 AND phone = $2", companyID, phone).Scan(&contactID)
	if err == nil {
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

	return contactID
}

func (s *EvolutionService) getOrCreateConversation(companyID, contactID string, channelID *string) string {
	var conversationID string
	err := s.db.QueryRow(`
		SELECT id FROM conversations 
		WHERE company_id = $1 AND contact_id = $2 AND status IN ('open', 'pending')
		ORDER BY created_at DESC LIMIT 1
	`, companyID, contactID).Scan(&conversationID)
	if err == nil {
		return conversationID
	}

	conversationID = uuid.New().String()
	s.db.Exec(`
		INSERT INTO conversations (id, company_id, contact_id, channel_id, status, last_message_at)
		VALUES ($1, $2, $3, $4, 'open', NOW())
	`, conversationID, companyID, contactID, channelID)

	return conversationID
}

// GetInstances returns all instances for a company
func (s *EvolutionService) GetInstances(companyID string) ([]models.WhatsAppInstance, error) {
	rows, err := s.db.Query(`
		SELECT id, company_id, channel_id, instance_name, instance_id, token, status, phone_number, webhook_url, connected_at, created_at, updated_at
		FROM whatsapp_instances
		WHERE company_id = $1
		ORDER BY created_at DESC
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

	if img, ok := message["imageMessage"].(map[string]interface{}); ok {
		caption, _ := img["caption"].(string)
		url, _ := img["url"].(string)
		return "image", caption, url
	}

	if audio, ok := message["audioMessage"].(map[string]interface{}); ok {
		url, _ := audio["url"].(string)
		return "audio", "", url
	}

	if video, ok := message["videoMessage"].(map[string]interface{}); ok {
		caption, _ := video["caption"].(string)
		url, _ := video["url"].(string)
		return "video", caption, url
	}

	if doc, ok := message["documentMessage"].(map[string]interface{}); ok {
		fileName, _ := doc["fileName"].(string)
		url, _ := doc["url"].(string)
		return "document", fileName, url
	}

	if sticker, ok := message["stickerMessage"].(map[string]interface{}); ok {
		url, _ := sticker["url"].(string)
		return "sticker", "", url
	}

	return "text", "", ""
}
