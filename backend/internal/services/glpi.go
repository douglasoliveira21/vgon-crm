package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type GLPIService struct {
	baseURL  string
	appToken string
}

type GLPISession struct {
	SessionToken string `json:"session_token"`
}

type GLPITicket struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Content     string `json:"content"`
	Status      int    `json:"status"`
	EntityID    int    `json:"entities_id"`
	Date        string `json:"date"`
	DateMod     string `json:"date_mod"`
	Type        int    `json:"type"`
	Priority    int    `json:"priority"`
	RequestType int    `json:"requesttypes_id"`
}

type GLPIEntity struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	CompleteName string `json:"completename"`
}

type GLPISearchOption struct {
	Name string `json:"name"`
	UID  string `json:"uid"`
}

type glpiSearchResponse struct {
	Data []map[string]interface{} `json:"data"`
}

func NewGLPIService(baseURL, appToken string) *GLPIService {
	// Remove trailing slash if present
	baseURL = strings.TrimRight(baseURL, "/")
	return &GLPIService{
		baseURL:  baseURL,
		appToken: appToken,
	}
}

// InitSession creates a session with GLPI using user_token
func (g *GLPIService) InitSession(userToken string) (string, error) {
	req, err := http.NewRequest("GET", g.baseURL+"/initSession", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("App-Token", g.appToken)
	req.Header.Set("Authorization", "user_token "+userToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to GLPI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GLPI auth failed (%d): %s", resp.StatusCode, string(body))
	}

	var session GLPISession
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return "", err
	}
	return session.SessionToken, nil
}

