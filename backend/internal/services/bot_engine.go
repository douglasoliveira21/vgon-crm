package services

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

func secondsUntilBusinessHours(startTime, endTime string) float64 {
	now := time.Now()
	startHour, startMinute := parseClock(startTime, 8, 0)
	endHour, endMinute := parseClock(endTime, 18, 0)
	start := time.Date(now.Year(), now.Month(), now.Day(), startHour, startMinute, 0, 0, now.Location())
	end := time.Date(now.Year(), now.Month(), now.Day(), endHour, endMinute, 0, 0, now.Location())
	if now.Before(start) {
		return start.Sub(now).Seconds()
	}
	if now.Before(end) {
		return 1
	}
	next := start.Add(24 * time.Hour)
	return next.Sub(now).Seconds()
}

func parseClock(value string, defaultHour, defaultMinute int) (int, int) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 2 {
		return defaultHour, defaultMinute
	}
	var hour, minute int
	if _, err := fmt.Sscanf(parts[0]+":"+parts[1], "%d:%d", &hour, &minute); err != nil {
		return defaultHour, defaultMinute
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return defaultHour, defaultMinute
	}
	return hour, minute
}

func isWithinBusinessHours(startTime, endTime string) bool {
	now := time.Now()
	startHour, startMinute := parseClock(startTime, 8, 0)
	endHour, endMinute := parseClock(endTime, 18, 0)
	start := time.Date(now.Year(), now.Month(), now.Day(), startHour, startMinute, 0, 0, now.Location())
	end := time.Date(now.Year(), now.Month(), now.Day(), endHour, endMinute, 0, 0, now.Location())
	return !now.Before(start) && now.Before(end)
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

func buildOutgoingEdges(edges []BotEdge) map[string][]BotEdge {
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

	outgoing := map[string][]BotEdge{}
	for _, edge := range edges {
		if edge.Source == "" || edge.Target == "" {
			continue
		}
		outgoing[edge.Source] = append(outgoing[edge.Source], edge)
	}
	return outgoing
}

func chooseNextNodeID(node BotNode, outgoing []BotEdge, conditionResult *bool) string {
	if len(outgoing) == 0 {
		return ""
	}
	if conditionResult == nil || len(outgoing) == 1 {
		return outgoing[0].Target
	}

	preferred := []string{"true", "then", "sim", "entao", "então", "yes", "1"}
	if !*conditionResult {
		preferred = []string{"false", "else", "nao", "não", "senao", "senão", "no", "2"}
	}

	for _, edge := range outgoing {
		text := strings.ToLower(edge.SourceHandle + " " + edge.TargetHandle + " " + edge.Label + " " + edge.ID)
		for _, token := range preferred {
			if strings.Contains(text, token) {
				return edge.Target
			}
		}
	}

	if *conditionResult {
		return outgoing[0].Target
	}
	return outgoing[len(outgoing)-1].Target
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
			startTime := getTriggerConfigString(nodes, "trigger_off_hours", "start_time")
			endTime := getTriggerConfigString(nodes, "trigger_off_hours", "end_time")
			shouldTrigger = !isWithinBusinessHours(startTime, endTime)

		case "trigger_inbox_message":
			shouldTrigger = triggerMatchesChannel(nodes, channelID)

		case "tag_added", "trigger_tag_added":
			shouldTrigger = false
		}

		if shouldTrigger {
			if !e.reserveExecution(flowID, conversationID, contactID) {
				continue
			}

			go e.executeFlow(flowID, triggerType, companyID, conversationID, contactID, instanceName, phone, message, nodesJSON, edgesJSON)
			return // Only trigger first matching flow
		}
	}
}

