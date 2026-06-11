package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// SyncContacts fetches contacts from Evolution API and saves to database
func (s *EvolutionService) SyncContacts(instanceName, companyID string) (int, error) {
	// Fetch contacts from Evolution API
	httpReq, err := http.NewRequest("GET", fmt.Sprintf("%s/chat/fetchAllContacts/%s", s.cfg.EvolutionAPIURL, instanceName), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("apikey", s.cfg.EvolutionAPIKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch contacts: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Evolution API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var contacts []map[string]interface{}
	if err := json.Unmarshal(respBody, &contacts); err != nil {
		// Try wrapping format
		var wrapped map[string]interface{}
		if err2 := json.Unmarshal(respBody, &wrapped); err2 == nil {
			if arr, ok := wrapped["contacts"].([]interface{}); ok {
				for _, c := range arr {
					if cm, ok := c.(map[string]interface{}); ok {
						contacts = append(contacts, cm)
					}
				}
			}
		}
		if len(contacts) == 0 {
			return 0, fmt.Errorf("failed to parse contacts: %w", err)
		}
	}

	synced := 0
	for _, contact := range contacts {
		jid, _ := contact["id"].(string)
		if jid == "" {
			jid, _ = contact["remoteJid"].(string)
		}
		if jid == "" {
			continue
		}

		// Skip groups and status
		if len(jid) < 5 || jid == "status@broadcast" {
			continue
		}
		if len(jid) > 20 && jid[len(jid)-15:] == "@g.us" {
			continue
		}

		phone := extractPhoneFromJid(jid)
		if phone == "" || len(phone) < 8 {
			continue
		}

		name, _ := contact["pushName"].(string)
		if name == "" {
			name, _ = contact["name"].(string)
		}
		if name == "" {
			name, _ = contact["notify"].(string)
		}
		if name == "" {
			name = phone
		}

		// Check if contact exists
		var existingID string
		err := s.db.QueryRow("SELECT id FROM contacts WHERE company_id = $1 AND phone = $2", companyID, phone).Scan(&existingID)
		if err == nil {
			// Update name if it was just the phone
			s.db.Exec("UPDATE contacts SET name = CASE WHEN name = phone THEN $1 ELSE name END, updated_at = NOW() WHERE id = $2", name, existingID)
			synced++
			continue
		}

		// Create contact
		contactID := uuid.New().String()
		s.db.Exec(`
			INSERT INTO contacts (id, company_id, name, phone, origin)
			VALUES ($1, $2, $3, $4, 'whatsapp')
			ON CONFLICT DO NOTHING
		`, contactID, companyID, name, phone)
		synced++
	}

	log.Printf("[SYNC] Synced %d contacts for instance %s", synced, instanceName)
	return synced, nil
}

// SyncContactPhoto fetches profile photo for a contact from Evolution API
func (s *EvolutionService) SyncContactPhoto(instanceName, phone, contactID string) (string, error) {
	payload := map[string]interface{}{
		"number": phone,
	}

	body, _ := json.Marshal(payload)

	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s/chat/fetchProfilePictureUrl/%s", s.cfg.EvolutionAPIURL, instanceName), bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apikey", s.cfg.EvolutionAPIKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to fetch profile picture: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("no profile picture available")
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	pictureURL, _ := result["profilePictureUrl"].(string)
	if pictureURL == "" {
		pictureURL, _ = result["picture"].(string)
		if pictureURL == "" {
			if wurl, ok := result["wpiUrl"].(string); ok {
				pictureURL = wurl
			}
		}
	}

	if pictureURL == "" {
		return "", fmt.Errorf("no profile picture URL")
	}

	// Download the photo and save locally
	photoResp, err := http.Get(pictureURL)
	if err != nil {
		return "", fmt.Errorf("failed to download photo: %w", err)
	}
	defer photoResp.Body.Close()

	if photoResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("photo download failed with status %d", photoResp.StatusCode)
	}

	// Save to disk
	avatarsDir := "/app/uploads/avatars"
	os.MkdirAll(avatarsDir, 0755)

	fileName := contactID + ".jpg"
	filePath := filepath.Join(avatarsDir, fileName)

	photoData, _ := io.ReadAll(photoResp.Body)
	if err := os.WriteFile(filePath, photoData, 0644); err != nil {
		return "", fmt.Errorf("failed to save photo: %w", err)
	}

	// Update contact avatar in database
	avatarURL := "/uploads/avatars/" + fileName
	s.db.Exec("UPDATE contacts SET avatar_url = $1, updated_at = NOW() WHERE id = $2", avatarURL, contactID)

	return avatarURL, nil
}

// SyncAllContactPhotos syncs photos for all contacts of a company
func (s *EvolutionService) SyncAllContactPhotos(instanceName, companyID string) {
	rows, err := s.db.Query("SELECT id, phone FROM contacts WHERE company_id = $1 AND avatar_url IS NULL AND phone IS NOT NULL", companyID)
	if err != nil {
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, phone string
		rows.Scan(&id, &phone)

		_, err := s.SyncContactPhoto(instanceName, phone, id)
		if err == nil {
			count++
		}

		// Rate limit - don't hammer the API
		if count > 50 {
			break
		}
	}

	log.Printf("[SYNC] Synced %d contact photos for instance %s", count, instanceName)
}
