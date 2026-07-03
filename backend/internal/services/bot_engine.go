package services

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
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

func getStringValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case bool:
		return strconv.FormatBool(v)
	default:
		return ""
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
	case "new_conversation", "keyword", "off_hours", "tag_added", "client_inactivity":
		return "trigger_" + triggerType
	default:
		return triggerType
	}
}

func nodeWaitsForResponse(node BotNode) bool {
	switch getNodeType(node) {
	case "ask_question", "ask_text", "ask_options", "ask_email", "ask_phone", "buttons", "send_buttons", "list_options", "wait_response":
		return true
	default:
		return false
	}
}

func nodeWaitsForExternalCompletion(node BotNode) bool {
	switch getNodeType(node) {
	case "glpi_open_ticket", "glpi_check_status":
		return true
	default:
		return false
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

func getTriggerConfigInt(nodes []BotNode, triggerType, key string, fallback int) int {
	triggerType = normalizeTriggerType(triggerType)
	for _, node := range nodes {
		if getNodeType(node) != triggerType {
			continue
		}
		value := getNodeConfig(node)[key]
		if n, ok := getNumber(value); ok && n > 0 {
			return int(n)
		}
		if text := getStringValue(value); text != "" {
			parsed, err := strconv.Atoi(text)
			if err == nil && parsed > 0 {
				return parsed
			}
		}
	}
	return fallback
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

func chooseNextNodeID(node BotNode, outgoing []BotEdge, conditionResult *bool, nodesByID map[string]BotNode) string {
	if len(outgoing) == 0 {
		return ""
	}
	if conditionResult == nil || len(outgoing) == 1 {
		return outgoing[0].Target
	}

	trueTokens := []string{"true", "then", "sim", "entao", "então", "yes", "verdadeiro", "verdadeira"}
	falseTokens := []string{"false", "else", "nao", "não", "senao", "senão", "no", "falso", "falsa"}

	preferred, opposite := trueTokens, falseTokens
	if !*conditionResult {
		preferred, opposite = falseTokens, trueTokens
	}

	// 1. Match only against explicit edge metadata (handle/label), NEVER the edge ID.
	// Auto-generated IDs (e.g. "edge_1719776001234") contain arbitrary digits/letters
	// that would falsely match short tokens and always return the first edge.
	for _, edge := range outgoing {
		if edgeMetaMatches(edge, preferred) {
			log.Printf("[BOT] condition %s -> %v routed by label to %s", node.ID, *conditionResult, edge.Target)
			return edge.Target
		}
	}

	// 2. If the opposite branch is explicitly labeled, take any edge that is NOT the opposite.
	for _, edge := range outgoing {
		if edgeMetaMatches(edge, opposite) {
			for _, candidate := range outgoing {
				if !edgeMetaMatches(candidate, opposite) {
					log.Printf("[BOT] condition %s -> %v routed by opposite-label to %s", node.ID, *conditionResult, candidate.Target)
					return candidate.Target
				}
			}
			break
		}
	}

	// 3. Geometry: in a tree layout the ENTÃO (true) branch continues to the SIDE
	// while the SENÃO (false) branch drops DOWN. This is independent of creation
	// order and of edge labels, so it also fixes older flows.
	if targetID := chooseConditionTargetByGeometry(node, outgoing, *conditionResult, nodesByID); targetID != "" {
		log.Printf("[BOT] condition %s -> %v routed by geometry to %s", node.ID, *conditionResult, targetID)
		return targetID
	}

	// 4. Fall back to output order, matching the builder convention:
	// Saída 1 (primeira aresta) = ENTÃO (verdadeiro), Saída 2 (última) = SENÃO (falso).
	if *conditionResult {
		log.Printf("[BOT] condition %s -> true routed by order to %s", node.ID, outgoing[0].Target)
		return outgoing[0].Target
	}
	log.Printf("[BOT] condition %s -> false routed by order to %s", node.ID, outgoing[len(outgoing)-1].Target)
	return outgoing[len(outgoing)-1].Target
}

// chooseConditionTargetByGeometry distinguishes the ENTÃO/SENÃO branches of a
// condition using the position of the target nodes. The branch whose target is
// positioned more "downward than sideways" is treated as the SENÃO (false)
// branch; the other is the ENTÃO (true) branch. Returns "" when the two targets
// are not clearly separable so the caller can fall back to another heuristic.
func chooseConditionTargetByGeometry(node BotNode, outgoing []BotEdge, conditionResult bool, nodesByID map[string]BotNode) string {
	if len(outgoing) < 2 || len(nodesByID) == 0 {
		return ""
	}
	type candidate struct {
		target    string
		downScore float64
	}
	candidates := make([]candidate, 0, len(outgoing))
	for _, edge := range outgoing {
		target, ok := nodesByID[edge.Target]
		if !ok {
			continue
		}
		dx := target.Position["x"] - node.Position["x"]
		dy := target.Position["y"] - node.Position["y"]
		// Higher score = more downward relative to horizontal displacement.
		candidates = append(candidates, candidate{target: edge.Target, downScore: dy - dx})
	}
	if len(candidates) < 2 {
		return ""
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].downScore < candidates[j].downScore
	})
	trueTarget := candidates[0].target                  // least downward -> continues to the side
	falseTarget := candidates[len(candidates)-1].target // most downward -> drops down
	// Require a meaningful separation before trusting geometry.
	if math.Abs(candidates[len(candidates)-1].downScore-candidates[0].downScore) < 1 {
		return ""
	}
	if conditionResult {
		return trueTarget
	}
	return falseTarget
}

// edgeMetaMatches reports whether an edge's semantic metadata (source/target handle
// or label) contains one of the given tokens. It intentionally ignores edge.ID.
func edgeMetaMatches(edge BotEdge, tokens []string) bool {
	for _, meta := range []string{edge.SourceHandle, edge.TargetHandle, edge.Label} {
		meta = strings.ToLower(strings.TrimSpace(meta))
		if meta == "" {
			continue
		}
		for _, token := range tokens {
			if strings.Contains(meta, token) {
				return true
			}
		}
	}
	return false
}

func chooseNextNodeIDForResponse(outgoing []BotEdge, response string) string {
	if len(outgoing) == 0 {
		return ""
	}
	response = strings.TrimSpace(strings.ToLower(response))
	if response != "" {
		if idx, err := strconv.Atoi(response); err == nil && idx >= 1 && idx <= len(outgoing) {
			return outgoing[idx-1].Target
		}
		for _, edge := range outgoing {
			text := strings.ToLower(edge.SourceHandle + " " + edge.TargetHandle + " " + edge.Label + " " + edge.ID)
			if strings.Contains(text, response) {
				return edge.Target
			}
		}
	}
	return outgoing[0].Target
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
	if e.conversationHasHumanOwner(companyID, conversationID) {
		log.Printf("[BOT] Skipping automation for conversation %s because it is assigned to human attendance", conversationID)
		return
	}

	if e.resumeWaitingExecution(companyID, conversationID, contactID, instanceName, phone, message) {
		return
	}

	// Find active flows for this company
	rows, err := e.db.Query(`
		SELECT id, trigger_type, trigger_value, nodes, edges, priority, stop_on_match
		FROM bot_flows
		WHERE company_id = $1 AND is_active = true
		ORDER BY priority DESC, created_at ASC
	`, companyID)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var flowID, triggerType string
		var triggerValue *string
		var priority int
		var stopOnMatch bool
		var nodesJSON, edgesJSON json.RawMessage

		if err := rows.Scan(&flowID, &triggerType, &triggerValue, &nodesJSON, &edgesJSON, &priority, &stopOnMatch); err != nil {
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

			log.Printf("[BOT] Triggering flow %s (%s) with priority %d for conversation %s", flowID, triggerType, priority, conversationID)
			go e.executeFlow(flowID, triggerType, companyID, conversationID, contactID, instanceName, phone, message, nodesJSON, edgesJSON)
			if stopOnMatch {
				return
			}
			return
		}
	}
}