// executeFlow runs a bot flow
func (e *BotEngine) executeFlow(flowID, triggerType, companyID, conversationID, contactID, instanceName, phone, triggerMessage string, nodesJSON, edgesJSON json.RawMessage) {
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

	outgoingBySource := buildOutgoingEdges(edges)
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

		conditionResult := e.evaluateCondition(node, companyID, conversationID, contactID, triggerMessage)

		err := e.executeNode(node, companyID, conversationID, contactID, instanceName, phone)
		if err != nil {
			log.Printf("[BOT] Error executing node %s: %v", node.ID, err)
			e.markExecutionError(execID, node.ID, err)
			e.addBotExecutionNote(companyID, conversationID, fmt.Sprintf("Falha no envio da automacao no bloco %s: %v", getNodeType(node), err))
			return
		}

		// Update current node
		e.db.Exec(`
			UPDATE bot_executions
			SET context = COALESCE(context, '{}'::jsonb) || jsonb_build_object('current_node_id', $1)
			WHERE id = $2
		`, node.ID, execID)

		nextID := chooseNextNodeID(node, outgoingBySource[node.ID], conditionResult)
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

func (e *BotEngine) markExecutionError(execID, nodeID string, err error) {
	if execID == "" {
		return
	}
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	e.db.Exec(`
		UPDATE bot_executions
		SET status = 'error',
			completed_at = NOW(),
			context = COALESCE(context, '{}'::jsonb) || jsonb_build_object('error', $1, 'error_node_id', $2)
		WHERE id = $3
	`, errMsg, nodeID, execID)
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

func (e *BotEngine) evaluateCondition(node BotNode, companyID, conversationID, contactID, incomingMessage string) *bool {
	nodeType := getNodeType(node)
	if nodeType == "condition_business_hours" {
		config := getNodeConfig(node)
		startTime, _ := config["start_time"].(string)
		endTime, _ := config["end_time"].(string)
		result := isWithinBusinessHours(startTime, endTime)
		return &result
	}
	if nodeType != "condition" && nodeType != "condition_contact_field" && nodeType != "condition_tag" {
		return nil
	}

	config := getNodeConfig(node)
	field, _ := config["field"].(string)
	operator, _ := config["operator"].(string)
	expected, _ := config["value"].(string)
	if operator == "" {
		operator = "contains"
	}

	actual := ""
	switch field {
	case "", "message":
		actual = incomingMessage
	case "contact_name":
		e.db.QueryRow("SELECT COALESCE(name, '') FROM contacts WHERE id = $1 AND company_id = $2", contactID, companyID).Scan(&actual)
	case "contact_tag":
		e.db.QueryRow(`
			SELECT COALESCE(string_agg(t.name, ','), '')
			FROM contact_tags ct
			JOIN tags t ON t.id = ct.tag_id
			WHERE ct.contact_id = $1 AND t.company_id = $2
		`, contactID, companyID).Scan(&actual)
	case "channel":
		e.db.QueryRow(`
			SELECT COALESCE(ch.name, '')
			FROM conversations c
			LEFT JOIN channels ch ON ch.id = c.channel_id
			WHERE c.id = $1 AND c.company_id = $2
		`, conversationID, companyID).Scan(&actual)
	case "team":
		e.db.QueryRow(`
			SELECT COALESCE(t.name, '')
			FROM conversations c
			LEFT JOIN teams t ON t.id = c.team_id
			WHERE c.id = $1 AND c.company_id = $2
		`, conversationID, companyID).Scan(&actual)
	case "agent":
		e.db.QueryRow(`
			SELECT COALESCE(u.email, '')
			FROM conversations c
			LEFT JOIN users u ON u.id = c.assigned_to
			WHERE c.id = $1 AND c.company_id = $2
		`, conversationID, companyID).Scan(&actual)
	case "day_of_week":
		actual = strings.ToLower(time.Now().Weekday().String())
	case "hour":
		actual = fmt.Sprintf("%02d", time.Now().Hour())
	case "custom_variable":
		variableName, _ := config["variable_name"].(string)
		if variableName != "" {
			e.db.QueryRow(`
				SELECT COALESCE(context ->> $1, '')
				FROM bot_executions
				WHERE conversation_id = $2 AND status = 'running'
				ORDER BY started_at DESC
				LIMIT 1
			`, variableName, conversationID).Scan(&actual)
		}
	default:
		e.db.QueryRow(fmt.Sprintf("SELECT COALESCE(%s, '') FROM contacts WHERE id = $1 AND company_id = $2", safeContactField(field)), contactID, companyID).Scan(&actual)
	}

	result := compareCondition(actual, expected, operator)
	return &result
}

func safeContactField(field string) string {
	switch field {
	case "name", "email", "company_name", "city", "state", "origin", "notes", "phone":
		return field
	default:
		return "name"
	}
}

func compareCondition(actual, expected, operator string) bool {
	actual = strings.TrimSpace(strings.ToLower(actual))
	expected = strings.TrimSpace(strings.ToLower(expected))

	switch operator {
	case "equals":
		return actual == expected
	case "not_equals":
		return actual != expected
	case "not_contains":
		return !strings.Contains(actual, expected)
	case "starts_with":
		return strings.HasPrefix(actual, expected)
	case "ends_with":
		return strings.HasSuffix(actual, expected)
	case "greater_than":
		return actual > expected
	case "less_than":
		return actual < expected
	case "is_empty":
		return actual == ""
	case "is_not_empty":
		return actual != ""
	default:
		return strings.Contains(actual, expected)
	}
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
	case "send_image":
		return e.nodeSendMedia(config, companyID, conversationID, instanceName, phone, "image")
	case "send_document":
		return e.nodeSendMedia(config, companyID, conversationID, instanceName, phone, "document")
	case "send_video":
		return e.nodeSendMedia(config, companyID, conversationID, instanceName, phone, "video")
	case "send_audio":
		return e.nodeSendAudio(config, companyID, conversationID, instanceName, phone)
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
	case "delay", "wait_seconds", "wait_minutes", "wait_hours", "wait_response", "wait_business_hours":
		if secs, ok := getNumber(config["seconds"]); ok {
			node.Data["seconds"] = secs
		}
		if mins, ok := getNumber(config["minutes"]); ok {
			node.Data["seconds"] = mins * 60
		}
		if mins, ok := getNumber(config["timeout_minutes"]); ok {
			node.Data["seconds"] = mins * 60
		}
		if hrs, ok := getNumber(config["hours"]); ok {
			node.Data["seconds"] = hrs * 3600
		}
		if nodeType == "wait_business_hours" {
			startTime, _ := config["start_time"].(string)
			endTime, _ := config["end_time"].(string)
			node.Data["seconds"] = secondsUntilBusinessHours(startTime, endTime)
		}
		return e.nodeDelay(node)
	case "add_tag", "action_add_tag":
		tn, _ := config["tag_name"].(string)
		if tn != "" {
			node.Data["tag_name"] = tn
		}
		return e.nodeAddTag(node, companyID, contactID)
	case "action_remove_tag":
		tn, _ := config["tag_name"].(string)
		if tn != "" {
			node.Data["tag_name"] = tn
		}
		return e.nodeRemoveTag(node, companyID, contactID)
	case "transfer_team", "action_transfer_team":
		tmID, _ := config["team_id"].(string)
		if tmID != "" {
			node.Data["team_id"] = tmID
		}
		tm, _ := config["team_name"].(string)
		if tm != "" {
			node.Data["team_name"] = tm
		}
		return e.nodeTransferTeam(node, companyID, conversationID)
	case "action_assign_agent":
		agentID, _ := config["agent_id"].(string)
		if agentID != "" {
			node.Data["agent_id"] = agentID
		}
		email, _ := config["agent_email"].(string)
		node.Data["agent_email"] = email
		return e.nodeAssignAgent(node, companyID, conversationID)
	case "action_update_contact":
		field, _ := config["contact_field"].(string)
		value, _ := config["contact_value"].(string)
		node.Data["contact_field"] = field
		node.Data["contact_value"] = value
		return e.nodeUpdateContact(node, companyID, conversationID, contactID, phone)
	case "action_change_funnel", "action_create_deal", "action_send_notification":
		return e.nodeInternalNote(node, companyID, conversationID, nodeType)
	case "call_webhook", "action_webhook":
		return e.nodeCallWebhook(node, companyID, conversationID, contactID)
	case "glpi_open_ticket":
		return e.nodeGLPIOpenTicket(companyID, conversationID, contactID, instanceName, phone)
	case "glpi_check_status":
		return e.nodeGLPICheckStatus(companyID, conversationID, contactID, instanceName, phone)
	case "end", "action_close_conversation":
		cc, _ := config["close_conversation"].(bool)
		if nodeType == "action_close_conversation" {
			cc = true
		}
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
	if instanceName == "" || phone == "" {
		return fmt.Errorf("cannot send bot message: instanceName='%s' phone='%s'", instanceName, phone)
	}
	var err error
	externalID, err = e.evo.SendTextMessage(instanceName, phone, message)
	if err != nil {
		log.Printf("[BOT] Failed to send message to %s via %s: %v", phone, instanceName, err)
		return err
	}
	log.Printf("[BOT] Sent message to %s via %s", phone, instanceName)

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

func (e *BotEngine) nodeSendMedia(config map[string]interface{}, companyID, conversationID, instanceName, phone, mediaType string) error {
	mediaURL := firstString(config, "media_url", "url", "file_url", "image_url", "document_url", "video_url")
	caption := firstString(config, "caption", "message")
	fileName := firstString(config, "file_name", "filename", "name")
	if mediaURL == "" {
		if caption != "" {
			return e.sendBotText(companyID, conversationID, instanceName, phone, caption, "text")
		}
		return nil
	}

	var externalID string
	if instanceName == "" || phone == "" {
		return fmt.Errorf("cannot send bot media: instanceName='%s' phone='%s'", instanceName, phone)
	}
	var err error
	externalID, err = e.evo.SendMediaMessage(instanceName, phone, mediaType, mediaURL, e.replaceVariables(caption, companyID, conversationID, phone), fileName)
	if err != nil {
		return err
	}
	return e.saveBotMessage(companyID, conversationID, e.replaceVariables(caption, companyID, conversationID, phone), mediaType, mediaURL, externalID)
}

func (e *BotEngine) nodeSendAudio(config map[string]interface{}, companyID, conversationID, instanceName, phone string) error {
	audioURL := firstString(config, "audio_url", "media_url", "url", "file_url")
	if audioURL == "" {
		return nil
	}

	var externalID string
	if instanceName == "" || phone == "" {
		return fmt.Errorf("cannot send bot audio: instanceName='%s' phone='%s'", instanceName, phone)
	}
	var err error
	externalID, err = e.evo.SendAudioMessage(instanceName, phone, audioURL)
	if err != nil {
		return err
	}
	return e.saveBotMessage(companyID, conversationID, "", "audio", audioURL, externalID)
}

func (e *BotEngine) sendBotText(companyID, conversationID, instanceName, phone, message, messageType string) error {
	message = e.replaceVariables(message, companyID, conversationID, phone)
	var externalID string
	if instanceName == "" || phone == "" {
		return fmt.Errorf("cannot send bot text: instanceName='%s' phone='%s'", instanceName, phone)
	}
	var err error
	externalID, err = e.evo.SendTextMessage(instanceName, phone, message)
	if err != nil {
		return err
	}
	return e.saveBotMessage(companyID, conversationID, message, messageType, "", externalID)
}

func (e *BotEngine) saveBotMessage(companyID, conversationID, content, messageType, mediaURL, externalID string) error {
	msgID := uuid.New().String()
	_, err := e.db.Exec(`
		INSERT INTO messages (id, conversation_id, company_id, sender_type, content, message_type, media_url, external_id, status)
		VALUES ($1, $2, $3, 'bot', NULLIF($4, ''), $5, NULLIF($6, ''), $7, 'sent')
	`, msgID, conversationID, companyID, content, messageType, mediaURL, externalID)
	if err != nil {
		return err
	}

	preview := content
	if preview == "" {
		preview = "[" + messageType + "]"
	}
	e.db.Exec("UPDATE conversations SET last_message_at = NOW(), last_message_preview = $1, updated_at = NOW() WHERE id = $2", preview, conversationID)

	e.wsHub.BroadcastToCompany(companyID, "new_message", map[string]interface{}{
		"id": msgID, "conversation_id": conversationID,
		"sender_type": "bot", "content": content, "message_type": messageType,
		"media_url": mediaURL, "sender_name": "Bot",
		"status": "sent", "created_at": time.Now(),
	})
	return nil
}

func firstString(config map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := config[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
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
	if instanceName == "" || phone == "" {
		return fmt.Errorf("cannot send bot question: instanceName='%s' phone='%s'", instanceName, phone)
	}
	var err error
	externalID, err = e.evo.SendTextMessage(instanceName, phone, message)
	if err != nil {
		log.Printf("[BOT] Failed to send question to %s via %s: %v", phone, instanceName, err)
		return err
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

func (e *BotEngine) nodeRemoveTag(node BotNode, companyID, contactID string) error {
	tagName, _ := node.Data["tag_name"].(string)
	if tagName == "" {
		return nil
	}

	_, err := e.db.Exec(`
		DELETE FROM contact_tags
		WHERE contact_id = $1
		  AND tag_id IN (SELECT id FROM tags WHERE company_id = $2 AND name ILIKE $3)
	`, contactID, companyID, tagName)
	return err
}

func (e *BotEngine) nodeTransferTeam(node BotNode, companyID, conversationID string) error {
	teamID, _ := node.Data["team_id"].(string)
	teamName, _ := node.Data["team_name"].(string)
	if teamID == "" && teamName == "" {
		return nil
	}

	if teamID == "" {
		err := e.db.QueryRow("SELECT id FROM teams WHERE company_id = $1 AND name ILIKE $2 AND is_active = true", companyID, teamName).Scan(&teamID)
		if err != nil {
			return nil
		}
	} else {
		var exists bool
		e.db.QueryRow("SELECT EXISTS(SELECT 1 FROM teams WHERE id = $1 AND company_id = $2 AND is_active = true)", teamID, companyID).Scan(&exists)
		if !exists {
			return nil
		}
	}

	e.db.Exec("UPDATE conversations SET team_id = $1, assigned_to = NULL, updated_at = NOW() WHERE id = $2 AND company_id = $3", teamID, conversationID, companyID)

	// Pause bot when transferring
	e.db.Exec("UPDATE bot_executions SET status = 'paused' WHERE conversation_id = $1 AND status = 'running'", conversationID)

	return nil
}

func (e *BotEngine) nodeAssignAgent(node BotNode, companyID, conversationID string) error {
	userID, _ := node.Data["agent_id"].(string)
	agentEmail, _ := node.Data["agent_email"].(string)
	if strings.TrimSpace(userID) == "" && strings.TrimSpace(agentEmail) == "" {
		return nil
	}

	if userID == "" {
		err := e.db.QueryRow(`
			SELECT id
			FROM users
			WHERE company_id = $1 AND LOWER(email) = LOWER($2) AND is_active = true
			LIMIT 1
		`, companyID, agentEmail).Scan(&userID)
		if err != nil {
			return nil
		}
	} else {
		var exists bool
		e.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND company_id = $2 AND is_active = true)", userID, companyID).Scan(&exists)
		if !exists {
			return nil
		}
	}

	_, err := e.db.Exec(`
		UPDATE conversations
		SET assigned_to = $1, status = 'in_progress', updated_at = NOW()
		WHERE id = $2 AND company_id = $3
	`, userID, conversationID, companyID)
	return err
}

func (e *BotEngine) nodeUpdateContact(node BotNode, companyID, conversationID, contactID, phone string) error {
	field, _ := node.Data["contact_field"].(string)
	value, _ := node.Data["contact_value"].(string)
	field = safeContactField(field)
	value = e.replaceVariables(value, companyID, conversationID, phone)
	if value == "" {
		return nil
	}

	_, err := e.db.Exec(fmt.Sprintf(`
		UPDATE contacts
		SET %s = $1, updated_at = NOW()
		WHERE id = $2 AND company_id = $3
	`, field), value, contactID, companyID)
	return err
}

func (e *BotEngine) nodeInternalNote(node BotNode, companyID, conversationID, nodeType string) error {
	content := fmt.Sprintf("Automacao executou bloco %s. Configure os campos especificos deste bloco para acao completa.", nodeType)
	return e.addBotExecutionNote(companyID, conversationID, content)
}

func (e *BotEngine) addBotExecutionNote(companyID, conversationID, content string) error {
	msgID := uuid.New().String()
	_, err := e.db.Exec(`
		INSERT INTO messages (id, conversation_id, company_id, sender_type, content, message_type, is_private, status)
		VALUES ($1, $2, $3, 'bot', $4, 'text', true, 'sent')
	`, msgID, conversationID, companyID, content)
	if err != nil {
		return err
	}

	e.wsHub.BroadcastToCompany(companyID, "new_message", map[string]interface{}{
		"id":              msgID,
		"conversation_id": conversationID,
		"sender_type":     "bot",
		"content":         content,
		"message_type":    "text",
		"is_private":      true,
		"sender_name":     "Bot",
		"status":          "sent",
		"created_at":      time.Now(),
	})
	return err
}

func (e *BotEngine) nodeCallWebhook(node BotNode, companyID, conversationID, contactID string) error {
	config := getNodeConfig(node)
	url := firstString(config, "url", "webhook_url")
	if url == "" {
		return nil
	}

	method := strings.ToUpper(firstString(config, "method"))
	if method == "" {
		method = http.MethodPost
	}

	payload := map[string]interface{}{
		"conversation_id": conversationID,
		"contact_id":      contactID,
		"company_id":      companyID,
		"node_id":         node.ID,
		"node_type":       getNodeType(node),
		"config":          config,
	}

	var body io.Reader
	if method != http.MethodGet {
		raw, _ := json.Marshal(payload)
		body = bytes.NewBuffer(raw)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}
	if method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned %d: %s", resp.StatusCode, string(data))
	}
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
