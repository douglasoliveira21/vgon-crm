package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sort"
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

type BotEdge struct {
	ID           string `json:"id"`
	Source       string `json:"source"`
	Target       string `json:"target"`
	SourceHandle string `json:"sourceHandle"`
	TargetHandle string `json:"targetHandle"`
	Label        string `json:"label"`
}

func parseBotNodes(raw json.RawMessage) ([]BotNode, error) {
	var nodes []BotNode
	if len(raw) == 0 {
		return nodes, nil
	}
	if err := json.Unmarshal(raw, &nodes); err == nil {
		return nodes, nil
	}

	var encoded string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return nil, err
	}
	if encoded == "" {
		return nodes, nil
	}
	if err := json.Unmarshal([]byte(encoded), &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

func parseBotEdges(raw json.RawMessage) ([]BotEdge, error) {
	var edges []BotEdge
	if len(raw) == 0 {
		return edges, nil
	}
	if err := json.Unmarshal(raw, &edges); err == nil {
		return edges, nil
	}

	var encoded string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return nil, err
	}
	if encoded == "" {
		return edges, nil
	}
	if err := json.Unmarshal([]byte(encoded), &edges); err != nil {
		return nil, err
	}
	return edges, nil
}

func getNodeType(node BotNode) string {
	if nt, ok := node.Data["nodeType"].(string); ok && nt != "" {
		return nt
	}
	return node.Type
}

func getNodeConfig(node BotNode) map[string]interface{} {
	if config, ok := node.Data["config"].(map[string]interface{}); ok && config != nil {
		return config
	}
	return node.Data
}

func getNumber(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		n, err := v.Float64()
		return n, err == nil
	default:
		return 0, false
	}
}

func getStringList(value interface{}) []string {
	switch items := value.(type) {
	case []string:
		return items
	case []interface{}:
		values := make([]string, 0, len(items))
		for _, item := range items {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				values = append(values, strings.TrimSpace(text))
			}
		}
		return values
	default:
		return nil
	}
}

func toInterfaceList(values []string) []interface{} {
	items := make([]interface{}, 0, len(values))
	for _, value := range values {
		items = append(items, value)
	}
	return items
}

func appendOptionsToMessage(message string, options []string) string {
	if len(options) == 0 {
		return message
	}
	var builder strings.Builder
	builder.WriteString(strings.TrimSpace(message))
	for i, option := range options {
		builder.WriteString(fmt.Sprintf("\n%d. %s", i+1, option))
	}
	return builder.String()
}

func normalizeTriggerType(triggerType string) string {
	if strings.HasPrefix(triggerType, "trigger_") {
		return triggerType
	}
	switch triggerType {
	case "new_conversation", "keyword", "off_hours", "tag_added":
		return "trigger_" + triggerType
	default:
		return triggerType
	}
}

