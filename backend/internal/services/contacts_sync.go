package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SyncContacts fetches contacts from Evolution API and saves to database
func (s *EvolutionService) SyncContacts(instanceName, companyID string) (int, error) {
	// Try multiple endpoint formats (varies by Evolution API version)
	endpoints := []string{
		fmt.Sprintf("%s/chat/findContacts/%s", s.cfg.EvolutionAPIURL, instanceName),
		fmt.Sprintf("%s/chat/fetchAllContacts/%s", s.cfg.EvolutionAPIURL, instanceName),
	}

	var respBody []byte
	var success bool

	for _, endpoint := range endpoints {
		httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer([]byte("{}")))
		if err != nil {
			continue
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("apikey", s.cfg.EvolutionAPIKey)

		resp, err := s.client.Do(httpReq)
		if err != nil {
			continue
		}
		respBody, _ = io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			success = true
			break
		}
	}

	if !success {
		return 0, fmt.Errorf("failed to fetch contacts from Evolution API: %s", string(respBody))
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
	skipped := 0
	for _, contact := range contacts {
		// Try to get the phone number from different fields
		var phone string

		// Try remoteJid first (format: 5531999999999@s.whatsapp.net)
		if jid, ok := contact["remoteJid"].(string); ok && jid != "" {
			phone = extractPhoneFromJid(jid)
		}
		if phone == "" {
			if jid, ok := contact["jid"].(string); ok && jid != "" {
				phone = extractPhoneFromJid(jid)
			}
		}
		// Try id field - could be JID or internal ID
		if phone == "" {
			if id, ok := contact["id"].(string); ok && id != "" {
				// Only use if it looks like a JID (contains @)
				if strings.Contains(id, "@") {
					phone = extractPhoneFromJid(id)
				}
			}
		}
		// Try owner field
		if phone == "" {
			if owner, ok := contact["owner"].(string); ok && strings.Contains(owner, "@") {
				// Skip - this is our own number
			}
		}

		// Validate: phone must be numeric and reasonable length
		if phone == "" || len(phone) < 8 || len(phone) > 15 {
			skipped++
			continue
		}
		// Check if phone is actually numeric
		isNumeric := true
		for _, c := range phone {
			if c < '0' || c > '9' {
				isNumeric = false
				break
			}
		}
		if !isNumeric {
			skipped++
			continue
		}

		// Skip groups and broadcasts
		name, _ := contact["pushName"].(string)
		if name == "" {
			name, _ = contact["name"].(string)
		}
		if name == "" {
			name, _ = contact["notify"].(string)
		}
		if name == "" {
			name, _ = contact["verifiedName"].(string)
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
		_, insertErr := s.db.Exec(`
			INSERT INTO contacts (id, company_id, name, phone, origin)
			VALUES ($1, $2, $3, $4, 'whatsapp')
		`, contactID, companyID, name, phone)
		if insertErr != nil {
			log.Printf("[SYNC] Failed to insert contact %s (%s): %v", name, phone, insertErr)
		} else {
			synced++
		}
	}

	log.Printf("[SYNC] Synced %d contacts for instance %s (skipped %d, total from API: %d)", synced, instanceName, skipped, len(contacts))
	return synced, nil
}

// SyncContactPhoto fetches profile photo for a contact from Evolution API and saves locally
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

	// Download the image and save locally
	localPath, err := s.downloadAndSaveAvatar(pictureURL, contactID)
	if err != nil {
		// Fallback: save the external URL
		s.db.Exec("UPDATE contacts SET avatar_url = $1, updated_at = NOW() WHERE id = $2", pictureURL, contactID)
		return pictureURL, nil
	}

	// Save local path in database
	s.db.Exec("UPDATE contacts SET avatar_url = $1, updated_at = NOW() WHERE id = $2", localPath, contactID)

	return localPath, nil
}

// downloadAndSaveAvatar downloads an image from URL and saves it to local storage
func (s *EvolutionService) downloadAndSaveAvatar(imageURL, contactID string) (string, error) {
	// Download image
	resp, err := http.Get(imageURL)
	if err != nil {
		return "", fmt.Errorf("failed to download avatar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read image data: %w", err)
	}

	if len(imageData) == 0 {
		return "", fmt.Errorf("empty image data")
	}

	// Create directory structure: /app/uploads/avatars/
	dir := "/app/uploads/avatars"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create avatar directory: %w", err)
	}

	// Save file
	fileName := fmt.Sprintf("%s.jpg", contactID)
	filePath := fmt.Sprintf("%s/%s", dir, fileName)

	if err := os.WriteFile(filePath, imageData, 0644); err != nil {
		return "", fmt.Errorf("failed to save avatar: %w", err)
	}

	// Return the URL path (served by static files)
	return fmt.Sprintf("/uploads/avatars/%s", fileName), nil
}

// SyncAllContactPhotos syncs photos for all contacts of a company
func (s *EvolutionService) SyncAllContactPhotos(instanceName, companyID string) {
	// Sync contacts without photo OR with photo older than 24h
	rows, err := s.db.Query(`
		SELECT id, phone FROM contacts 
		WHERE company_id = $1 AND phone IS NOT NULL 
		AND (avatar_url IS NULL OR updated_at < NOW() - INTERVAL '24 hours')
		LIMIT 100
	`, companyID)
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
		time.Sleep(500 * time.Millisecond)
		if count > 100 {
			break
		}
	}

	log.Printf("[SYNC] Synced %d contact photos for instance %s", count, instanceName)
}

// StartPeriodicPhotoSync starts a background goroutine that syncs contact photos every 6 hours
func (s *EvolutionService) StartPeriodicPhotoSync() {
	go func() {
		// Wait 2 minutes before first run (let server fully start)
		time.Sleep(2 * time.Minute)

		for {
			s.syncAllCompanyPhotos()
			// Run every 6 hours
			time.Sleep(6 * time.Hour)
		}
	}()
	log.Println("[SYNC] Periodic photo sync started (every 6 hours)")
}

func (s *EvolutionService) syncAllCompanyPhotos() {
	// Get all active instances
	rows, err := s.db.Query(`
		SELECT wi.instance_name, c.id as company_id
		FROM whatsapp_instances wi
		JOIN channels ch ON wi.channel_id = ch.id
		JOIN companies c ON ch.company_id = c.id
		WHERE ch.status = 'connected' AND c.is_active = true
	`)
	if err != nil {
		log.Printf("[SYNC] Failed to get instances for photo sync: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var instanceName, companyID string
		rows.Scan(&instanceName, &companyID)
		s.SyncAllContactPhotos(instanceName, companyID)
		// Pause between companies
		time.Sleep(5 * time.Second)
	}
}