// TriggerConversationClosed fires bot flows configured with the "conversation
// closed" trigger. It is intended to run after an agent manually resolves a
// conversation (via the /close endpoint), so a farewell/closing flow can run.
// Bot-driven closes (the action_close_conversation node) update the conversation
// directly and do NOT pass through here, which avoids re-trigger loops.
func (e *BotEngine) TriggerConversationClosed(companyID, conversationID string) {
	var contactID, channelID, phone, instanceName string
	err := e.db.QueryRow(`
		SELECT COALESCE(c.contact_id::text, ''),
		       COALESCE(c.channel_id::text, ''),
		       COALESCE(ct.phone, ''),
		       COALESCE(wi.instance_name, '')
		FROM conversations c
		LEFT JOIN contacts ct ON ct.id = c.contact_id
		LEFT JOIN whatsapp_instances wi ON wi.channel_id = c.channel_id AND wi.company_id = c.company_id
		WHERE c.id = $1 AND c.company_id = $2
	`, conversationID, companyID).Scan(&contactID, &channelID, &phone, &instanceName)
	if err != nil {
		log.Printf("[BOT] Could not load conversation %s for closing flow: %v", conversationID, err)
		return
	}

	rows, err := e.db.Query(`
		SELECT id, trigger_type, nodes, edges, priority, stop_on_match
		FROM bot_flows
		WHERE company_id = $1 AND is_active = true
			AND trigger_type IN ('conversation_closed', 'trigger_conversation_closed')
		ORDER BY priority DESC, created_at ASC
	`, companyID)
	if err != nil {
		log.Printf("[BOT] Failed to query closing flows for company %s: %v", companyID, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var flowID, triggerType string
		var priority int
		var stopOnMatch bool
		var nodesJSON, edgesJSON json.RawMessage
		if err := rows.Scan(&flowID, &triggerType, &nodesJSON, &edgesJSON, &priority, &stopOnMatch); err != nil {
			continue
		}
		nodes, err := parseBotNodes(nodesJSON)
		if err != nil || len(nodes) == 0 {
			continue
		}
		if !e.reserveExecution(flowID, conversationID, contactID) {
			continue
		}
		log.Printf("[BOT] Triggering closing flow %s (%s) priority %d for conversation %s", flowID, triggerType, priority, conversationID)
		go e.executeFlow(flowID, triggerType, companyID, conversationID, contactID, instanceName, phone, "", nodesJSON, edgesJSON)
		if stopOnMatch {
			return
		}
	}
}

func (e *BotEngine) StartClientInactivityMonitor() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		e.TriggerClientInactivityFlows()
		for range ticker.C {
			e.TriggerClientInactivityFlows()
		}
	}()
}

