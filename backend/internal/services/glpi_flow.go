package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/evocrm/backend/internal/safe"
	"github.com/evocrm/backend/internal/websocket"
	"github.com/google/uuid"
)

// GLPIFlowState tracks the multi-step GLPI ticket creation flow per conversation
type GLPIFlowState struct {
	Step         string `json:"step"`          // current step
	Mode         string `json:"mode"`          // flow mode: open_ticket | check_status | check_cnpj
	ContactID    string `json:"contact_id"`    // contact for resuming the parent bot flow
	EntitySearch string `json:"entity_search"` // legacy entity name search
	EntityCNPJ   string `json:"entity_cnpj"`   // normalized CNPJ searched
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
	// mode: "open_ticket", "check_status" or "check_cnpj"
	state := &GLPIFlowState{Mode: mode, ContactID: contactID, Phone: phone}

	if mode == "check_status" {
		state.Step = "ask_ticket_number"
		e.saveState(conversationID, state)
		e.sendBotMessage(companyID, conversationID, instanceName, phone,
			"🔍 *Consultar Chamado*\n\nPor favor, digite o *número do ticket* que deseja consultar:")
		return
	}

	if mode == "check_cnpj" {
		// If this contact already had a CNPJ validated against GLPI before, reuse
		// it instead of asking again.
		if cnpj, entityID, entityName, ok := e.loadValidatedCNPJ(contactID); ok {
			e.sendBotMessage(companyID, conversationID, instanceName, phone,
				fmt.Sprintf("✅ Você já está identificado como cliente:\n\n🏢 *Empresa:* %s\n📄 *CNPJ:* %s\n\nVou continuar o seu atendimento. 🙂", entityName, formatCNPJ(cnpj)))
			e.resumeBotWhenWaiting(companyID, conversationID, contactID, instanceName, phone, fmt.Sprintf("glpi_cnpj_found:%d", entityID))
			return
		}

		state.Step = "cnpj_lookup"
		e.saveState(conversationID, state)
		e.sendBotMessage(companyID, conversationID, instanceName, phone,
			"🔎 *Consulta de Cliente*\n\nPara verificar se você já é nosso cliente, digite o *CNPJ da empresa* (com ou sem pontuação):")
		return
	}

	// Default: open_ticket
	state.Phone = phone

	// A CNPJ already validated for this contact (by the "consultar CNPJ" block
	// earlier in the flow, or in a past conversation) is reused. Asking for it
	// again right after the customer had just typed it made the flow feel
	// broken, and gave them a second chance to mistype it.
	if cnpj, entityID, entityName, ok := e.loadValidatedCNPJ(contactID); ok {
		state.EntityCNPJ = cnpj
		state.EntityID = entityID
		state.EntityName = entityName
		state.Step = "ask_full_name"
		e.saveState(conversationID, state)
		e.sendBotMessage(companyID, conversationID, instanceName, phone,
			fmt.Sprintf("🎫 *Abertura de Chamado*\n\n✅ Empresa identificada: *%s*\nCNPJ: *%s*\n\nAgora, informe seu *nome completo*:", entityName, formatCNPJ(cnpj)))
		return
	}

	state.Step = "ask_entity"
	e.saveState(conversationID, state)

	e.sendBotMessage(companyID, conversationID, instanceName, phone,
		"🎫 *Abertura de Chamado*\n\nPara abrir um chamado, preciso de algumas informações.\n\nPrimeiro, digite o *CNPJ da empresa* (com ou sem pontuação):")
}

