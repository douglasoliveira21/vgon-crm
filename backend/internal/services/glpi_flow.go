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

// GLPIFlowState tracks the multi-step GLPI ticket creation flow per conversation
type GLPIFlowState struct {
	Step         string `json:"step"`          // current step
	EntitySearch string `json:"entity_search"` // entity name searched
	EntityID     int    `json:"entity_id"`     // confirmed entity ID
	EntityName   string `json:"entity_name"`   // confirmed entity name
	FullName     string `json:"full_name"`     // requester name
	Email        string `json:"email"`         // requester email
	Phone        string `json:"phone"`         // requester phone
	Title        string `json:"title"`         // ticket title
	Description  string `json:"description"`   // ticket description
	TicketID     int    `json:"ticket_id"`     // created ticket id
}

type GLPIFlowEngine struct {
	db    *sql.DB
	wsHub *websocket.Hub
	evo   *EvolutionService
	glpi  *GLPIService
	bot   *BotEngine
	cfg   struct {
		UserToken string
	}
}

func NewGLPIFlowEngine(db *sql.DB, wsHub *websocket.Hub, evo *EvolutionService, glpi *GLPIService, userToken string) *GLPIFlowEngine {
	e := &GLPIFlowEngine{
		db:    db,
		wsHub: wsHub,
		evo:   evo,
		glpi:  glpi,
	}
	e.cfg.UserToken = userToken
	return e
}

func (e *GLPIFlowEngine) SetBotEngine(bot *BotEngine) {
	e.bot = bot
}

// StartGLPIFlow initiates the GLPI ticket creation flow for a conversation
func (e *GLPIFlowEngine) StartGLPIFlow(companyID, conversationID, contactID, instanceName, phone string, mode string) {
	// mode: "open_ticket" or "check_status"
	state := &GLPIFlowState{}

	if mode == "check_status" {
		state.Step = "ask_ticket_number"
		e.saveState(conversationID, state)
		e.sendBotMessage(companyID, conversationID, instanceName, phone,
			"🔍 *Consultar Chamado*\n\nPor favor, digite o *número do ticket* que deseja consultar:")
		return
	}

	// Default: open_ticket
	state.Step = "ask_entity"
	state.Phone = phone
	e.saveState(conversationID, state)

	e.sendBotMessage(companyID, conversationID, instanceName, phone,
		"🎫 *Abertura de Chamado*\n\nPara abrir um chamado, preciso de algumas informações.\n\nPrimeiro, me diga o *nome da empresa* (entidade):")
}

// HandleGLPIMessage processes user responses during the GLPI flow
func (e *GLPIFlowEngine) HandleGLPIMessage(companyID, conversationID, contactID, instanceName, phone, message string) bool {
	state := e.loadState(conversationID)
	if state == nil {
		return false // No active GLPI flow
	}

	switch state.Step {
	case "ask_entity":
		return e.handleAskEntity(companyID, conversationID, instanceName, phone, message, state)
	case "confirm_entity":
		return e.handleConfirmEntity(companyID, conversationID, instanceName, phone, message, state)
	case "ask_full_name":
		return e.handleAskFullName(companyID, conversationID, instanceName, phone, message, state)
	case "ask_email":
		return e.handleAskEmail(companyID, conversationID, instanceName, phone, message, state)
	case "ask_title":
		return e.handleAskTitle(companyID, conversationID, instanceName, phone, message, state)
	case "ask_description":
		return e.handleAskDescription(companyID, conversationID, contactID, instanceName, phone, message, state)
	case "ask_ticket_number":
		return e.handleAskTicketNumber(companyID, conversationID, instanceName, phone, message, state)
	default:
		return false
	}
}

// HasActiveFlow checks if a conversation has an active GLPI flow
func (e *GLPIFlowEngine) HasActiveFlow(conversationID string) bool {
	state := e.loadState(conversationID)
	return state != nil
}