func getTriggerConfigString(nodes []BotNode, triggerType, key string) string {
	triggerType = normalizeTriggerType(triggerType)
	for _, node := range nodes {
		if getNodeType(node) != triggerType {
			continue
		}
		if value, ok := getNodeConfig(node)[key].(string); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func triggerMatchesChannel(nodes []BotNode, channelID string) bool {
	configuredChannelID := getTriggerConfigString(nodes, "trigger_inbox_message", "channel_id")
	if configuredChannelID == "" {
		return true
	}
	return channelID != "" && configuredChannelID == channelID
}

func buildNextNodeMap(edges []BotEdge) map[string]string {
	sort.SliceStable(edges, func(i, j int) bool {
		if edges[i].Source != edges[j].Source {
			return edges[i].Source < edges[j].Source
		}
		if edges[i].SourceHandle != edges[j].SourceHandle {
			return edges[i].SourceHandle < edges[j].SourceHandle
		}
		if edges[i].Label != edges[j].Label {
			return edges[i].Label < edges[j].Label
		}
		return edges[i].ID < edges[j].ID
	})

	nextBySource := make(map[string]string, len(edges))
	for _, edge := range edges {
		if edge.Source == "" || edge.Target == "" {
			continue
		}
		if _, exists := nextBySource[edge.Source]; !exists {
			nextBySource[edge.Source] = edge.Target
		}
	}
	return nextBySource
}

func findStartNode(nodes []BotNode, edges []BotEdge, triggerType string) *BotNode {
	triggerType = normalizeTriggerType(triggerType)
	incoming := map[string]bool{}
	for _, edge := range edges {
		if edge.Target != "" {
			incoming[edge.Target] = true
		}
	}

	for i := range nodes {
		if getNodeType(nodes[i]) == triggerType {
			return &nodes[i]
		}
	}
	for i := range nodes {
		if strings.HasPrefix(getNodeType(nodes[i]), "trigger") {
			return &nodes[i]
		}
	}
	for i := range nodes {
		if !incoming[nodes[i].ID] {
			return &nodes[i]
		}
	}
	return &nodes[0]
}

func findNextNodeByPosition(current BotNode, nodes []BotNode, visited map[string]bool) *BotNode {
	currentX := current.Position["x"]
	currentY := current.Position["y"]
	bestScore := 1.0e18
	var best *BotNode

	for i := range nodes {
		candidate := nodes[i]
		if candidate.ID == current.ID || visited[candidate.ID] || strings.HasPrefix(getNodeType(candidate), "trigger") {
			continue
		}

		dx := candidate.Position["x"] - currentX
		dy := candidate.Position["y"] - currentY
		score := dx*dx + dy*dy
		if dx < -40 {
			score += 1000000
		}
		if score < bestScore {
			bestScore = score
			best = &nodes[i]
		}
	}

	return best
}

// TriggerBot checks if any bot flow should be triggered for a new message
func (e *BotEngine) TriggerBot(companyID, conversationID, contactID, channelID, message, instanceName, phone string) {
	// Find active flows for this company
	rows, err := e.db.Query(`
		SELECT id, trigger_type, trigger_value, nodes, edges
		FROM bot_flows
		WHERE company_id = $1 AND is_active = true
		ORDER BY created_at ASC
	`, companyID)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var flowID, triggerType string
		var triggerValue *string
		var nodesJSON, edgesJSON json.RawMessage

		if err := rows.Scan(&flowID, &triggerType, &triggerValue, &nodesJSON, &edgesJSON); err != nil {
			continue
		}

		nodes, err := parseBotNodes(nodesJSON)
		if err != nil || len(nodes) == 0 {
			log.Printf("[BOT] Failed to parse nodes for flow %s: %v", flowID, err)
			continue
		}

		// Check if this flow should trigger
		shouldTrigger := false

		switch triggerType {
		case "new_conversation", "trigger_new_conversation":
			// Check if this is the first message in conversation
			var msgCount int
			e.db.QueryRow("SELECT COUNT(*) FROM messages WHERE conversation_id = $1", conversationID).Scan(&msgCount)
			shouldTrigger = msgCount <= 1

		case "keyword", "trigger_keyword":
			keywordsValue := ""
			if triggerValue != nil {
				keywordsValue = *triggerValue
			}
			if keywordsValue == "" {
				keywordsValue = getTriggerConfigString(nodes, "trigger_keyword", "keywords")
			}
			if keywordsValue != "" {
				keywords := strings.Split(keywordsValue, ",")
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
			shouldTrigger = triggerMatchesChannel(nodes, channelID)

		case "tag_added", "trigger_tag_added":
			shouldTrigger = false
		}

		if shouldTrigger {
			if !e.reserveExecution(flowID, conversationID, contactID) {
				continue
			}

			go e.executeFlow(flowID, triggerType, companyID, conversationID, contactID, instanceName, phone, nodesJSON, edgesJSON)
			return // Only trigger first matching flow
		}
	}
}

// executeFlow runs a bot flow
func (e *BotEngine) executeFlow(flowID, triggerType, companyID, conversationID, contactID, instanceName, phone string, nodesJSON, edgesJSON json.RawMessage) {
	nodes, err := parseBotNodes(nodesJSON)
	if err != nil {
		log.Printf("[BOT] Failed to parse nodes for flow %s: %v", flowID, err)
		e.finishLatestExecution(flowID, conversationID, "error")
		return
	}
	if len(nodes) == 0 {
		e.finishLatestExecution(flowID, conversationID, "completed")
		return
	}

	edges, err := parseBotEdges(edgesJSON)
	if err != nil {
		log.Printf("[BOT] Failed to parse edges for flow %s: %v", flowID, err)
	}

	var execID string
	err = e.db.QueryRow(`
		SELECT id
		FROM bot_executions
		WHERE flow_id = $1 AND conversation_id = $2 AND status = 'running'
		ORDER BY started_at DESC
		LIMIT 1
	`, flowID, conversationID).Scan(&execID)
	if err != nil {
		log.Printf("[BOT] Missing running execution for flow %s conversation %s", flowID, conversationID)
		return
	}

	log.Printf("[BOT] Starting flow %s for conversation %s", flowID, conversationID)

	nodesByID := make(map[string]BotNode, len(nodes))
	for _, node := range nodes {
		nodesByID[node.ID] = node
	}

	nextBySource := buildNextNodeMap(edges)
	current := findStartNode(nodes, edges, triggerType)
	visited := map[string]bool{}
	steps := 0
	maxSteps := len(nodes) + len(edges) + 5

	for current != nil && steps < maxSteps {
		steps++
		node := *current
		if visited[node.ID] {
			log.Printf("[BOT] Stopping flow %s because node %s would repeat", flowID, node.ID)
			break
		}
		visited[node.ID] = true

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
		e.db.Exec(`
			UPDATE bot_executions
			SET context = COALESCE(context, '{}'::jsonb) || jsonb_build_object('current_node_id', $1)
			WHERE id = $2
		`, node.ID, execID)

		nextID := nextBySource[node.ID]
		if nextID == "" {
			nextNode := findNextNodeByPosition(node, nodes, visited)
			if nextNode == nil {
				break
			}
			log.Printf("[BOT] Flow %s using position fallback from node %s to node %s", flowID, node.ID, nextNode.ID)
			current = nextNode
			time.Sleep(1 * time.Second)
			continue
		}
		nextNode, ok := nodesByID[nextID]
		if !ok {
			log.Printf("[BOT] Flow %s points to missing node %s", flowID, nextID)
			break
		}
		current = &nextNode
		time.Sleep(1 * time.Second)
	}

	// Mark execution as completed
	e.db.Exec("UPDATE bot_executions SET status = 'completed', completed_at = NOW() WHERE id = $1", execID)
	log.Printf("[BOT] Flow %s completed for conversation %s", flowID, conversationID)
}

func (e *BotEngine) reserveExecution(flowID, conversationID, contactID string) bool {
	tx, err := e.db.Begin()
	if err != nil {
		log.Printf("[BOT] Failed to start execution transaction: %v", err)
		return false
	}
	defer tx.Rollback()

	// Serializes bot starts per conversation and avoids duplicate executions when webhooks arrive together.
	if _, err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext($1))", conversationID); err != nil {
		log.Printf("[BOT] Failed to acquire execution lock: %v", err)
		return false
	}

	var execCount int
	if err := tx.QueryRow(`
		SELECT COUNT(*)
		FROM bot_executions
		WHERE conversation_id = $1 AND status = 'running'
	`, conversationID).Scan(&execCount); err != nil {
		log.Printf("[BOT] Failed to check running executions: %v", err)
		return false
	}
	if execCount > 0 {
		return false
	}

	_, err = tx.Exec(`
		INSERT INTO bot_executions (id, flow_id, conversation_id, contact_id, status)
		VALUES ($1, $2, $3, $4, 'running')
	`, uuid.New().String(), flowID, conversationID, contactID)
	if err != nil {
		log.Printf("[BOT] Failed to reserve execution: %v", err)
		return false
	}

	return tx.Commit() == nil
}