// HandleGLPIMessage processes user responses during the GLPI flow
func (e *GLPIFlowEngine) HandleGLPIMessage(companyID, conversationID, contactID, instanceName, phone, message string) bool {
	state := e.loadState(conversationID)
	if state == nil {
		return false // No active GLPI flow
	}

	switch state.Step {
	case "cnpj_lookup":
		return e.handleCNPJLookup(companyID, conversationID, contactID, instanceName, phone, message, state)
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

// handleCNPJLookup runs the dedicated "sou cliente?" lookup block. It only
// checks whether the CNPJ exists as an Entity in GLPI, confirms the customer
// data when found, informs the user when not found, and in every terminal case
// resumes the parent bot flow so it can branch (found vs not_found) and move on.
func (e *GLPIFlowEngine) handleCNPJLookup(companyID, conversationID, contactID, instanceName, phone, message string, state *GLPIFlowState) bool {
	cnpj := onlyDigits(message)
	if len(cnpj) != 14 {
		e.sendBotMessage(companyID, conversationID, instanceName, phone,
			"❌ CNPJ inválido. Digite os *14 números do CNPJ*, com ou sem pontuação:")
		return true
	}
	state.EntityCNPJ = cnpj
	if contactID == "" {
		contactID = state.ContactID
	}

	sessionToken, err := e.glpi.InitSession(e.cfg.UserToken)
	if err != nil {
		log.Printf("[GLPI-FLOW] CNPJ lookup: failed to init session: %v", err)
		e.sendBotMessage(companyID, conversationID, instanceName, phone,
			"⚠️ Não consegui consultar a base de clientes agora. Vou seguir com o atendimento mesmo assim.")
		e.finishCNPJLookup(companyID, conversationID, contactID, instanceName, phone, "glpi_cnpj_not_found:error")
		return true
	}
	defer e.glpi.KillSession(sessionToken)

	matches, err := e.glpi.FindEntitiesByCNPJ(sessionToken, cnpj)
	if err != nil {
		log.Printf("[GLPI-FLOW] CNPJ lookup failed: %v", err)
		e.sendBotMessage(companyID, conversationID, instanceName, phone,
			"⚠️ Não consegui consultar a base de clientes agora. Vou seguir com o atendimento mesmo assim.")
		e.finishCNPJLookup(companyID, conversationID, contactID, instanceName, phone, "glpi_cnpj_not_found:error")
		return true
	}

	if len(matches) == 0 {
		e.sendBotMessage(companyID, conversationID, instanceName, phone,
			fmt.Sprintf("ℹ️ Não localizei o CNPJ *%s* na nossa base como cliente.\n\nSem problemas, vou dar continuidade ao seu atendimento.", formatCNPJ(cnpj)))
		e.finishCNPJLookup(companyID, conversationID, contactID, instanceName, phone, "glpi_cnpj_not_found")
		return true
	}

	// Found as customer: confirm the data back to the user.
	entity := matches[0]
	entityName := entity.CompleteName
	if entityName == "" {
		entityName = entity.Name
	}
	state.EntityID = entity.ID
	state.EntityName = entityName

	e.sendBotMessage(companyID, conversationID, instanceName, phone, buildCustomerFoundMessage(entity, cnpj))

	// Persist the VALIDATED CNPJ on the CRM contact so future conversations with
	// this phone number skip the CNPJ question entirely. Only runs here, in the
	// matches-found branch, so unvalidated/not-found CNPJs are never saved.
	if contactID != "" {
		e.db.Exec(`
			UPDATE contacts
			SET glpi_cnpj = $1,
			    glpi_entity_id = $2,
			    glpi_entity_name = $3,
			    glpi_validated_at = NOW(),
			    company_name = CASE WHEN company_name IS NULL OR company_name = '' THEN $3 ELSE company_name END,
			    updated_at = NOW()
			WHERE id = $4
		`, cnpj, entity.ID, entityName, contactID)
	}

	e.finishCNPJLookup(companyID, conversationID, contactID, instanceName, phone, fmt.Sprintf("glpi_cnpj_found:%d", entity.ID))
	return true
}

// loadValidatedCNPJ returns a previously validated GLPI CNPJ for the contact,
// so the flow can skip asking for it again. Returns ok=false when the contact
// has no stored CNPJ (or when contactID is empty).
func (e *GLPIFlowEngine) loadValidatedCNPJ(contactID string) (cnpj string, entityID int, entityName string, ok bool) {
	if contactID == "" {
		return "", 0, "", false
	}
	var storedCNPJ sql.NullString
	var storedID sql.NullInt64
	var storedName sql.NullString
	err := e.db.QueryRow(`
		SELECT glpi_cnpj, glpi_entity_id, glpi_entity_name
		FROM contacts
		WHERE id = $1
	`, contactID).Scan(&storedCNPJ, &storedID, &storedName)
	if err != nil {
		return "", 0, "", false
	}
	cnpj = onlyDigits(storedCNPJ.String)
	if len(cnpj) != 14 || !storedID.Valid || storedID.Int64 <= 0 {
		return "", 0, "", false
	}
	return cnpj, int(storedID.Int64), strings.TrimSpace(storedName.String), true
}

// finishCNPJLookup ends the dedicated lookup flow and hands control back to the
// parent bot flow with a result string so it can take the found/not_found edge.
func (e *GLPIFlowEngine) finishCNPJLookup(companyID, conversationID, contactID, instanceName, phone, result string) {
	e.CancelFlow(conversationID)
	e.resumeBotAfterCompletion(companyID, conversationID, contactID, instanceName, phone, result)
}

// buildCustomerFoundMessage renders the confirmation shown when the CNPJ is
// found in the GLPI customer base, including whatever cadastral data exists.
func buildCustomerFoundMessage(entity GLPIEntity, cnpj string) string {
	entityName := entity.CompleteName
	if entityName == "" {
		entityName = entity.Name
	}

	msg := "✅ *Cliente localizado na nossa base!*\n\n"
	msg += fmt.Sprintf("🏢 *Empresa:* %s\n", entityName)
	msg += fmt.Sprintf("📄 *CNPJ:* %s\n", formatCNPJ(cnpj))
	if strings.TrimSpace(entity.Phone) != "" {
		msg += fmt.Sprintf("📞 *Telefone:* %s\n", strings.TrimSpace(entity.Phone))
	}
	if strings.TrimSpace(entity.Email) != "" {
		msg += fmt.Sprintf("📧 *E-mail:* %s\n", strings.TrimSpace(entity.Email))
	}
	location := joinNonEmpty(" - ", strings.TrimSpace(entity.Town), strings.TrimSpace(entity.State))
	if strings.TrimSpace(entity.Address) != "" {
		msg += fmt.Sprintf("📍 *Endereço:* %s\n", strings.TrimSpace(entity.Address))
	}
	if location != "" {
		msg += fmt.Sprintf("🗺️ *Local:* %s\n", location)
	}
	msg += "\nDados confirmados. Vou continuar o seu atendimento. 🙂"
	return msg
}

// joinNonEmpty joins the non-empty parts with the given separator.
func joinNonEmpty(sep string, parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			kept = append(kept, strings.TrimSpace(p))
		}
	}
	return strings.Join(kept, sep)
}

