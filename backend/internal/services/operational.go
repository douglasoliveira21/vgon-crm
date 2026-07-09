package services

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
)

func (s *EvolutionService) StartWhatsAppInstanceMonitor() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			s.monitorWhatsAppInstances()
		}
	}()
}

func (s *EvolutionService) monitorWhatsAppInstances() {
	rows, err := s.db.Query(`
		SELECT id, company_id, COALESCE(channel_id::text, ''), instance_name, COALESCE(status, 'disconnected')
		FROM whatsapp_instances
		WHERE COALESCE(instance_name, '') <> ''
	`)
	if err != nil {
		log.Printf("[WHATSAPP_MONITOR] failed to list instances: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, companyID, channelID, instanceName, previousStatus string
		if err := rows.Scan(&id, &companyID, &channelID, &instanceName, &previousStatus); err != nil {
			continue
		}

		status, err := s.GetConnectionStatus(instanceName)
		if err != nil {
			status = "error"
			_, _ = s.db.Exec(`
				UPDATE whatsapp_instances
				SET status = 'error', last_status_check_at = NOW(), last_status_error = $1, updated_at = NOW()
				WHERE id = $2
			`, err.Error(), id)
			if channelID != "" {
				_, _ = s.db.Exec("UPDATE channels SET status = 'error', updated_at = NOW() WHERE id = $1", channelID)
			}
			log.Printf("[WHATSAPP_MONITOR] failed to check %s: %v", instanceName, err)
		} else {
			_, _ = s.db.Exec(`
				UPDATE whatsapp_instances
				SET last_status_check_at = NOW(),
					last_status_error = NULL,
					last_known_connected_at = CASE WHEN $1 = 'connected' THEN NOW() ELSE last_known_connected_at END,
					disconnected_alerted_at = CASE WHEN $1 = 'connected' THEN NULL ELSE disconnected_alerted_at END,
					updated_at = NOW()
				WHERE id = $2
			`, status, id)
		}

		if previousStatus == "connected" && status != "connected" {
			s.createWhatsAppDownAlert(companyID, id, channelID, instanceName, status)
		}
	}
}

func (s *EvolutionService) createWhatsAppDownAlert(companyID, instanceID, channelID, instanceName, status string) {
	var alreadyAlerted sql.NullTime
	_ = s.db.QueryRow("SELECT disconnected_alerted_at FROM whatsapp_instances WHERE id = $1", instanceID).Scan(&alreadyAlerted)
	if alreadyAlerted.Valid && time.Since(alreadyAlerted.Time) < 30*time.Minute {
		return
	}

	var authorID string
	err := s.db.QueryRow(`
		SELECT id
		FROM users
		WHERE company_id = $1 AND COALESCE(is_active, true) = true
		ORDER BY created_at ASC
		LIMIT 1
	`, companyID).Scan(&authorID)
	if err != nil || authorID == "" {
		return
	}

	title := "Instância WhatsApp desconectada"
	content := "A instância " + instanceName + " saiu do status conectado. Status atual: " + status + "."
	announcementID := uuid.New().String()
	_, _ = s.db.Exec(`
		INSERT INTO internal_announcements (id, company_id, author_id, title, content, priority, target_type, is_pinned)
		VALUES ($1, $2, $3, $4, $5, 'urgent', 'all', true)
	`, announcementID, companyID, authorID, title, content)

	metadata, _ := json.Marshal(map[string]string{
		"instance_id":   instanceID,
		"instance_name": instanceName,
		"channel_id":    channelID,
		"status":        status,
	})
	_, _ = s.db.Exec(`
		INSERT INTO audit_logs (id, company_id, user_id, action, entity_type, entity_id, metadata, created_at)
		VALUES ($1, $2, NULL, 'whatsapp.instance.down', 'whatsapp_instance', $3, $4::jsonb, NOW())
	`, uuid.New().String(), companyID, instanceID, string(metadata))

	_, _ = s.db.Exec("UPDATE whatsapp_instances SET disconnected_alerted_at = NOW() WHERE id = $1", instanceID)
	s.wsHub.BroadcastToCompany(companyID, "operational_alert", map[string]interface{}{
		"type":          "whatsapp_instance_down",
		"title":         title,
		"content":       content,
		"instance_id":   instanceID,
		"instance_name": instanceName,
		"status":        status,
	})
}
