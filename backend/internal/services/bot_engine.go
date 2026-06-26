package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/evocrm/backend/internal/websocket"
	"github.com/google/uuid"
)

type BotEngine struct {
	db       *sql.DB
	wsHub    *websocket.Hub
	evo      *EvolutionService
	glpiFlow *GLPIFlowEngine
}

func NewBotEngine(db *sql.DB, wsHub *websocket.Hub, evo *EvolutionService) *BotEngine {
	return &BotEngine{db: db, wsHub: wsHub, evo: evo}
}

type BotNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Position map[string]float64     `json:"position"`
	Data     map[string]interface{} `json:"data"`
}

// TriggerBot checks if any bot flow should be triggered for a new message
func (e *BotEngine) TriggerBot(companyID, conversationID, contactID, channelID, message, instanceName, phone string) {
	// Find active flows for this company
	rows, err := e.db.Query(`
		SELECT id, trigger_type, trigger_value, nodes, edges
		FROM bot_flows
		WHERE company_id = $1 AND is_active = true
	`, companyID)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var flowID, triggerType string
		var triggerValue *string
		var nodesJSON, edgesJSON json.RawMessage

		rows.Scan(&flowID, &triggerType, &triggerValue, &nodesJSON, &edgesJSON)

		// Check if this flow should trigger
		shouldTrigger := false

		switch triggerType {
		case "new_conversation", "trigger_new_conversation":
			// Check if this is the first message in conversation
			var msgCount int
			e.db.QueryRow("SELECT COUNT(*) FROM messages WHERE conversation_id = $1", conversationID).Scan(&msgCount)
			shouldTrigger = msgCount <= 1

		case "keyword", "trigger_keyword":
			if triggerValue != nil && *triggerValue != "" {
				keywords := strings.Split(*triggerValue, ",")
				msgLower := strings.ToLower(message)
				for _, kw := range keywords {
					if strings.Contains(msgLower, strings.TrimSpace(strings.ToLower(kw))) {
						shouldTrigger = true
						break
					}
				}
			}

		case "off_hours", "trigger_off_hours":
			hour := time.Now().Hour()
			shouldTrigger = hour < 8 || hour >= 18

		case "trigger_inbox_message":
			// Always trigger for inbox message (channel filtering done later)
			shouldTrigger = true

		case "tag_added", "trigger_tag_added":
			shouldTrigger = false
		}

		if shouldTrigger {
			// Check if bot is already running for this conversation
			var execCount int
			e.db.QueryRow("SELECT COUNT(*) FROM bot_executions WHERE conversation_id = $1 AND status = 'running'", conversationID).Scan(&execCount)
			if execCount > 0 {
				continue // Don't start another bot
			}

			go e.executeFlow(flowID, companyID, conversationID, contactID, instanceName, phone, nodesJSON)
			return // Only trigger first matching flow
		}
	}
}

