package services

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/evocrm/backend/internal/models"
	"github.com/evocrm/backend/internal/websocket"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type MessageService struct {
	db    *sql.DB
	redis *redis.Client
	wsHub *websocket.Hub
}

func NewMessageService(db *sql.DB, rdb *redis.Client, wsHub *websocket.Hub) *MessageService {
	return &MessageService{db: db, redis: rdb, wsHub: wsHub}
}

// SendTextMessageRequest represents a text message send request
type SendTextMessageRequest struct {
	ConversationID string `json:"conversation_id"`
	Content        string `json:"content"`
	IsPrivate      bool   `json:"is_private"`
}

// GetConversationMessages returns messages for a conversation
func (s *MessageService) GetConversationMessages(conversationID, companyID string, limit, offset int) ([]models.Message, error) {
	if limit == 0 {
		limit = 50
	}

	rows, err := s.db.Query(`
		SELECT m.id, m.conversation_id, m.company_id, m.sender_type, m.sender_id,
			   m.content, m.message_type, m.media_url, m.media_mime_type, m.media_filename,
			   m.external_id, m.status, m.is_private, m.metadata, m.created_at,
			   COALESCE(u.name, c.name, 'Unknown') as sender_name,
			   COALESCE(u.avatar_url, c.avatar_url) as sender_avatar_url,
			   m.reply_to_content, m.reply_to_sender
		FROM messages m
		LEFT JOIN users u ON m.sender_type = 'user' AND m.sender_id = u.id
		LEFT JOIN contacts c ON m.sender_type = 'contact' AND m.sender_id = c.id
		WHERE m.conversation_id = $1 AND m.company_id = $2
		ORDER BY m.created_at ASC
		LIMIT $3 OFFSET $4
	`, conversationID, companyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		err := rows.Scan(
			&msg.ID, &msg.ConversationID, &msg.CompanyID, &msg.SenderType, &msg.SenderID,
			&msg.Content, &msg.MessageType, &msg.MediaURL, &msg.MediaMimeType, &msg.MediaFilename,
			&msg.ExternalID, &msg.Status, &msg.IsPrivate, &msg.Metadata, &msg.CreatedAt,
			&msg.SenderName, &msg.SenderAvatarURL, &msg.ReplyToContent, &msg.ReplyToSender,
		)
		if err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// SaveAndSendMessage saves a message and sends it via WhatsApp
func (s *MessageService) SaveAndSendMessage(companyID, userID string, req *SendTextMessageRequest) (*models.Message, error) {
	msgID := uuid.New().String()

	_, err := s.db.Exec(`
		INSERT INTO messages (id, conversation_id, company_id, sender_type, sender_id, content, message_type, is_private, status)
		VALUES ($1, $2, $3, 'user', $4, $5, 'text', $6, 'sent')
	`, msgID, req.ConversationID, companyID, userID, req.Content, req.IsPrivate)
	if err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	// Update conversation
	preview := req.Content
	if len(preview) > 100 {
		preview = preview[:100]
	}
	s.db.Exec(`
		UPDATE conversations SET last_message_at = NOW(), last_message_preview = $1, updated_at = NOW()
		WHERE id = $2
	`, preview, req.ConversationID)

	msg := &models.Message{
		ID:             msgID,
		ConversationID: req.ConversationID,
		CompanyID:      companyID,
		SenderType:     "user",
		SenderID:       &userID,
		Content:        &req.Content,
		MessageType:    "text",
		IsPrivate:      req.IsPrivate,
		Status:         "sent",
		CreatedAt:      time.Now(),
	}

	// Broadcast to WebSocket
	if !req.IsPrivate {
		s.wsHub.BroadcastToCompany(companyID, websocket.EventNewMessage, msg)
	}

	return msg, nil
}

// GetConversations returns conversations for a company
func (s *MessageService) GetConversations(companyID string, status string, assignedTo string, teamID string, channelID string, unassigned bool, limit, offset int, conversationIDs ...string) ([]models.Conversation, error) {
	if limit == 0 {
		limit = 50
	}

	query := `
		SELECT c.id, c.company_id, c.contact_id, c.channel_id, c.assigned_to, c.team_id,
			   c.status, c.priority, c.subject, c.last_message_at, c.last_message_preview,
			   c.unread_count, c.created_at, c.updated_at,
			   co.name as contact_name, co.phone as contact_phone, co.avatar_url as contact_avatar_url,
			   COALESCE(c.customer_company_id, co.customer_company_id) as customer_company_id,
			   cc.name as customer_company_name,
			   c.first_response_due_at, c.resolution_due_at, c.first_response_at, c.resolved_at,
			   u.name as assigned_to_name, t.name as team_name, ch.name as channel_name
		FROM conversations c
		LEFT JOIN contacts co ON c.contact_id = co.id
		LEFT JOIN customer_companies cc ON COALESCE(c.customer_company_id, co.customer_company_id) = cc.id
		LEFT JOIN users u ON c.assigned_to = u.id
		LEFT JOIN teams t ON c.team_id = t.id
		LEFT JOIN channels ch ON c.channel_id = ch.id
		WHERE c.company_id = $1
	`
	args := []interface{}{companyID}
	argIdx := 2

	if len(conversationIDs) > 0 && conversationIDs[0] != "" {
		query += fmt.Sprintf(" AND c.id = $%d", argIdx)
		args = append(args, conversationIDs[0])
		argIdx++
	}

	if status != "" {
		// Support comma-separated status values
		if strings.Contains(status, ",") {
			statuses := strings.Split(status, ",")
			placeholders := []string{}
			for _, s := range statuses {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argIdx))
				args = append(args, strings.TrimSpace(s))
				argIdx++
			}
			query += " AND c.status IN (" + strings.Join(placeholders, ",") + ")"
		} else {
			query += fmt.Sprintf(" AND c.status = $%d", argIdx)
			args = append(args, status)
			argIdx++
		}
	}

	if assignedTo != "" {
		query += fmt.Sprintf(" AND c.assigned_to = $%d", argIdx)
		args = append(args, assignedTo)
		argIdx++
	}

	if unassigned {
		query += " AND c.assigned_to IS NULL"
	}

	if teamID != "" {
		query += fmt.Sprintf(" AND c.team_id = $%d", argIdx)
		args = append(args, teamID)
		argIdx++
	}

	if channelID != "" {
		query += fmt.Sprintf(" AND c.channel_id = $%d", argIdx)
		args = append(args, channelID)
		argIdx++
	}

	query += " ORDER BY c.last_message_at DESC NULLS LAST"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch conversations: %w", err)
	}
	defer rows.Close()

	var conversations []models.Conversation
	for rows.Next() {
		var conv models.Conversation
		err := rows.Scan(
			&conv.ID, &conv.CompanyID, &conv.ContactID, &conv.ChannelID, &conv.AssignedTo,
			&conv.TeamID, &conv.Status, &conv.Priority, &conv.Subject, &conv.LastMessageAt,
			&conv.LastMessagePreview, &conv.UnreadCount, &conv.CreatedAt, &conv.UpdatedAt,
			&conv.ContactName, &conv.ContactPhone, &conv.ContactAvatarURL,
			&conv.CustomerCompanyID, &conv.CustomerCompanyName,
			&conv.FirstResponseDueAt, &conv.ResolutionDueAt, &conv.FirstResponseAt, &conv.ResolvedAt,
			&conv.AssignedToName, &conv.TeamName, &conv.ChannelName,
		)
		if err != nil {
			continue
		}
		conversations = append(conversations, conv)
	}

	return conversations, nil
}