func (e *BotEngine) finishLatestExecution(flowID, conversationID, status string) {
	if status == "" {
		status = "completed"
	}
	e.db.Exec(`
		UPDATE bot_executions
		SET status = $1, completed_at = NOW()
		WHERE id = (
			SELECT id
			FROM bot_executions
			WHERE flow_id = $2 AND conversation_id = $3 AND status = 'running'
			ORDER BY started_at DESC
			LIMIT 1
		)
	`, status, flowID, conversationID)
}

// executeNode processes a single node
func (e *BotEngine) executeNode(node BotNode, companyID, conversationID, contactID, instanceName, phone string) error {
	// Get the actual node type from data.nodeType (React Flow format)
	nodeType := getNodeType(node)

	// Get config from data.config
	config := getNodeConfig(node)

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
	case "send_buttons", "send_list":
		msg, _ := config["message"].(string)
		if msg == "" {
			msg = "Escolha uma opcao:"
		}
		if nodeType == "send_buttons" {
			msg = appendOptionsToMessage(msg, getStringList(config["buttons"]))
		} else {
			msg = appendOptionsToMessage(msg, getStringList(config["options"]))
		}
		node.Data["message"] = msg
		return e.nodeSendMessage(node, companyID, conversationID, instanceName, phone)
	case "ask_question", "ask_text", "ask_options", "ask_email", "ask_phone":
		q, _ := config["question"].(string)
		if q == "" {
			q, _ = node.Data["question"].(string)
		}
		if q == "" && nodeType == "ask_email" {
			q = "Qual e o seu e-mail?"
		}
		if q == "" && nodeType == "ask_phone" {
			q = "Qual e o seu telefone?"
		}
		if q != "" {
			node.Data["question"] = q
		}
		if len(getStringList(config["options"])) > 0 {
			node.Data["options"] = toInterfaceList(getStringList(config["options"]))
		}
		return e.nodeAskQuestion(node, companyID, conversationID, instanceName, phone)
	case "condition", "condition_business_hours", "condition_contact_field", "condition_tag":
		return nil
	case "delay", "wait_seconds", "wait_minutes", "wait_hours":
		if secs, ok := getNumber(config["seconds"]); ok {
			node.Data["seconds"] = secs
		}
		if mins, ok := getNumber(config["minutes"]); ok {
			node.Data["seconds"] = mins * 60
		}
		if hrs, ok := getNumber(config["hours"]); ok {
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
		var err error
		externalID, err = e.evo.SendTextMessage(instanceName, phone, message)
		if err != nil {
			log.Printf("[BOT] Failed to send message to %s via %s: %v", phone, instanceName, err)
			return err
		}
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
		var err error
		externalID, err = e.evo.SendTextMessage(instanceName, phone, message)
		if err != nil {
			log.Printf("[BOT] Failed to send question to %s via %s: %v", phone, instanceName, err)
			return err
		}
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