func (e *GLPIFlowEngine) ActiveConversationForMessage(companyID, conversationID, contactID string) string {
	if e.HasActiveFlow(conversationID) {
		return conversationID
	}
	if contactID == "" {
		return ""
	}

	var activeConversationID string
	err := e.db.QueryRow(`
		SELECT gfs.conversation_id::text
		FROM glpi_flow_states gfs
		JOIN conversations c ON c.id = gfs.conversation_id
		WHERE c.company_id = $1
			AND c.contact_id = $2
			AND c.status != 'resolved'
		ORDER BY gfs.updated_at DESC
		LIMIT 1
	`, companyID, contactID).Scan(&activeConversationID)
	if err != nil {
		return ""
	}
	if activeConversationID != conversationID {
		log.Printf("[GLPI-FLOW] Resuming active flow from conversation %s using response received in conversation %s", activeConversationID, conversationID)
	}
	return activeConversationID
}

// CancelFlow removes any active GLPI flow for a conversation
func (e *GLPIFlowEngine) CancelFlow(conversationID string) {
	e.db.Exec("DELETE FROM glpi_flow_states WHERE conversation_id = $1", conversationID)
}

// --- Step handlers ---

func (e *GLPIFlowEngine) handleAskEntity(companyID, conversationID, instanceName, phone, message string, state *GLPIFlowState) bool {
	state.EntitySearch = strings.TrimSpace(message)

	// Search GLPI entities
	sessionToken, err := e.glpi.InitSession(e.cfg.UserToken)
	if err != nil {
		e.sendBotMessage(companyID, conversationID, instanceName, phone,
			"❌ Não consegui conectar ao sistema de chamados. Tente novamente mais tarde.")
		e.CancelFlow(conversationID)
		return true
	}
	defer e.glpi.KillSession(sessionToken)

	entities, err := e.glpi.GetEntities(sessionToken)
	if err != nil {
		e.sendBotMessage(companyID, conversationID, instanceName, phone,
			"❌ Erro ao buscar entidades. Tente novamente mais tarde.")
		e.CancelFlow(conversationID)
		return true
	}

	// Filter entities that match the search
	searchLower := strings.ToLower(state.EntitySearch)
	var matches []GLPIEntity
	for _, ent := range entities {
		nameLower := strings.ToLower(ent.Name)
		completeLower := strings.ToLower(ent.CompleteName)
		if strings.Contains(nameLower, searchLower) || strings.Contains(completeLower, searchLower) {
			matches = append(matches, ent)
		}
	}

	if len(matches) == 0 {
		e.sendBotMessage(companyID, conversationID, instanceName, phone,
			fmt.Sprintf("❌ Nenhuma empresa encontrada com o nome *\"%s\"*.\n\nPor favor, digite novamente o nome da empresa:", state.EntitySearch))
		return true
	}

	// Build options message
	msg := fmt.Sprintf("🏢 Encontrei %d resultado(s) para *\"%s\"*:\n", len(matches), state.EntitySearch)
	for i, ent := range matches {
		if i >= 5 {
			msg += fmt.Sprintf("\n... e mais %d resultados. Seja mais específico.", len(matches)-5)
			break
		}
		displayName := ent.Name
		if ent.CompleteName != "" {
			displayName = ent.CompleteName
		}
		msg += fmt.Sprintf("\n*%d.* %s", i+1, displayName)
	}
	msg += "\n\nDigite o *número* da opção correta ou *0* para buscar novamente:"

	// Store matches in state temporarily (serialize IDs)
	matchIDs := make([]int, 0, len(matches))
	matchNames := make([]string, 0, len(matches))
	for i, m := range matches {
		if i >= 5 {
			break
		}
		matchIDs = append(matchIDs, m.ID)
		if m.CompleteName != "" {
			matchNames = append(matchNames, m.CompleteName)
		} else {
			matchNames = append(matchNames, m.Name)
		}
	}

	// Store in DB as JSON metadata
	metaJSON, _ := json.Marshal(map[string]interface{}{
		"match_ids":   matchIDs,
		"match_names": matchNames,
	})
	e.db.Exec("UPDATE glpi_flow_states SET metadata = $1 WHERE conversation_id = $2", string(metaJSON), conversationID)

	state.Step = "confirm_entity"
	e.saveState(conversationID, state)
	e.sendBotMessage(companyID, conversationID, instanceName, phone, msg)
	return true
}