func (e *BotEngine) TriggerClientInactivityFlows() {
	rows, err := e.db.Query(`
		SELECT id, company_id::text, trigger_type, trigger_value, nodes, edges, priority, stop_on_match
		FROM bot_flows
		WHERE is_active = true
			AND trigger_type IN ('client_inactivity', 'trigger_client_inactivity', 'no_response', 'trigger_no_response')
		ORDER BY priority DESC, created_at ASC
	`)
	if err != nil {
		log.Printf("[BOT] Failed to query client inactivity flows: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var flowID, companyID, triggerType string
		var triggerValue *string
		var priority int
		var stopOnMatch bool
		var nodesJSON, edgesJSON json.RawMessage
		if err := rows.Scan(&flowID, &companyID, &triggerType, &triggerValue, &nodesJSON, &edgesJSON, &priority, &stopOnMatch); err != nil {
			continue
		}

		nodes, err := parseBotNodes(nodesJSON)
		if err != nil || len(nodes) == 0 {
			continue
		}
		inactivityMinutes := getTriggerConfigInt(nodes, triggerType, "inactivity_minutes", 0)
		if inactivityMinutes <= 0 {
			inactivityMinutes = getTriggerConfigInt(nodes, triggerType, "timeout_minutes", 30)
		}
		if triggerValue != nil && strings.TrimSpace(*triggerValue) != "" {
			if parsed, err := strconv.Atoi(strings.TrimSpace(*triggerValue)); err == nil && parsed > 0 {
				inactivityMinutes = parsed
			}
		}
		channelID := getTriggerConfigString(nodes, triggerType, "channel_id")

		conversations, err := e.findClientInactiveConversations(companyID, flowID, channelID, inactivityMinutes)
		if err != nil {
			log.Printf("[BOT] Failed to find client inactive conversations for flow %s: %v", flowID, err)
			continue
		}

		for _, conv := range conversations {
			if !e.reserveExecution(flowID, conv.conversationID, conv.contactID) {
				continue
			}
			log.Printf("[BOT] Triggering client inactivity flow %s priority %d for conversation %s", flowID, priority, conv.conversationID)
			go e.executeFlow(flowID, triggerType, companyID, conv.conversationID, conv.contactID, conv.instanceName, conv.phone, "", nodesJSON, edgesJSON)
		}
	}
}

type inactiveConversation struct {
	conversationID string
	contactID      string
	channelID      string
	phone          string
	instanceName   string
}

func (e *BotEngine) findClientInactiveConversations(companyID, flowID, channelID string, inactivityMinutes int) ([]inactiveConversation, error) {
	if inactivityMinutes <= 0 {
		inactivityMinutes = 30
	}
	rows, err := e.db.Query(`
		SELECT c.id::text,
		       COALESCE(c.contact_id::text, ''),
		       COALESCE(c.channel_id::text, ''),
		       COALESCE(ct.phone, ''),
		       COALESCE(wi.instance_name, '')
		FROM conversations c
		JOIN LATERAL (
			SELECT m.sender_type, m.created_at
			FROM messages m
			WHERE m.conversation_id = c.id
				AND m.company_id = c.company_id
				AND COALESCE(m.is_private, false) = false
			ORDER BY m.created_at DESC
			LIMIT 1
		) last_message ON true
		LEFT JOIN contacts ct ON ct.id = c.contact_id
		LEFT JOIN whatsapp_instances wi ON wi.channel_id = c.channel_id AND wi.company_id = c.company_id
		WHERE c.company_id = $1
			AND c.status IN ('open', 'pending', 'in_progress')
			AND c.contact_id IS NOT NULL
			AND last_message.sender_type IN ('user', 'bot')
			AND last_message.created_at <= NOW() - ($2::int * INTERVAL '1 minute')
			AND ($3 = '' OR c.channel_id::text = $3)
			AND NOT EXISTS (
				SELECT 1
				FROM bot_executions be
				WHERE be.flow_id = $4
					AND be.conversation_id = c.id
					AND be.started_at >= last_message.created_at
			)
		ORDER BY last_message.created_at ASC
		LIMIT 50
	`, companyID, inactivityMinutes, channelID, flowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conversations := []inactiveConversation{}
	for rows.Next() {
		var conv inactiveConversation
		if err := rows.Scan(&conv.conversationID, &conv.contactID, &conv.channelID, &conv.phone, &conv.instanceName); err != nil {
			continue
		}
		conversations = append(conversations, conv)
	}
	return conversations, rows.Err()
}

func (e *BotEngine) conversationHasHumanOwner(companyID, conversationID string) bool {
	var hasOwner bool
	err := e.db.QueryRow(`
		SELECT assigned_to IS NOT NULL OR team_id IS NOT NULL
		FROM conversations
		WHERE id = $1 AND company_id = $2 AND status != 'resolved'
	`, conversationID, companyID).Scan(&hasOwner)
	return err == nil && hasOwner
}

// executeFlow runs a bot flow
func (e *BotEngine) executeFlow(flowID, triggerType, companyID, conversationID, contactID, instanceName, phone, triggerMessage string, nodesJSON, edgesJSON json.RawMessage) {
	e.executeFlowFrom(flowID, triggerType, companyID, conversationID, contactID, instanceName, phone, triggerMessage, nodesJSON, edgesJSON, "")
}

func (e *BotEngine) executeFlowFrom(flowID, triggerType, companyID, conversationID, contactID, instanceName, phone, triggerMessage string, nodesJSON, edgesJSON json.RawMessage, startNodeID string) {
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

	instanceName = e.resolveConversationInstance(companyID, conversationID, instanceName)

	log.Printf("[BOT] Starting flow %s for conversation %s", flowID, conversationID)

	nodesByID := make(map[string]BotNode, len(nodes))
	for _, node := range nodes {
		nodesByID[node.ID] = node
	}

	outgoingBySource := buildOutgoingEdges(edges)

	current := findStartNode(nodes, edges, triggerType)
	if startNodeID != "" {
		if node, ok := nodesByID[startNodeID]; ok {
			current = &node
		} else {
			log.Printf("[BOT] Flow %s cannot resume at missing node %s", flowID, startNodeID)
			e.markExecutionError(execID, startNodeID, fmt.Errorf("resume node not found"))
			return
		}
	}
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

		if nodeWaitsForExternalCompletion(node) {
			e.markExecutionExternalWait(execID, node.ID)
			log.Printf("[BOT] Flow %s waiting for external completion at node %s", flowID, node.ID)
			return
		}

		if nodeWaitsForResponse(node) {
			e.markExecutionWaiting(execID, node.ID)
			log.Printf("[BOT] Flow %s waiting for response at node %s", flowID, node.ID)
			return
		}

		// Update current node
		e.db.Exec(`
			UPDATE bot_executions
			SET context = COALESCE(context, '{}'::jsonb) || jsonb_build_object('current_node_id', $1::text)
			WHERE id = $2
		`, node.ID, execID)

		nextID := chooseNextNodeID(node, outgoingBySource[node.ID], conditionResult, nodesByID)
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

func (e *BotEngine) resumeWaitingExecution(companyID, conversationID, contactID, instanceName, phone, message string) bool {
	var execID, flowID, triggerType, waitingNodeID, executionConversationID string
	var nodesJSON, edgesJSON json.RawMessage
	err := e.db.QueryRow(`
		SELECT be.id, bf.id, bf.trigger_type, bf.nodes, bf.edges, COALESCE(be.context ->> 'waiting_node_id', ''), be.conversation_id::text
		FROM bot_executions be
		JOIN bot_flows bf ON bf.id = be.flow_id
		WHERE be.conversation_id = $1
			AND bf.company_id = $2
			AND (be.status = 'waiting' OR (be.status = 'running' AND COALESCE(be.context ->> 'waiting_node_id', '') <> ''))
			AND bf.is_active = true
		ORDER BY be.started_at DESC
		LIMIT 1
	`, conversationID, companyID).Scan(&execID, &flowID, &triggerType, &nodesJSON, &edgesJSON, &waitingNodeID, &executionConversationID)
	if err != nil || waitingNodeID == "" {
		if err == sql.ErrNoRows {
			log.Printf("[BOT] No waiting execution for conversation %s", conversationID)
			err = e.db.QueryRow(`
				SELECT be.id, bf.id, bf.trigger_type, bf.nodes, bf.edges, COALESCE(be.context ->> 'waiting_node_id', ''), be.conversation_id::text
				FROM bot_executions be
				JOIN bot_flows bf ON bf.id = be.flow_id
				WHERE be.contact_id = $1
					AND bf.company_id = $2
					AND (be.status = 'waiting' OR (be.status = 'running' AND COALESCE(be.context ->> 'waiting_node_id', '') <> ''))
					AND bf.is_active = true
				ORDER BY be.started_at DESC
				LIMIT 1
			`, contactID, companyID).Scan(&execID, &flowID, &triggerType, &nodesJSON, &edgesJSON, &waitingNodeID, &executionConversationID)
			if err == nil && waitingNodeID != "" && executionConversationID != conversationID {
				log.Printf("[BOT] Resuming waiting execution from conversation %s using response received in conversation %s for contact %s", executionConversationID, conversationID, contactID)
			}
			if err == sql.ErrNoRows {
				log.Printf("[BOT] No waiting execution for contact %s", contactID)
			}
		} else {
			log.Printf("[BOT] Could not load waiting execution for conversation %s: %v", conversationID, err)
		}
		if err != nil || waitingNodeID == "" {
			return false
		}
	}
	if executionConversationID == "" {
		executionConversationID = conversationID
	}
	log.Printf("[BOT] Resuming waiting flow %s for conversation %s with response %q", flowID, executionConversationID, message)

	nodes, err := parseBotNodes(nodesJSON)
	if err != nil {
		e.markExecutionError(execID, waitingNodeID, err)
		return true
	}
	edges, err := parseBotEdges(edgesJSON)
	if err != nil {
		e.markExecutionError(execID, waitingNodeID, err)
		return true
	}

	nodesByID := make(map[string]BotNode, len(nodes))
	for _, node := range nodes {
		nodesByID[node.ID] = node
	}
	waitingNode, ok := nodesByID[waitingNodeID]
	if !ok {
		e.markExecutionError(execID, waitingNodeID, fmt.Errorf("waiting node not found"))
		return true
	}

	outgoingBySource := buildOutgoingEdges(edges)
	nextID := chooseNextNodeIDForResponse(outgoingBySource[waitingNode.ID], message)
	log.Printf("[BOT] Waiting node %s response %q selected next node %s", waitingNode.ID, message, nextID)
	if nextID == "" {
		e.db.Exec(`
			UPDATE bot_executions
			SET status = 'completed',
				completed_at = NOW(),
				context = COALESCE(context, '{}'::jsonb) || jsonb_build_object('last_response', $1::text)
			WHERE id = $2
		`, message, execID)
		return true
	}

	_, err = e.db.Exec(`
		UPDATE bot_executions
		SET status = 'running',
			context = jsonb_set(
				jsonb_set(
					jsonb_set(COALESCE(context, '{}'::jsonb), '{last_response}', to_jsonb($1::text), true),
					'{waiting_node_id}', 'null'::jsonb, true
				),
				'{current_node_id}', to_jsonb($2::text), true
			)
		WHERE id = $3
	`, message, nextID, execID)
	if err != nil {
		log.Printf("[BOT] Failed to resume waiting execution %s: %v", execID, err)
		return true
	}

	instanceName = e.resolveConversationInstance(companyID, executionConversationID, instanceName)
	go e.executeFlowFrom(flowID, triggerType, companyID, executionConversationID, contactID, instanceName, phone, message, nodesJSON, edgesJSON, nextID)
	return true
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
			context = COALESCE(context, '{}'::jsonb) || jsonb_build_object('error', $1::text, 'error_node_id', $2::text)
		WHERE id = $3
	`, errMsg, nodeID, execID)
}

func (e *BotEngine) markExecutionWaiting(execID, nodeID string) {
	if execID == "" {
		return
	}
	result, err := e.db.Exec(`
		UPDATE bot_executions
		SET status = 'waiting',
			context = jsonb_set(
				jsonb_set(COALESCE(context, '{}'::jsonb), '{waiting_node_id}', to_jsonb($1::text), true),
				'{current_node_id}', to_jsonb($1::text), true
			)
		WHERE id = $2
	`, nodeID, execID)
	if err != nil {
		log.Printf("[BOT] Failed to mark execution %s waiting at node %s: %v", execID, nodeID, err)
		return
	}
	rowsAffected, _ := result.RowsAffected()
	log.Printf("[BOT] Marked execution %s waiting at node %s (%d rows)", execID, nodeID, rowsAffected)
}

func (e *BotEngine) markExecutionExternalWait(execID, nodeID string) {
	if execID == "" {
		return
	}
	result, err := e.db.Exec(`
		UPDATE bot_executions
		SET status = 'external_wait',
			context = jsonb_set(
				jsonb_set(COALESCE(context, '{}'::jsonb), '{external_node_id}', to_jsonb($1::text), true),
				'{current_node_id}', to_jsonb($1::text), true
			)
		WHERE id = $2
	`, nodeID, execID)
	if err != nil {
		log.Printf("[BOT] Failed to mark execution %s external_wait at node %s: %v", execID, nodeID, err)
		return
	}
	rowsAffected, _ := result.RowsAffected()
	log.Printf("[BOT] Marked execution %s external_wait at node %s (%d rows)", execID, nodeID, rowsAffected)
}

func (e *BotEngine) ResumeAfterExternalNode(companyID, conversationID, contactID, instanceName, phone, resultMessage string) {
	var execID, flowID, triggerType, externalNodeID string
	var nodesJSON, edgesJSON json.RawMessage
	err := e.db.QueryRow(`
		SELECT be.id, bf.id, bf.trigger_type, bf.nodes, bf.edges, COALESCE(be.context ->> 'external_node_id', '')
		FROM bot_executions be
		JOIN bot_flows bf ON bf.id = be.flow_id
		WHERE be.conversation_id = $1
			AND bf.company_id = $2
			AND be.status = 'external_wait'
			AND bf.is_active = true
		ORDER BY be.started_at DESC
		LIMIT 1
	`, conversationID, companyID).Scan(&execID, &flowID, &triggerType, &nodesJSON, &edgesJSON, &externalNodeID)
	if err != nil || externalNodeID == "" {
		log.Printf("[BOT] No external_wait execution to resume for conversation %s: %v", conversationID, err)
		return
	}

	nodes, err := parseBotNodes(nodesJSON)
	if err != nil {
		e.markExecutionError(execID, externalNodeID, err)
		return
	}
	edges, err := parseBotEdges(edgesJSON)
	if err != nil {
		e.markExecutionError(execID, externalNodeID, err)
		return
	}
	nodesByID := make(map[string]BotNode, len(nodes))
	for _, node := range nodes {
		nodesByID[node.ID] = node
	}
	externalNode, ok := nodesByID[externalNodeID]
	if !ok {
		e.markExecutionError(execID, externalNodeID, fmt.Errorf("external node not found"))
		return
	}
	outgoingBySource := buildOutgoingEdges(edges)
	nextID := chooseNextNodeID(externalNode, outgoingBySource[externalNodeID], nil, nodesByID)
	if nextID == "" {
		e.db.Exec("UPDATE bot_executions SET status = 'completed', completed_at = NOW() WHERE id = $1", execID)
		log.Printf("[BOT] External node %s completed without next node for flow %s", externalNodeID, flowID)
		return
	}

	_, err = e.db.Exec(`
		UPDATE bot_executions
		SET status = 'running',
			context = jsonb_set(
				jsonb_set(
					jsonb_set(COALESCE(context, '{}'::jsonb), '{external_result}', to_jsonb($1::text), true),
					'{external_node_id}', 'null'::jsonb, true
				),
				'{current_node_id}', to_jsonb($2::text), true
			)
		WHERE id = $3
	`, resultMessage, nextID, execID)
	if err != nil {
		log.Printf("[BOT] Failed to resume external_wait execution %s: %v", execID, err)
		return
	}

	instanceName = e.resolveConversationInstance(companyID, conversationID, instanceName)
	log.Printf("[BOT] Resuming flow %s after external node %s at next node %s", flowID, externalNodeID, nextID)
	go e.executeFlowFrom(flowID, triggerType, companyID, conversationID, contactID, instanceName, phone, resultMessage, nodesJSON, edgesJSON, nextID)
}

func (e *BotEngine) resolveConversationInstance(companyID, conversationID, fallback string) string {
	var instanceName string
	err := e.db.QueryRow(`
		SELECT COALESCE(wi.instance_name, '')
		FROM conversations c
		LEFT JOIN whatsapp_instances wi ON wi.channel_id = c.channel_id AND wi.company_id = c.company_id
		WHERE c.id = $1 AND c.company_id = $2
		LIMIT 1
	`, conversationID, companyID).Scan(&instanceName)
	if err == nil && strings.TrimSpace(instanceName) != "" {
		instanceName = strings.TrimSpace(instanceName)
		if fallback != "" && fallback != instanceName {
			log.Printf("[BOT] Using conversation instance %s instead of stale instance %s for conversation %s", instanceName, fallback, conversationID)
		}
		return instanceName
	}

	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		return fallback
	}

	var exists bool
	err = e.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM whatsapp_instances
			WHERE company_id = $1 AND instance_name = $2
		)
	`, companyID, fallback).Scan(&exists)
	if err == nil && exists {
		return fallback
	}

	log.Printf("[BOT] Instance %s is not registered for company %s", fallback, companyID)
	return fallback
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
		WHERE conversation_id = $1 AND status IN ('running', 'waiting')
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
	field := getStringValue(config["field"])
	operator := getStringValue(config["operator"])
	expected := getStringValue(config["value"])
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
	log.Printf("[BOT][COND] node %s field=%q operator=%q expected=%q actual=%q -> %v", node.ID, field, operator, expected, actual, result)
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
	operator = strings.TrimSpace(strings.ToLower(operator))

	switch operator {
	case "equals", "equal", "igual", "igual_a", "is", "==":
		return actual == expected
	case "not_equals", "not_equal", "diferente", "diferente_de", "!=":
		return actual != expected
	case "contains", "contem", "contém":
		return strings.Contains(actual, expected)
	case "not_contains", "nao_contem", "não_contém", "nao_contém":
		return !strings.Contains(actual, expected)
	case "starts_with", "comeca_com", "começa_com":
		return strings.HasPrefix(actual, expected)
	case "ends_with", "termina_com":
		return strings.HasSuffix(actual, expected)
	case "greater_than", "maior_que":
		return compareNumericOrText(actual, expected, func(a, b float64) bool { return a > b }, func(a, b string) bool { return a > b })
	case "less_than", "menor_que":
		return compareNumericOrText(actual, expected, func(a, b float64) bool { return a < b }, func(a, b string) bool { return a < b })
	case "is_empty", "vazio":
		return actual == ""
	case "is_not_empty", "nao_vazio", "não_vazio":
		return actual != ""
	default:
		return strings.Contains(actual, expected)
	}
}

func compareNumericOrText(actual, expected string, numeric func(float64, float64) bool, text func(string, string) bool) bool {
	actualNum, actualErr := strconv.ParseFloat(actual, 64)
	expectedNum, expectedErr := strconv.ParseFloat(expected, 64)
	if actualErr == nil && expectedErr == nil {
		return numeric(actualNum, expectedNum)
	}
	return text(actual, expected)
}

// executeNode processes a single node
func (e *BotEngine) executeNode(node BotNode, companyID, conversationID, contactID, instanceName, phone string) error {
	// Get the actual node type from data.nodeType (React Flow format)
	nodeType := getNodeType(node)

	// Get config from data.config
	config := getNodeConfig(node)

	switch nodeType {
	case "trigger_new_conversation", "trigger_inbox_message", "trigger_keyword", "trigger_off_hours", "trigger_tag_added", "trigger_funnel_stage", "trigger_no_response", "trigger_client_inactivity", "trigger_contact_created", "trigger_campaign_replied", "trigger_conversation_closed":
		return nil
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
	botName := e.botNameForConversation(companyID, conversationID)
	outboundMessage := formatBotOutboundMessage(botName, message)

	// Send via WhatsApp
	var externalID string
	if instanceName == "" || phone == "" {
		return fmt.Errorf("cannot send bot message: instanceName='%s' phone='%s'", instanceName, phone)
	}
	var err error
	externalID, err = e.evo.SendTextMessage(instanceName, phone, outboundMessage)
	if err != nil {
		log.Printf("[BOT] Failed to send message to %s via %s: %v", phone, instanceName, err)
		return err
	}
	log.Printf("[BOT] Sent message to %s via %s", phone, instanceName)

	// Save message
	msgID := uuid.New().String()
	e.db.Exec(`
		INSERT INTO messages (id, conversation_id, company_id, sender_type, content, message_type, external_id, status, metadata)
		VALUES ($1, $2, $3, 'bot', $4, 'text', $5, 'sent', jsonb_build_object('bot_name', $6::text))
	`, msgID, conversationID, companyID, message, externalID, botName)

	e.db.Exec("UPDATE conversations SET last_message_at = NOW(), last_message_preview = $1 WHERE id = $2", message, conversationID)

	e.wsHub.BroadcastToCompany(companyID, "new_message", map[string]interface{}{
		"id": msgID, "conversation_id": conversationID,
		"sender_type": "bot", "content": message, "message_type": "text",
		"sender_name": botName,
		"metadata":    map[string]interface{}{"bot_name": botName},
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

	// For webchat, save directly (no Evolution API)
	if instanceName == "" || phone == "" {
		var channelType string
		e.db.QueryRow(`
			SELECT COALESCE(ch.type, '')
			FROM conversations c LEFT JOIN channels ch ON ch.id = c.channel_id
			WHERE c.id = $1 AND c.company_id = $2
		`, conversationID, companyID).Scan(&channelType)
		if channelType == "webchat" {
			return e.saveBotMessage(companyID, conversationID, e.replaceVariables(caption, companyID, conversationID, phone), mediaType, mediaURL, "")
		}
		return fmt.Errorf("cannot send bot media: instanceName='%s' phone='%s'", instanceName, phone)
	}

	externalID, err := e.evo.SendMediaMessage(instanceName, phone, mediaType, mediaURL, e.replaceVariables(caption, companyID, conversationID, phone), fileName)
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

	if instanceName == "" || phone == "" {
		var channelType string
		e.db.QueryRow(`
			SELECT COALESCE(ch.type, '')
			FROM conversations c LEFT JOIN channels ch ON ch.id = c.channel_id
			WHERE c.id = $1 AND c.company_id = $2
		`, conversationID, companyID).Scan(&channelType)
		if channelType == "webchat" {
			return e.saveBotMessage(companyID, conversationID, "", "audio", audioURL, "")
		}
		return fmt.Errorf("cannot send bot audio: instanceName='%s' phone='%s'", instanceName, phone)
	}

	externalID, err := e.evo.SendAudioMessage(instanceName, phone, audioURL)
	if err != nil {
		return err
	}
	return e.saveBotMessage(companyID, conversationID, "", "audio", audioURL, externalID)
}

func (e *BotEngine) sendBotText(companyID, conversationID, instanceName, phone, message, messageType string) error {
	message = e.replaceVariables(message, companyID, conversationID, phone)
	botName := e.botNameForConversation(companyID, conversationID)
	outboundMessage := formatBotOutboundMessage(botName, message)

	// For webchat conversations (no phone/instance), just save the message.
	// The visitor receives it via WebSocket broadcast from saveBotMessage.
	if instanceName == "" || phone == "" {
		var channelType string
		e.db.QueryRow(`
			SELECT COALESCE(ch.type, '')
			FROM conversations c LEFT JOIN channels ch ON ch.id = c.channel_id
			WHERE c.id = $1 AND c.company_id = $2
		`, conversationID, companyID).Scan(&channelType)
		if channelType == "webchat" {
			log.Printf("[BOT] Sending webchat message for conversation %s", conversationID)
			return e.saveBotMessage(companyID, conversationID, message, messageType, "", "")
		}
		return fmt.Errorf("cannot send bot text: instanceName='%s' phone='%s'", instanceName, phone)
	}

	var externalID string
	var err error
	externalID, err = e.evo.SendTextMessage(instanceName, phone, outboundMessage)
	if err != nil {
		return err
	}
	return e.saveBotMessage(companyID, conversationID, message, messageType, "", externalID)
}

func (e *BotEngine) saveBotMessage(companyID, conversationID, content, messageType, mediaURL, externalID string) error {
	msgID := uuid.New().String()
	botName := e.botNameForConversation(companyID, conversationID)
	_, err := e.db.Exec(`
		INSERT INTO messages (id, conversation_id, company_id, sender_type, content, message_type, media_url, external_id, status, metadata)
		VALUES ($1, $2, $3, 'bot', NULLIF($4, ''), $5, NULLIF($6, ''), $7, 'sent', jsonb_build_object('bot_name', $8::text))
	`, msgID, conversationID, companyID, content, messageType, mediaURL, externalID, botName)
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
		"media_url": mediaURL, "sender_name": botName, "metadata": map[string]interface{}{"bot_name": botName},
		"status": "sent", "created_at": time.Now(),
	})

	// Also push to widget visitor WebSocket room so the embed chat updates instantly.
	e.wsHub.BroadcastToRoom("widget:"+conversationID, "new_message", map[string]interface{}{
		"id": msgID, "conversation_id": conversationID,
		"sender_type": "bot", "content": content, "message_type": messageType,
		"created_at": time.Now(),
	})
	return nil
}

func (e *BotEngine) botNameForConversation(companyID, conversationID string) string {
	var botName string
	err := e.db.QueryRow(`
		SELECT COALESCE(NULLIF(bf.bot_name, ''), bf.name, 'Assistente')
		FROM bot_executions be
		JOIN bot_flows bf ON bf.id = be.flow_id
		WHERE be.conversation_id = $1 AND bf.company_id = $2
		ORDER BY be.started_at DESC
		LIMIT 1
	`, conversationID, companyID).Scan(&botName)
	if err != nil || strings.TrimSpace(botName) == "" {
		return "Assistente"
	}
	return strings.TrimSpace(botName)
}

func formatBotOutboundMessage(botName, message string) string {
	botName = strings.TrimSpace(botName)
	message = strings.TrimSpace(message)
	if botName == "" || message == "" {
		return message
	}
	prefix := "*" + botName + ":*"
	if strings.HasPrefix(message, prefix) {
		return message
	}
	return prefix + "\n" + message
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

	// For webchat conversations, just save the message (delivered via WebSocket)
	if instanceName == "" || phone == "" {
		var channelType string
		e.db.QueryRow(`
			SELECT COALESCE(ch.type, '')
			FROM conversations c LEFT JOIN channels ch ON ch.id = c.channel_id
			WHERE c.id = $1 AND c.company_id = $2
		`, conversationID, companyID).Scan(&channelType)
		if channelType == "webchat" {
			log.Printf("[BOT] Sending webchat question for conversation %s", conversationID)
			return e.saveBotMessage(companyID, conversationID, message, "text", "", "")
		}
		return fmt.Errorf("cannot send bot question: instanceName='%s' phone='%s'", instanceName, phone)
	}

	// Send via WhatsApp
	botName := e.botNameForConversation(companyID, conversationID)
	outboundMessage := formatBotOutboundMessage(botName, message)
	externalID, err := e.evo.SendTextMessage(instanceName, phone, outboundMessage)
	if err != nil {
		log.Printf("[BOT] Failed to send question to %s via %s: %v", phone, instanceName, err)
		return err
	}

	// Save message
	return e.saveBotMessage(companyID, conversationID, message, "text", "", externalID)
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

	// Pause bot when transferring to human attendance.
	_ = pauseConversationAutomationState(e.db, conversationID)

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
	if err != nil {
		return err
	}
	return pauseConversationAutomationState(e.db, conversationID)
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
	}
	return nil
}

func (e *BotEngine) nodeGLPICheckStatus(companyID, conversationID, contactID, instanceName, phone string) error {
	if e.glpiFlow != nil {
		e.glpiFlow.StartGLPIFlow(companyID, conversationID, contactID, instanceName, phone, "check_status")
	}
	return nil
}

func (e *BotEngine) nodeEnd(node BotNode, companyID, conversationID string) error {
	closeConv, _ := node.Data["close_conversation"].(bool)
	if closeConv {
		_, err := e.db.Exec("UPDATE conversations SET status = 'resolved', resolved_at = NOW(), assigned_to = NULL, team_id = NULL, updated_at = NOW() WHERE id = $1 AND company_id = $2", conversationID, companyID)
		if err != nil {
			return err
		}
		return clearConversationAutomationState(e.db, conversationID)
	}
	return nil
}

// PauseBotForConversation stops any running bot when a human takes over
func (e *BotEngine) PauseBotForConversation(conversationID string) {
	_ = pauseConversationAutomationState(e.db, conversationID)
}