// executeFlow runs a bot flow
func (e *BotEngine) executeFlow(flowID, companyID, conversationID, contactID, instanceName, phone string, nodesJSON json.RawMessage) {
	var nodes []BotNode

	// Try direct unmarshal
	if err := json.Unmarshal(nodesJSON, &nodes); err != nil {
		// Might be a double-encoded string - try unwrapping
		var nodesStr string
		if err2 := json.Unmarshal(nodesJSON, &nodesStr); err2 == nil {
			if err3 := json.Unmarshal([]byte(nodesStr), &nodes); err3 != nil {
				log.Printf("[BOT] Failed to parse nodes for flow %s: %v", flowID, err3)
				return
			}
		} else {
			log.Printf("[BOT] Failed to parse nodes for flow %s: %v", flowID, err)
			return
		}
	}

	if len(nodes) == 0 {
		return
	}

	// Create execution record
	execID := uuid.New().String()
	e.db.Exec(`
		INSERT INTO bot_executions (id, flow_id, conversation_id, contact_id, status)
		VALUES ($1, $2, $3, $4, 'running')
	`, execID, flowID, conversationID, contactID)

	log.Printf("[BOT] Starting flow %s for conversation %s", flowID, conversationID)

	// Execute nodes sequentially
	for i, node := range nodes {
		// Check if bot was paused (human took over)
		var status string
		e.db.QueryRow("SELECT status FROM bot_executions WHERE id = $1", execID).Scan(&status)
		if status != "running" {
			log.Printf("[BOT] Flow %s stopped (status: %s)", flowID, status)
			return
		}

		err := e.executeNode(node, companyID, conversationID, contactID, instanceName, phone)
		if err != nil {
			log.Printf("[BOT] Error executing node %s: %v", node.ID, err)
			break
		}

		// Update current node
		e.db.Exec("UPDATE bot_executions SET current_node_id = $1 WHERE id = $2", node.ID, execID)

		// Small delay between nodes
		if i < len(nodes)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	// Mark execution as completed
	e.db.Exec("UPDATE bot_executions SET status = 'completed', completed_at = NOW() WHERE id = $1", execID)
	log.Printf("[BOT] Flow %s completed for conversation %s", flowID, conversationID)
}

// executeNode processes a single node
func (e *BotEngine) executeNode(node BotNode, companyID, conversationID, contactID, instanceName, phone string) error {
	// Get the actual node type from data.nodeType (React Flow format)
	nodeType := node.Type
	if nt, ok := node.Data["nodeType"].(string); ok && nt != "" {
		nodeType = nt
	}

	// Get config from data.config
	config, _ := node.Data["config"].(map[string]interface{})
	if config == nil {
		config = node.Data
	}

	switch nodeType {
	case "send_message", "send_text":
		msg, _ := config["message"].(string)
		if msg == "" {
			msg, _ = node.Data["message"].(string)
		}
		if msg != "" {
			node.Data["message"] = msg
		}
		return e.nodeSendMessage(node, companyID, conversationID, instanceName, phone)
	case "ask_question", "ask_text", "ask_options":
		q, _ := config["question"].(string)
		if q == "" {
			q, _ = node.Data["question"].(string)
		}
		if q != "" {
			node.Data["question"] = q
		}
		return e.nodeAskQuestion(node, companyID, conversationID, instanceName, phone)
	case "delay", "wait_seconds", "wait_minutes", "wait_hours":
		if secs, ok := config["seconds"].(float64); ok {
			node.Data["seconds"] = secs
		}
		if mins, ok := config["minutes"].(float64); ok {
			node.Data["seconds"] = mins * 60
		}
		if hrs, ok := config["hours"].(float64); ok {
			node.Data["seconds"] = hrs * 3600
		}
		return e.nodeDelay(node)
	case "add_tag", "action_add_tag":
		tn, _ := config["tag_name"].(string)
		if tn != "" {
			node.Data["tag_name"] = tn
		}
		return e.nodeAddTag(node, companyID, contactID)
	case "transfer_team", "action_transfer_team":
		tm, _ := config["team_name"].(string)
		if tm != "" {
			node.Data["team_name"] = tm
		}
		return e.nodeTransferTeam(node, companyID, conversationID)
	case "call_webhook", "action_webhook":
		return e.nodeCallWebhook(node)
	case "glpi_open_ticket":
		return e.nodeGLPIOpenTicket(companyID, conversationID, contactID, instanceName, phone)
	case "glpi_check_status":
		return e.nodeGLPICheckStatus(companyID, conversationID, contactID, instanceName, phone)
	case "end", "action_close_conversation":
		cc, _ := config["close_conversation"].(bool)
		node.Data["close_conversation"] = cc
		return e.nodeEnd(node, companyID, conversationID)
	default:
		log.Printf("[BOT] Unknown node type: %s", nodeType)
		return nil
	}
}

func (e *BotEngine) nodeSendMessage(node BotNode, companyID, conversationID, instanceName, phone string) error {
	message, _ := node.Data["message"].(string)
	if message == "" {
		if cfg, ok := node.Data["config"].(map[string]interface{}); ok {
			message, _ = cfg["message"].(string)
		}
	}
	if message == "" {
		return nil
	}

	// Replace variables
	message = e.replaceVariables(message, companyID, conversationID, phone)

	// Send via WhatsApp
	var externalID string
	if instanceName != "" && phone != "" {
		externalID, _ = e.evo.SendTextMessage(instanceName, phone, message)
		log.Printf("[BOT] Sent message to %s via %s", phone, instanceName)
	} else {
		log.Printf("[BOT] Cannot send - instanceName='%s' phone='%s'", instanceName, phone)
	}

	// Save message
	msgID := uuid.New().String()
	e.db.Exec(`
		INSERT INTO messages (id, conversation_id, company_id, sender_type, content, message_type, external_id, status)
		VALUES ($1, $2, $3, 'bot', $4, 'text', $5, 'sent')
	`, msgID, conversationID, companyID, message, externalID)

	e.db.Exec("UPDATE conversations SET last_message_at = NOW(), last_message_preview = $1 WHERE id = $2", message, conversationID)

	e.wsHub.BroadcastToCompany(companyID, "new_message", map[string]interface{}{
		"id": msgID, "conversation_id": conversationID,
		"sender_type": "bot", "content": message, "message_type": "text",
		"sender_name": "Bot",
		"status":      "sent", "created_at": time.Now(),
	})

	return nil
}

func (e *BotEngine) replaceVariables(text, companyID, conversationID, phone string) string {
	var contactName, contactPhone, contactEmail, contactCompany string
	e.db.QueryRow(`
		SELECT COALESCE(co.name, ''), COALESCE(co.phone, ''), COALESCE(co.email, ''), COALESCE(co.company_name, '')
		FROM conversations c JOIN contacts co ON c.contact_id = co.id WHERE c.id = $1
	`, conversationID).Scan(&contactName, &contactPhone, &contactEmail, &contactCompany)

	if contactName == "" {
		contactName = phone
	}

	text = strings.ReplaceAll(text, "{{nome}}", contactName)
	text = strings.ReplaceAll(text, "{{telefone}}", contactPhone)
	text = strings.ReplaceAll(text, "{{email}}", contactEmail)
	text = strings.ReplaceAll(text, "{{empresa}}", contactCompany)
	text = strings.ReplaceAll(text, "{{data_atual}}", time.Now().Format("02/01/2006"))
	return text
}

func (e *BotEngine) nodeAskQuestion(node BotNode, companyID, conversationID, instanceName, phone string) error {
	question, _ := node.Data["question"].(string)
	if question == "" {
		return nil
	}

	// Build message with options
	message := question
	if options, ok := node.Data["options"].([]interface{}); ok && len(options) > 0 {
		message += "\n"
		for i, opt := range options {
			if optStr, ok := opt.(string); ok && optStr != "" {
				message += fmt.Sprintf("\n%d. %s", i+1, optStr)
			}
		}
	}

	// Send via WhatsApp
	var externalID string
	if instanceName != "" && phone != "" {
		externalID, _ = e.evo.SendTextMessage(instanceName, phone, message)
	}

	// Save message
	msgID := uuid.New().String()
	e.db.Exec(`
		INSERT INTO messages (id, conversation_id, company_id, sender_type, content, message_type, external_id, status)
		VALUES ($1, $2, $3, 'bot', $4, 'text', $5, 'sent')
	`, msgID, conversationID, companyID, message, externalID)

	e.wsHub.BroadcastToCompany(companyID, "new_message", map[string]interface{}{
		"id": msgID, "conversation_id": conversationID,
		"sender_type": "bot", "content": message, "message_type": "text",
		"sender_name": "Assistente",
		"status":      "sent", "created_at": time.Now(),
	})

	return nil
}

func (e *BotEngine) nodeDelay(node BotNode) error {
	seconds, _ := node.Data["seconds"].(float64)
	if seconds <= 0 {
		seconds = 5
	}
	time.Sleep(time.Duration(seconds) * time.Second)
	return nil
}

func (e *BotEngine) nodeAddTag(node BotNode, companyID, contactID string) error {
	tagName, _ := node.Data["tag_name"].(string)
	if tagName == "" {
		return nil
	}

	// Find or create tag
	var tagID string
	err := e.db.QueryRow("SELECT id FROM tags WHERE company_id = $1 AND name = $2", companyID, tagName).Scan(&tagID)
	if err != nil {
		tagID = uuid.New().String()
		e.db.Exec("INSERT INTO tags (id, company_id, name) VALUES ($1, $2, $3)", tagID, companyID, tagName)
	}

	// Add to contact
	e.db.Exec("INSERT INTO contact_tags (contact_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", contactID, tagID)

	return nil
}

func (e *BotEngine) nodeTransferTeam(node BotNode, companyID, conversationID string) error {
	teamName, _ := node.Data["team_name"].(string)
	if teamName == "" {
		return nil
	}

	var teamID string
	err := e.db.QueryRow("SELECT id FROM teams WHERE company_id = $1 AND name ILIKE $2", companyID, teamName).Scan(&teamID)
	if err != nil {
		return nil
	}

	e.db.Exec("UPDATE conversations SET team_id = $1, assigned_to = NULL WHERE id = $2", teamID, conversationID)

	// Pause bot when transferring
	e.db.Exec("UPDATE bot_executions SET status = 'paused' WHERE conversation_id = $1 AND status = 'running'", conversationID)

	return nil
}

func (e *BotEngine) nodeCallWebhook(node BotNode) error {
	// TODO: Implement webhook call
	return nil
}

func (e *BotEngine) nodeGLPIOpenTicket(companyID, conversationID, contactID, instanceName, phone string) error {
	if e.glpiFlow != nil {
		e.glpiFlow.StartGLPIFlow(companyID, conversationID, contactID, instanceName, phone, "open_ticket")
		// Pause the bot since GLPI flow will take over the conversation
		e.db.Exec("UPDATE bot_executions SET status = 'paused' WHERE conversation_id = $1 AND status = 'running'", conversationID)
	}
	return nil
}

func (e *BotEngine) nodeGLPICheckStatus(companyID, conversationID, contactID, instanceName, phone string) error {
	if e.glpiFlow != nil {
		e.glpiFlow.StartGLPIFlow(companyID, conversationID, contactID, instanceName, phone, "check_status")
		e.db.Exec("UPDATE bot_executions SET status = 'paused' WHERE conversation_id = $1 AND status = 'running'", conversationID)
	}
	return nil
}

func (e *BotEngine) nodeEnd(node BotNode, companyID, conversationID string) error {
	closeConv, _ := node.Data["close_conversation"].(bool)
	if closeConv {
		e.db.Exec("UPDATE conversations SET status = 'resolved', resolved_at = NOW() WHERE id = $1", conversationID)
	}
	return nil
}

// PauseBotForConversation stops any running bot when a human takes over
func (e *BotEngine) PauseBotForConversation(conversationID string) {
	e.db.Exec("UPDATE bot_executions SET status = 'paused' WHERE conversation_id = $1 AND status = 'running'", conversationID)
}