func (s *MessageService) GetConversationByID(companyID, conversationID string) (*models.Conversation, error) {
	conversations, err := s.GetConversations(companyID, "", "", "", "", false, 1, 0, conversationID)
	if err != nil {
		return nil, err
	}
	if len(conversations) == 0 {
		return nil, sql.ErrNoRows
	}
	return &conversations[0], nil
}

// AssignConversation assigns a conversation to a user
func (s *MessageService) AssignConversation(conversationID, userID, companyID string) error {
	_, err := s.db.Exec(`
		UPDATE conversations SET assigned_to = $1, status = 'in_progress', updated_at = NOW()
		WHERE id = $2 AND company_id = $3
	`, userID, conversationID, companyID)
	if err != nil {
		return fmt.Errorf("failed to assign conversation: %w", err)
	}

	s.wsHub.BroadcastToCompany(companyID, websocket.EventConversationUpdate, map[string]interface{}{
		"id":          conversationID,
		"assigned_to": userID,
		"status":      "in_progress",
	})

	return nil
}

// TransferConversation transfers a conversation to another user or team
func (s *MessageService) TransferConversation(conversationID, companyID string, toUserID *string, toTeamID *string, clearTeam bool) error {
	if toUserID != nil {
		_, err := s.db.Exec(`
			UPDATE conversations SET assigned_to = $1, status = 'in_progress', updated_at = NOW()
			WHERE id = $2 AND company_id = $3
		`, *toUserID, conversationID, companyID)
		if err != nil {
			return err
		}

		s.wsHub.BroadcastToCompany(companyID, websocket.EventConversationUpdate, map[string]interface{}{
			"id":          conversationID,
			"assigned_to": *toUserID,
			"status":      "in_progress",
		})
	}

	if toTeamID != nil {
		_, err := s.db.Exec(`
			UPDATE conversations SET team_id = $1, assigned_to = NULL, updated_at = NOW()
			WHERE id = $2 AND company_id = $3
		`, *toTeamID, conversationID, companyID)
		if err != nil {
			return err
		}

		s.wsHub.BroadcastToCompany(companyID, websocket.EventConversationUpdate, map[string]interface{}{
			"id":          conversationID,
			"team_id":     *toTeamID,
			"assigned_to": nil,
		})
	}

	if clearTeam {
		_, err := s.db.Exec(`
			UPDATE conversations SET team_id = NULL, updated_at = NOW()
			WHERE id = $1 AND company_id = $2
		`, conversationID, companyID)
		if err != nil {
			return err
		}

		s.wsHub.BroadcastToCompany(companyID, websocket.EventConversationUpdate, map[string]interface{}{
			"id":      conversationID,
			"team_id": nil,
		})
	}

	return nil
}