func (e *GLPIFlowEngine) handleConfirmEntity(companyID, conversationID, instanceName, phone, message string, state *GLPIFlowState) bool {
	choice := strings.TrimSpace(message)

	if choice == "0" {
		state.Step = "ask_entity"
		e.saveState(conversationID, state)
		e.sendBotMessage(companyID, conversationID, instanceName, phone,
			"Ok! Digite novamente o *nome da empresa*:")
		return true
	}

	// Load metadata
	var metaStr string
	e.db.QueryRow("SELECT COALESCE(metadata, '{}') FROM glpi_flow_states WHERE conversation_id = $1", conversationID).Scan(&metaStr)

	var meta map[string]interface{}
	json.Unmarshal([]byte(metaStr), &meta)

	matchIDsRaw, _ := meta["match_ids"].([]interface{})
	matchNamesRaw, _ := meta["match_names"].([]interface{})

	// Parse choice as number
	var idx int
	fmt.Sscanf(choice, "%d", &idx)
	if idx < 1 || idx > len(matchIDsRaw) {
		e.sendBotMessage(companyID, conversationID, instanceName, phone,
			"❌ Opção inválida. Digite o *número* da opção ou *0* para buscar novamente:")
		return true
	}

	entityID := int(matchIDsRaw[idx-1].(float64))
	entityName := matchNamesRaw[idx-1].(string)

	state.EntityID = entityID
	state.EntityName = entityName
	state.Step = "ask_full_name"
	e.saveState(conversationID, state)

	e.sendBotMessage(companyID, conversationID, instanceName, phone,
		fmt.Sprintf("✅ Empresa confirmada: *%s*\n\nAgora, informe seu *nome completo*:", entityName))
	return true
}

func (e *GLPIFlowEngine) handleAskFullName(companyID, conversationID, instanceName, phone, message string, state *GLPIFlowState) bool {
	state.FullName = strings.TrimSpace(message)
	state.Step = "ask_email"
	e.saveState(conversationID, state)

	e.sendBotMessage(companyID, conversationID, instanceName, phone,
		fmt.Sprintf("👤 Nome: *%s*\n\nAgora, informe seu *e-mail de contato*:", state.FullName))
	return true
}

func (e *GLPIFlowEngine) handleAskEmail(companyID, conversationID, instanceName, phone, message string, state *GLPIFlowState) bool {
	state.Email = strings.TrimSpace(message)
	state.Step = "ask_title"
	e.saveState(conversationID, state)

	e.sendBotMessage(companyID, conversationID, instanceName, phone,
		"📧 E-mail registrado!\n\nAgora, descreva em poucas palavras o *título do problema*:")
	return true
}

func (e *GLPIFlowEngine) handleAskTitle(companyID, conversationID, instanceName, phone, message string, state *GLPIFlowState) bool {
	state.Title = strings.TrimSpace(message)
	state.Step = "ask_description"
	e.saveState(conversationID, state)

	e.sendBotMessage(companyID, conversationID, instanceName, phone,
		"📝 Título registrado!\n\nAgora, descreva com detalhes o *problema* que está ocorrendo:")
	return true
}