// CreateTicket opens a new ticket in GLPI
func (g *GLPIService) CreateTicket(sessionToken string, title, content string, entityID, ticketType, priority int) (*GLPITicket, error) {
	payload := map[string]interface{}{
		"input": map[string]interface{}{
			"name":        title,
			"content":     content,
			"entities_id": entityID,
			"type":        ticketType,
			"priority":    priority,
			"status":      1, // New
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", g.baseURL+"/Ticket", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("App-Token", g.appToken)
	req.Header.Set("Session-Token", sessionToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GLPI create ticket failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Get the created ticket ID
	ticketID := 0
	if id, ok := result["id"].(float64); ok {
		ticketID = int(id)
	}

	return &GLPITicket{ID: ticketID, Name: title, Content: content, EntityID: entityID, Status: 1}, nil
}

// GetTicket retrieves a ticket by ID
func (g *GLPIService) GetTicket(sessionToken string, ticketID int) (*GLPITicket, error) {
	url := fmt.Sprintf("%s/Ticket/%d", g.baseURL, ticketID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("App-Token", g.appToken)
	req.Header.Set("Session-Token", sessionToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GLPI get ticket failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var ticket GLPITicket
	if err := json.NewDecoder(resp.Body).Decode(&ticket); err != nil {
		return nil, err
	}
	return &ticket, nil
}

// GetEntities retrieves entities available in GLPI
func (g *GLPIService) GetEntities(sessionToken string) ([]GLPIEntity, error) {
	req, err := http.NewRequest("GET", g.baseURL+"/Entity?range=0-100", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("App-Token", g.appToken)
	req.Header.Set("Session-Token", sessionToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get entities: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GLPI get entities failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var entities []GLPIEntity
	if err := json.NewDecoder(resp.Body).Decode(&entities); err != nil {
		return nil, err
	}
	return entities, nil
}

// GetEntity retrieves a specific entity by ID
func (g *GLPIService) GetEntity(sessionToken string, entityID int) (*GLPIEntity, error) {
	url := fmt.Sprintf("%s/Entity/%d", g.baseURL, entityID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("App-Token", g.appToken)
	req.Header.Set("Session-Token", sessionToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GLPI get entity failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var entity GLPIEntity
	if err := json.NewDecoder(resp.Body).Decode(&entity); err != nil {
		return nil, err
	}
	return &entity, nil
}

func (g *GLPIService) FindEntitiesByCNPJ(sessionToken, cnpj string) ([]GLPIEntity, error) {
	cnpj = onlyDigits(cnpj)
	if len(cnpj) != 14 {
		return nil, fmt.Errorf("CNPJ deve conter 14 dígitos")
	}

	options, err := g.listSearchOptions(sessionToken, "Entity")
	if err != nil {
		return nil, err
	}
	cnpjField := findSearchOption(options, "cnpj")
	if cnpjField == "" {
		return nil, fmt.Errorf("campo CNPJ não encontrado nas opções de pesquisa da entidade no GLPI")
	}
	idField := findEntityIDSearchOption(options)
	if idField == "" {
		return nil, fmt.Errorf("campo ID da entidade não encontrado nas opções de pesquisa do GLPI")
	}

	query := url.Values{}
	query.Set("criteria[0][field]", cnpjField)
	query.Set("criteria[0][searchtype]", "contains")
	query.Set("criteria[0][value]", "^"+cnpj+"$")
	query.Set("forcedisplay[0]", idField)
	query.Set("forcedisplay[1]", cnpjField)
	query.Set("range", "0-49")
	searchURL := g.baseURL + "/search/Entity?" + query.Encode()

	var result glpiSearchResponse
	if err := g.getJSON(sessionToken, searchURL, &result); err != nil {
		return nil, fmt.Errorf("falha ao pesquisar CNPJ nas entidades: %w", err)
	}

	entityIDs := map[int]struct{}{}
	for _, row := range result.Data {
		if rawCNPJ, exists := row[cnpjField]; exists && onlyDigits(fmt.Sprint(rawCNPJ)) != cnpj {
			continue
		}
		entityID, ok := interfaceToInt(row[idField])
		if !ok {
			entityID, ok = interfaceToInt(row["2"])
		}
		if ok && entityID > 0 {
			entityIDs[entityID] = struct{}{}
		}
	}

	ids := make([]int, 0, len(entityIDs))
	for id := range entityIDs {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	entities := make([]GLPIEntity, 0, len(ids))
	for _, id := range ids {
		entity, err := g.GetEntity(sessionToken, id)
		if err != nil {
			return nil, err
		}
		entities = append(entities, *entity)
	}
	return entities, nil
}

func (g *GLPIService) listSearchOptions(sessionToken, itemType string) (map[string]GLPISearchOption, error) {
	var options map[string]GLPISearchOption
	if err := g.getJSON(sessionToken, g.baseURL+"/listSearchOptions/"+url.PathEscape(itemType), &options); err != nil {
		return nil, fmt.Errorf("falha ao consultar campos pesquisáveis do GLPI: %w", err)
	}
	return options, nil
}

func (g *GLPIService) getJSON(sessionToken, requestURL string, target interface{}) error {
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("App-Token", g.appToken)
	req.Header.Set("Session-Token", sessionToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GLPI retornou %d: %s", resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func findSearchOption(options map[string]GLPISearchOption, needle string) string {
	needle = strings.ToLower(strings.TrimSpace(needle))
	keys := make([]string, 0, len(options))
	for key := range options {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		option := options[key]
		if strings.EqualFold(strings.TrimSpace(option.Name), needle) {
			return key
		}
	}
	for _, key := range keys {
		option := options[key]
		text := strings.ToLower(option.Name + " " + option.UID)
		if strings.Contains(text, needle) {
			return key
		}
	}
	return ""
}

func findEntityIDSearchOption(options map[string]GLPISearchOption) string {
	for key, option := range options {
		if strings.EqualFold(strings.TrimSpace(option.UID), "Entity.id") {
			return key
		}
	}
	for key, option := range options {
		if strings.EqualFold(strings.TrimSpace(option.Name), "ID") {
			return key
		}
	}
	return ""
}

func interfaceToInt(value interface{}) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	case string:
		number, err := strconv.Atoi(strings.TrimSpace(typed))
		return number, err == nil
	default:
		return 0, false
	}
}

func onlyDigits(value string) string {
	var result strings.Builder
	for _, char := range value {
		if char >= '0' && char <= '9' {
			result.WriteRune(char)
		}
	}
	return result.String()
}

// KillSession ends a GLPI session
func (g *GLPIService) KillSession(sessionToken string) {
	req, _ := http.NewRequest("GET", g.baseURL+"/killSession", nil)
	req.Header.Set("App-Token", g.appToken)
	req.Header.Set("Session-Token", sessionToken)
	http.DefaultClient.Do(req)
}