// CloseConversation marks a conversation as resolved
func (s *MessageService) CloseConversation(conversationID, companyID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE conversations SET status = 'resolved', resolved_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND company_id = $2
	`, conversationID, companyID)
	if err != nil {
		return err
	}

	if err := clearConversationAutomationState(tx, conversationID); err != nil {
		return err
	}

	return tx.Commit()
}

// ReopenConversation reopens a resolved conversation
func (s *MessageService) ReopenConversation(conversationID, companyID string) error {
	_, err := s.db.Exec(`
		UPDATE conversations SET status = 'open', resolved_at = NULL, updated_at = NOW()
		WHERE id = $1 AND company_id = $2
	`, conversationID, companyID)
	return err
}

func clearConversationAutomationState(exec interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}, conversationID string) error {
	if _, err := exec.Exec(`
		DELETE FROM glpi_flow_states gfs
		USING conversations closed_conv, conversations state_conv
		WHERE closed_conv.id = $1
			AND state_conv.id = gfs.conversation_id
			AND state_conv.company_id = closed_conv.company_id
			AND state_conv.contact_id = closed_conv.contact_id
	`, conversationID); err != nil {
		return err
	}
	if _, err := exec.Exec(`
		UPDATE bot_executions
		SET status = 'completed',
			completed_at = COALESCE(completed_at, NOW()),
			context = COALESCE(context, '{}'::jsonb) || jsonb_build_object('closed_conversation_id', $1::text, 'closed_at', NOW()::text)
		WHERE conversation_id = $1
			AND status IN ('running', 'waiting', 'external_wait', 'paused')
	`, conversationID); err != nil {
		return err
	}
	_, err := exec.Exec(`
		UPDATE bot_executions be
		SET status = 'completed',
			completed_at = COALESCE(be.completed_at, NOW()),
			context = COALESCE(be.context, '{}'::jsonb) || jsonb_build_object('closed_contact_conversation_id', $1::text, 'closed_at', NOW()::text)
		FROM conversations closed_conv, conversations exec_conv
		WHERE closed_conv.id = $1
			AND exec_conv.id = be.conversation_id
			AND exec_conv.company_id = closed_conv.company_id
			AND exec_conv.contact_id = closed_conv.contact_id
			AND be.status IN ('running', 'waiting', 'external_wait', 'paused')
	`, conversationID)
	return err
}