func (e *GLPIFlowEngine) handleAskDescription(companyID, conversationID, contactID, instanceName, phone, message string, state *GLPIFlowState) bool {
	state.Description = strings.TrimSpace(message)

	// All data collected - create ticket in GLPI
	e.sendBotMessage(companyID, conversationID, instanceName, phone,
		"⏳ Aguarde, estou abrindo seu chamado...")

	sessionToken, err := e.glpi.InitSession(e.cfg.UserToken)
	if err != nil {
		e.sendBotMessage(companyID, conversationID, instanceName, phone,
			"❌ Não consegui conectar ao sistema de chamados. Tente novamente mais tarde.")
		e.CancelFlow(conversationID)
		return true
	}
	defer e.glpi.KillSession(sessionToken)

	// Build content with all info
	content := fmt.Sprintf(
		"<b>Solicitante:</b> %s<br/><b>E-mail:</b> %s<br/><b>Telefone:</b> %s<br/><b>Empresa:</b> %s<br/><br/><b>Descrição:</b><br/>%s",
		state.FullName, state.Email, state.Phone, state.EntityName, state.Description,
	)

	ticket, err := e.glpi.CreateTicket(sessionToken, state.Title, content, state.EntityID, 1, 3)
	if err != nil {
		e.sendBotMessage(companyID, conversationID, instanceName, phone,
			"❌ Erro ao criar chamado: "+err.Error()+"\n\nTente novamente mais tarde.")
		e.CancelFlow(conversationID)
		return true
	}

	state.TicketID = ticket.ID
	e.CancelFlow(conversationID) // Flow completed

	// Success message
	successMsg := fmt.Sprintf(
		"✅ *Chamado aberto com sucesso!*\n\n"+
			"🎫 *Ticket #%d*\n"+
			"🏢 Empresa: %s\n"+
			"👤 Solicitante: %s\n"+
			"📧 E-mail: %s\n"+
			"📞 Telefone: %s\n"+
			"📋 Título: %s\n\n"+
			"Guarde o número *#%d* para acompanhar o status do seu chamado.",
		ticket.ID, state.EntityName, state.FullName, state.Email, state.Phone, state.Title, ticket.ID,
	)
	e.sendBotMessage(companyID, conversationID, instanceName, phone, successMsg)
	e.resumeBotAfterCompletion(companyID, conversationID, contactID, instanceName, phone, fmt.Sprintf("glpi_ticket_opened:%d", ticket.ID))

	// Update contact info in CRM
	if contactID != "" {
		e.db.Exec("UPDATE contacts SET name = $1, email = $2, company_name = $3 WHERE id = $4 AND (name IS NULL OR name = '' OR name = phone)",
			state.FullName, state.Email, state.EntityName, contactID)
	}

	return true
}

func (e *GLPIFlowEngine) handleAskTicketNumber(companyID, conversationID, instanceName, phone, message string, state *GLPIFlowState) bool {
	ticketNum := strings.TrimSpace(message)
	// Remove # if present
	ticketNum = strings.TrimPrefix(ticketNum, "#")

	var ticketID int
	fmt.Sscanf(ticketNum, "%d", &ticketID)
	if ticketID <= 0 {
		e.sendBotMessage(companyID, conversationID, instanceName, phone,
			"❌ Número inválido. Digite apenas o *número do ticket* (ex: 1234):")
		return true
	}

	sessionToken, err := e.glpi.InitSession(e.cfg.UserToken)
	if err != nil {
		e.sendBotMessage(companyID, conversationID, instanceName, phone,
			"❌ Não consegui conectar ao sistema de chamados. Tente novamente mais tarde.")
		e.CancelFlow(conversationID)
		return true
	}
	defer e.glpi.KillSession(sessionToken)

	ticket, err := e.glpi.GetTicket(sessionToken, ticketID)
	if err != nil {
		e.sendBotMessage(companyID, conversationID, instanceName, phone,
			fmt.Sprintf("❌ Chamado #%d não encontrado. Verifique o número e tente novamente:", ticketID))
		return true
	}

	statusLabels := map[int]string{
		1: "🆕 Novo",
		2: "🔧 Em atendimento (atribuído)",
		3: "📋 Em atendimento (planejado)",
		4: "⏸️ Pendente",
		5: "✅ Solucionado",
		6: "🔒 Fechado",
	}

	statusLabel := statusLabels[ticket.Status]
	if statusLabel == "" {
		statusLabel = fmt.Sprintf("Status %d", ticket.Status)
	}

	msg := fmt.Sprintf(
		"🎫 *Chamado #%d*\n\n"+
			"📋 *Título:* %s\n"+
			"📊 *Status:* %s\n"+
			"📅 *Abertura:* %s\n"+
			"📅 *Última atualização:* %s",
		ticket.ID, ticket.Name, statusLabel, ticket.Date, ticket.DateMod,
	)

	e.sendBotMessage(companyID, conversationID, instanceName, phone, msg)
	e.CancelFlow(conversationID) // Done
	e.resumeBotAfterCompletion(companyID, conversationID, "", instanceName, phone, fmt.Sprintf("glpi_ticket_checked:%d", ticket.ID))
	return true
}

// --- Helpers ---