func (e *GLPIFlowEngine) handleAskEntity(companyID, conversationID, instanceName, phone, message string, state *GLPIFlowState) bool {
	state.EntityCNPJ = onlyDigits(message)
	if len(state.EntityCNPJ) != 14 {
		e.sendBotMessage(companyID, conversationID, instanceName, phone,
			"❌ CNPJ inválido. Digite os *14 números do CNPJ*, com ou sem pontuação:")
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

	matches, err := e.glpi.FindEntitiesByCNPJ(sessionToken, state.EntityCNPJ)
	if err != nil {
		log.Printf("[GLPI-FLOW] Failed to search entity by CNPJ: %v", err)
		e.sendBotMessage(companyID, conversationID, instanceName, phone,
			"❌ Não consegui consultar o CNPJ no GLPI. Verifique se o campo *CNPJ* está disponível para pesquisa e tente novamente:")
		return true
	}

	if len(matches) == 0 {
		e.sendBotMessage(companyID, conversationID, instanceName, phone,
			fmt.Sprintf("❌ Nenhuma empresa encontrada com o CNPJ *%s*.\n\nConfira os números e digite novamente:", formatCNPJ(state.EntityCNPJ)))
		return true
	}

	if len(matches) == 1 {
		return e.confirmEntityAndAskName(companyID, conversationID, instanceName, phone, state, matches[0])
	}

	// Build options message
	msg := fmt.Sprintf("🏢 Encontrei %d entidades com o CNPJ *%s*:\n", len(matches), formatCNPJ(state.EntityCNPJ))
	for i, ent := range matches {
		if i >= 5 {
			msg += fmt.Sprintf("\n... e mais %d resultados.", len(matches)-5)
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
			"Ok! Digite novamente o *CNPJ da empresa*:")
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

	return e.confirmEntityAndAskName(companyID, conversationID, instanceName, phone, state, GLPIEntity{
		ID: entityID, Name: entityName, CompleteName: entityName,
	})
}

func (e *GLPIFlowEngine) confirmEntityAndAskName(companyID, conversationID, instanceName, phone string, state *GLPIFlowState, entity GLPIEntity) bool {
	entityName := entity.CompleteName
	if entityName == "" {
		entityName = entity.Name
	}
	state.EntityID = entity.ID
	state.EntityName = entityName
	state.Step = "ask_full_name"
	e.saveState(conversationID, state)

	// Remember the validated CNPJ so this contact isn't asked again, here or
	// in a later conversation.
	if state.ContactID != "" && len(state.EntityCNPJ) == 14 {
		if _, err := e.db.Exec(`
			UPDATE contacts
			SET glpi_cnpj = $1,
			    glpi_entity_id = $2,
			    glpi_entity_name = $3,
			    glpi_validated_at = NOW(),
			    company_name = CASE WHEN company_name IS NULL OR company_name = '' THEN $3 ELSE company_name END,
			    updated_at = NOW()
			WHERE id = $4
		`, state.EntityCNPJ, entity.ID, entityName, state.ContactID); err != nil {
			log.Printf("[GLPI-FLOW] failed to save validated CNPJ for contact %s: %v", state.ContactID, err)
		}
	}
	e.sendBotMessage(companyID, conversationID, instanceName, phone,
		fmt.Sprintf("✅ Empresa identificada: *%s*\nCNPJ: *%s*\n\nAgora, informe seu *nome completo*:", entityName, formatCNPJ(state.EntityCNPJ)))
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

func formatCNPJ(cnpj string) string {
	cnpj = onlyDigits(cnpj)
	if len(cnpj) != 14 {
		return cnpj
	}
	return fmt.Sprintf("%s.%s.%s/%s-%s", cnpj[0:2], cnpj[2:5], cnpj[5:8], cnpj[8:12], cnpj[12:14])
}

// resumeBotWhenWaiting continues the parent bot flow once it has been marked
// as waiting on this block.
//
// The "already identified" shortcut finishes while the block is still
// executing, and the executor only marks the execution as external_wait after
// the block returns. Resuming immediately found nothing to resume and gave up,
// after which the flow was marked as waiting forever: the customer saw
// "Você já está identificado" and then silence. This runs off the executor's
// goroutine (it must be free to set the marker) and waits for the marker.
func (e *GLPIFlowEngine) resumeBotWhenWaiting(companyID, conversationID, contactID, instanceName, phone, resultMessage string) {
	safe.Go("glpi-resume", func() {
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			var waiting bool
			if err := e.db.QueryRow(`
				SELECT EXISTS (
					SELECT 1 FROM bot_executions
					WHERE conversation_id = $1 AND status = 'external_wait'
				)
			`, conversationID).Scan(&waiting); err == nil && waiting {
				e.resumeBotAfterCompletion(companyID, conversationID, contactID, instanceName, phone, resultMessage)
				return
			}
			time.Sleep(200 * time.Millisecond)
		}
		log.Printf("[GLPI-FLOW] Gave up waiting for conversation %s to be marked external_wait; flow not resumed", conversationID)
	})
}
