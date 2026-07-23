package services

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

const widgetSessionLifetime = 30 * 24 * time.Hour

func (s *AuthService) CreateWidgetSession(widgetID, conversationID, visitorID string) (string, error) {
	token, err := randomToken(32)
	if err != nil {
		return "", err
	}
	_, err = s.db.Exec(`
		INSERT INTO widget_sessions
			(id, widget_id, conversation_id, visitor_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, uuid.New().String(), widgetID, conversationID, visitorID, tokenHash(token), time.Now().Add(widgetSessionLifetime))
	return token, err
}

func (s *AuthService) ValidateWidgetSession(widgetID, conversationID, visitorID, token string) bool {
	if widgetID == "" || conversationID == "" || visitorID == "" || token == "" {
		return false
	}
	var id string
	err := s.db.QueryRow(`
		UPDATE widget_sessions
		SET last_used_at = NOW()
		WHERE widget_id = $1 AND conversation_id = $2 AND visitor_id = $3
		  AND token_hash = $4 AND revoked_at IS NULL AND expires_at > NOW()
		RETURNING id
	`, widgetID, conversationID, visitorID, tokenHash(token)).Scan(&id)
	return err == nil
}

func (s *AuthService) ValidateWidgetSocket(conversationID, visitorID, token string) bool {
	if conversationID == "" || visitorID == "" || token == "" {
		return false
	}
	var id string
	err := s.db.QueryRow(`
		UPDATE widget_sessions
		SET last_used_at = NOW()
		WHERE conversation_id = $1 AND visitor_id = $2 AND token_hash = $3
		  AND revoked_at IS NULL AND expires_at > NOW()
		RETURNING id
	`, conversationID, visitorID, tokenHash(token)).Scan(&id)
	return err == nil
}

func (s *AuthService) HasWidgetSession(widgetID, conversationID, visitorID string) bool {
	var exists bool
	_ = s.db.QueryRow(`SELECT EXISTS (
		SELECT 1 FROM widget_sessions
		WHERE widget_id = $1 AND conversation_id = $2 AND visitor_id = $3
	)`, widgetID, conversationID, visitorID).Scan(&exists)
	return exists
}

func (s *AuthService) RevokeWidgetSession(widgetID, conversationID, visitorID string) error {
	result, err := s.db.Exec(`
		UPDATE widget_sessions SET revoked_at = NOW()
		WHERE widget_id = $1 AND conversation_id = $2 AND visitor_id = $3
		  AND revoked_at IS NULL
	`, widgetID, conversationID, visitorID)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

var ErrWidgetSession = errors.New("sessão do widget inválida ou expirada")