func (e *GLPIFlowEngine) sendBotMessage(companyID, conversationID, instanceName, phone, message string) {
	var externalID string
	instanceName = e.resolveConversationInstance(companyID, conversationID, instanceName)
	if instanceName == "" || phone == "" {
		e.addInternalNote(companyID, conversationID, fmt.Sprintf("Falha no envio GLPI: instancia='%s' telefone='%s'", instanceName, phone))
		return
	}

	externalID, err := e.evo.SendTextMessage(instanceName, phone, message)
	if err != nil {
		log.Printf("[GLPI-FLOW] Failed to send message to %s via %s: %v", phone, instanceName, err)
		e.addInternalNote(companyID, conversationID, fmt.Sprintf("Falha no envio GLPI para o cliente: %v", err))
		return
	}

	msgID := uuid.New().String()
	e.db.Exec(`
		INSERT INTO messages (id, conversation_id, company_id, sender_type, content, message_type, external_id, status)
		VALUES ($1, $2, $3, 'bot', $4, 'text', $5, 'sent')
	`, msgID, conversationID, companyID, message, externalID)

	e.db.Exec("UPDATE conversations SET last_message_at = NOW(), last_message_preview = $1 WHERE id = $2", message, conversationID)

	e.wsHub.BroadcastToCompany(companyID, "new_message", map[string]interface{}{
		"id": msgID, "conversation_id": conversationID,
		"sender_type": "bot", "content": message, "message_type": "text",
		"sender_name": "Assistente GLPI",
		"status":      "sent", "created_at": time.Now(),
	})

	log.Printf("[GLPI-FLOW] Sent message to %s: %s", phone, message[:min(50, len(message))])
}

func (e *GLPIFlowEngine) resolveConversationInstance(companyID, conversationID, fallback string) string {
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
			log.Printf("[GLPI-FLOW] Using conversation instance %s instead of stale instance %s for conversation %s", instanceName, fallback, conversationID)
		}
		return instanceName
	}
	return strings.TrimSpace(fallback)
}

func (e *GLPIFlowEngine) addInternalNote(companyID, conversationID, content string) {
	msgID := uuid.New().String()
	_, err := e.db.Exec(`
		INSERT INTO messages (id, conversation_id, company_id, sender_type, content, message_type, is_private, status)
		VALUES ($1, $2, $3, 'bot', $4, 'text', true, 'sent')
	`, msgID, conversationID, companyID, content)
	if err != nil {
		log.Printf("[GLPI-FLOW] Failed to save internal note: %v", err)
		return
	}

	e.wsHub.BroadcastToCompany(companyID, "new_message", map[string]interface{}{
		"id":              msgID,
		"conversation_id": conversationID,
		"sender_type":     "bot",
		"content":         content,
		"message_type":    "text",
		"is_private":      true,
		"sender_name":     "Assistente GLPI",
		"status":          "sent",
		"created_at":      time.Now(),
	})
}

func (e *GLPIFlowEngine) resumeBotAfterCompletion(companyID, conversationID, contactID, instanceName, phone, resultMessage string) {
	if e.bot == nil {
		return
	}
	if contactID == "" {
		_ = e.db.QueryRow("SELECT contact_id::text FROM conversations WHERE id = $1 AND company_id = $2", conversationID, companyID).Scan(&contactID)
	}
	e.bot.ResumeAfterExternalNode(companyID, conversationID, contactID, instanceName, phone, resultMessage)
}

func (e *GLPIFlowEngine) saveState(conversationID string, state *GLPIFlowState) {
	stateJSON, _ := json.Marshal(state)
	e.db.Exec(`
		INSERT INTO glpi_flow_states (conversation_id, state, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (conversation_id) DO UPDATE SET state = $2, updated_at = NOW()
	`, conversationID, string(stateJSON))
}

func (e *GLPIFlowEngine) loadState(conversationID string) *GLPIFlowState {
	var stateStr string
	err := e.db.QueryRow("SELECT state FROM glpi_flow_states WHERE conversation_id = $1", conversationID).Scan(&stateStr)
	if err != nil {
		return nil
	}
	var state GLPIFlowState
	if err := json.Unmarshal([]byte(stateStr), &state); err != nil {
		return nil
	}
	return &state
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
