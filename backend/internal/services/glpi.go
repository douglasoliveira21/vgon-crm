package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
			"name":         title,
			"content":      content,
			"entities_id":  entityID,
			"type":         ticketType,
			"priority":     priority,
			"status":       1, // New
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

// KillSession ends a GLPI session
func (g *GLPIService) KillSession(sessionToken string) {
	req, _ := http.NewRequest("GET", g.baseURL+"/killSession", nil)
	req.Header.Set("App-Token", g.appToken)
	req.Header.Set("Session-Token", sessionToken)
	http.DefaultClient.Do(req)
}
