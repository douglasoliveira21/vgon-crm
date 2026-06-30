package services

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/evocrm/backend/internal/config"
	ws "github.com/evocrm/backend/internal/websocket"
	"github.com/google/uuid"
)

type EmailService struct {
	db    *sql.DB
	wsHub *ws.Hub
	cfg   *config.Config
}

type EmailChannelSettings struct {
	Provider     string    `json:"provider,omitempty"`
	IMAPHost     string    `json:"imap_host"`
	IMAPPort     int       `json:"imap_port"`
	Username     string    `json:"username"`
	Password     string    `json:"password,omitempty"`
	SMTPHost     string    `json:"smtp_host,omitempty"`
	SMTPPort     int       `json:"smtp_port,omitempty"`
	SMTPUsername string    `json:"smtp_username,omitempty"`
	SMTPPassword string    `json:"smtp_password,omitempty"`
	SMTPUseTLS   bool      `json:"smtp_use_tls"`
	Mailbox      string    `json:"mailbox"`
	UseTLS       bool      `json:"use_tls"`
	LastUID      uint32    `json:"last_uid,omitempty"`
	MaxImport    int       `json:"max_import,omitempty"`
	AccessToken  string    `json:"access_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenExpiry  time.Time `json:"token_expiry,omitempty"`
	EmailAddress string    `json:"email_address,omitempty"`
}

type emailSendTarget struct {
	ChannelID    string
	ChannelName  string
	ContactEmail string
	ContactName  string
	Subject      string
	Settings     EmailChannelSettings
	RawSettings  []byte
}

type fetchedEmail struct {
	UID       uint32
	MessageID string
	FromName  string
	FromEmail string
	Subject   string
	Date      time.Time
	Body      string
}

func NewEmailService(db *sql.DB, wsHub *ws.Hub, cfg *config.Config) *EmailService {
	return &EmailService{db: db, wsHub: wsHub, cfg: cfg}
}

func (s *EmailService) StartPeriodicSync() {
	ticker := time.NewTicker(2 * time.Minute)
	go func() {
		for range ticker.C {
			s.syncActiveChannels()
		}
	}()
}

func (s *EmailService) syncActiveChannels() {
	rows, err := s.db.Query(`
		SELECT id, company_id
		FROM channels
		WHERE type = 'email' AND is_active = true AND status = 'connected'
	`)
	if err != nil {
		log.Printf("[EMAIL] failed to list email channels: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var channelID, companyID string
		if rows.Scan(&channelID, &companyID) == nil {
			if _, err := s.SyncChannel(companyID, channelID); err != nil {
				log.Printf("[EMAIL] sync failed for channel %s: %v", channelID, err)
			}
		}
	}
}

func (s *EmailService) SyncChannel(companyID, channelID string) (int, error) {
	var rawSettings []byte
	var status string
	err := s.db.QueryRow(`
		SELECT settings, status
		FROM channels
		WHERE id = $1 AND company_id = $2 AND type = 'email' AND is_active = true
	`, channelID, companyID).Scan(&rawSettings, &status)
	if err != nil {
		return 0, fmt.Errorf("email channel not found: %w", err)
	}

	settings, err := parseEmailSettings(rawSettings)
	if err != nil {
		return 0, err
	}

	var emails []fetchedEmail
	var lastUID uint32
	if settings.Provider == "gmail" || settings.Provider == "outlook" {
		emails, err = s.fetchOAuthEmails(&settings)
		lastUID = settings.LastUID
	} else {
		emails, lastUID, err = fetchInboxEmails(settings)
	}
	if err != nil {
		s.db.Exec("UPDATE channels SET status = 'error', updated_at = NOW() WHERE id = $1 AND company_id = $2", channelID, companyID)
		return 0, err
	}

	imported := 0
	for _, item := range emails {
		if item.UID > 0 && item.UID <= settings.LastUID {
			continue
		}
		ok, err := s.saveEmail(companyID, channelID, item)
		if err != nil {
			log.Printf("[EMAIL] failed to save uid %d from channel %s: %v", item.UID, channelID, err)
			continue
		}
		if ok {
			imported++
		}
	}

	if lastUID > settings.LastUID {
		settings.LastUID = lastUID
	}
	updatedSettings, _ := json.Marshal(settings)
	s.db.Exec(`
		UPDATE channels
		SET settings = $1, status = 'connected', updated_at = NOW()
		WHERE id = $2 AND company_id = $3
	`, updatedSettings, channelID, companyID)

	return imported, nil
}

func (s *EmailService) SendReply(companyID, conversationID, content string) (string, error) {
	target, err := s.getEmailSendTarget(companyID, conversationID)
	if err != nil {
		return "", err
	}
	if target.ContactEmail == "" {
		return "", fmt.Errorf("contato sem e-mail")
	}

	subject := strings.TrimSpace(target.Subject)
	if subject == "" {
		subject = "Resposta"
	}
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}

	var externalID string
	switch target.Settings.Provider {
	case "gmail":
		externalID, err = s.sendGmailReply(&target.Settings, target.ContactEmail, subject, content)
	case "outlook":
		externalID, err = s.sendOutlookReply(&target.Settings, target.ContactEmail, subject, content)
	default:
		externalID, err = sendSMTPReply(target.Settings, target.ContactEmail, subject, content)
	}
	if err != nil {
		return "", err
	}

	updatedSettings, _ := json.Marshal(target.Settings)
	_, _ = s.db.Exec("UPDATE channels SET settings = $1, updated_at = NOW() WHERE id = $2 AND company_id = $3", updatedSettings, target.ChannelID, companyID)
	return externalID, nil
}

func (s *EmailService) getEmailSendTarget(companyID, conversationID string) (emailSendTarget, error) {
	var target emailSendTarget
	err := s.db.QueryRow(`
		SELECT ch.id, ch.name, ch.settings, COALESCE(co.email, ''), COALESCE(co.name, ''), COALESCE(c.subject, '')
		FROM conversations c
		JOIN channels ch ON c.channel_id = ch.id
		JOIN contacts co ON c.contact_id = co.id
		WHERE c.id = $1 AND c.company_id = $2 AND ch.type = 'email'
	`, conversationID, companyID).Scan(&target.ChannelID, &target.ChannelName, &target.RawSettings, &target.ContactEmail, &target.ContactName, &target.Subject)
	if err != nil {
		return target, fmt.Errorf("conversa de e-mail não encontrada: %w", err)
	}
	settings, err := parseEmailSettings(target.RawSettings)
	if err != nil {
		return target, err
	}
	target.Settings = settings
	return target, nil
}

func (s *EmailService) sendGmailReply(settings *EmailChannelSettings, to, subject, body string) (string, error) {
	if err := s.ensureOAuthAccess(settings); err != nil {
		return "", err
	}
	from := settings.EmailAddress
	if from == "" {
		from = settings.Username
	}
	rawMessage := buildRawEmail(from, to, subject, body)
	payload, _ := json.Marshal(map[string]string{
		"raw": base64.RawURLEncoding.EncodeToString([]byte(rawMessage)),
	})
	req, _ := http.NewRequest("POST", "https://gmail.googleapis.com/gmail/v1/users/me/messages/send", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+settings.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("Gmail API falhou ao enviar: %s", string(respBody))
	}
	var raw struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&raw)
	if raw.ID != "" {
		return "gmail:" + raw.ID, nil
	}
	return "gmail:sent:" + uuid.New().String(), nil
}

func (s *EmailService) sendOutlookReply(settings *EmailChannelSettings, to, subject, body string) (string, error) {
	if err := s.ensureOAuthAccess(settings); err != nil {
		return "", err
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"message": map[string]interface{}{
			"subject": subject,
			"body": map[string]string{
				"contentType": "Text",
				"content":     body,
			},
			"toRecipients": []map[string]interface{}{
				{"emailAddress": map[string]string{"address": to}},
			},
		},
		"saveToSentItems": true,
	})
	req, _ := http.NewRequest("POST", "https://graph.microsoft.com/v1.0/me/sendMail", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+settings.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("Microsoft Graph falhou ao enviar: %s", string(respBody))
	}
	return "outlook:sent:" + uuid.New().String(), nil
}

func (s *EmailService) ensureOAuthAccess(settings *EmailChannelSettings) error {
	if settings.AccessToken == "" {
		return fmt.Errorf("conta OAuth sem token de acesso")
	}
	if !settings.TokenExpiry.IsZero() && time.Now().After(settings.TokenExpiry.Add(-2*time.Minute)) {
		return s.refreshOAuthToken(settings)
	}
	return nil
}

func sendSMTPReply(settings EmailChannelSettings, to, subject, body string) (string, error) {
	host := strings.TrimSpace(settings.SMTPHost)
	if host == "" {
		return "", fmt.Errorf("SMTP não configurado para este canal")
	}
	port := settings.SMTPPort
	if port == 0 {
		if settings.SMTPUseTLS {
			port = 465
		} else {
			port = 587
		}
	}
	username := settings.SMTPUsername
	if username == "" {
		username = settings.Username
	}
	password := settings.SMTPPassword
	if password == "" {
		password = settings.Password
	}
	from := settings.EmailAddress
	if from == "" {
		from = username
	}
	if username == "" || password == "" || from == "" {
		return "", fmt.Errorf("usuário, senha ou remetente SMTP ausente")
	}

	address := fmt.Sprintf("%s:%d", host, port)
	message := []byte(buildRawEmail(from, to, subject, body))
	auth := smtp.PlainAuth("", username, password, host)

	if settings.SMTPUseTLS && port == 465 {
		conn, err := tls.Dial("tcp", address, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
		if err != nil {
			return "", err
		}
		defer conn.Close()
		client, err := smtp.NewClient(conn, host)
		if err != nil {
			return "", err
		}
		defer client.Quit()
		if err := client.Auth(auth); err != nil {
			return "", err
		}
		if err := client.Mail(from); err != nil {
			return "", err
		}
		if err := client.Rcpt(to); err != nil {
			return "", err
		}
		writer, err := client.Data()
		if err != nil {
			return "", err
		}
		if _, err := writer.Write(message); err != nil {
			return "", err
		}
		if err := writer.Close(); err != nil {
			return "", err
		}
		return "smtp:sent:" + uuid.New().String(), nil
	}

	if err := smtp.SendMail(address, auth, from, []string{to}, message); err != nil {
		return "", err
	}
	return "smtp:sent:" + uuid.New().String(), nil
}

func buildRawEmail(from, to, subject, body string) string {
	encodedSubject := mime.QEncoding.Encode("utf-8", subject)
	return fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n%s", from, to, encodedSubject, body)
}

func (s *EmailService) fetchOAuthEmails(settings *EmailChannelSettings) ([]fetchedEmail, error) {
	if settings.AccessToken == "" {
		return nil, fmt.Errorf("conta OAuth sem token de acesso")
	}
	if !settings.TokenExpiry.IsZero() && time.Now().After(settings.TokenExpiry.Add(-2*time.Minute)) {
		if err := s.refreshOAuthToken(settings); err != nil {
			return nil, err
		}
	}
	if settings.Provider == "gmail" {
		return fetchGmailAPIEmails(settings)
	}
	if settings.Provider == "outlook" {
		return fetchOutlookAPIEmails(settings)
	}
	return nil, fmt.Errorf("provedor OAuth não suportado")
}

func (s *EmailService) refreshOAuthToken(settings *EmailChannelSettings) error {
	if settings.RefreshToken == "" {
		return fmt.Errorf("refresh token ausente")
	}

	values := url.Values{}
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", settings.RefreshToken)

	tokenURL := ""
	if settings.Provider == "gmail" {
		tokenURL = "https://oauth2.googleapis.com/token"
		values.Set("client_id", s.cfg.GoogleClientID)
		values.Set("client_secret", s.cfg.GoogleClientSecret)
	} else if settings.Provider == "outlook" {
		tokenURL = "https://login.microsoftonline.com/common/oauth2/v2.0/token"
		values.Set("client_id", s.cfg.MicrosoftClientID)
		values.Set("client_secret", s.cfg.MicrosoftClientSecret)
	}
	if tokenURL == "" {
		return fmt.Errorf("provedor inválido")
	}

	resp, err := http.PostForm(tokenURL, values)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("falha ao renovar token: %s", string(body))
	}

	var token struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return err
	}
	if token.AccessToken != "" {
		settings.AccessToken = token.AccessToken
	}
	if token.RefreshToken != "" {
		settings.RefreshToken = token.RefreshToken
	}
	if token.ExpiresIn > 0 {
		settings.TokenExpiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}
	return nil
}

func fetchGmailAPIEmails(settings *EmailChannelSettings) ([]fetchedEmail, error) {
	maxResults := settings.MaxImport
	if maxResults <= 0 || maxResults > 100 {
		maxResults = 50
	}
	endpoint := fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages?maxResults=%d&q=in:inbox", maxResults)
	req, _ := http.NewRequest("GET", endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+settings.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("Gmail API retornou erro: %s", string(body))
	}

	var list struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}

	var result []fetchedEmail
	for _, item := range list.Messages {
		msg, err := fetchGmailMessage(settings.AccessToken, item.ID)
		if err == nil && msg.FromEmail != "" {
			result = append(result, msg)
		}
	}
	return result, nil
}

func fetchGmailMessage(accessToken, id string) (fetchedEmail, error) {
	req, _ := http.NewRequest("GET", "https://gmail.googleapis.com/gmail/v1/users/me/messages/"+url.PathEscape(id)+"?format=full", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fetchedEmail{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fetchedEmail{}, fmt.Errorf("falha ao ler mensagem Gmail")
	}
	var raw struct {
		ID           string `json:"id"`
		Snippet      string `json:"snippet"`
		InternalDate string `json:"internalDate"`
		Payload      struct {
			Headers []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"headers"`
		} `json:"payload"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return fetchedEmail{}, err
	}
	msg := fetchedEmail{MessageID: "gmail:" + raw.ID, Body: raw.Snippet}
	for _, h := range raw.Payload.Headers {
		switch strings.ToLower(h.Name) {
		case "subject":
			msg.Subject = decodeHeader(h.Value)
		case "from":
			if addr, err := mail.ParseAddress(h.Value); err == nil {
				msg.FromName = decodeHeader(addr.Name)
				msg.FromEmail = strings.ToLower(addr.Address)
			}
		case "date":
			if parsed, err := mail.ParseDate(h.Value); err == nil {
				msg.Date = parsed
			}
		case "message-id":
			if h.Value != "" {
				msg.MessageID = strings.Trim(h.Value, "<>")
			}
		}
	}
	if msg.Date.IsZero() && raw.InternalDate != "" {
		if ms, err := strconv.ParseInt(raw.InternalDate, 10, 64); err == nil {
			msg.Date = time.UnixMilli(ms)
		}
	}
	return msg, nil
}

func fetchOutlookAPIEmails(settings *EmailChannelSettings) ([]fetchedEmail, error) {
	top := settings.MaxImport
	if top <= 0 || top > 100 {
		top = 50
	}
	endpoint := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/messages?$top=%d&$select=id,subject,from,receivedDateTime,bodyPreview", top)
	req, _ := http.NewRequest("GET", endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+settings.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("Microsoft Graph retornou erro: %s", string(body))
	}

	var raw struct {
		Value []struct {
			ID               string `json:"id"`
			Subject          string `json:"subject"`
			ReceivedDateTime string `json:"receivedDateTime"`
			BodyPreview      string `json:"bodyPreview"`
			From             struct {
				EmailAddress struct {
					Name    string `json:"name"`
					Address string `json:"address"`
				} `json:"emailAddress"`
			} `json:"from"`
		} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	var result []fetchedEmail
	for _, item := range raw.Value {
		date, _ := time.Parse(time.RFC3339, item.ReceivedDateTime)
		result = append(result, fetchedEmail{
			MessageID: "outlook:" + item.ID,
			FromName:  item.From.EmailAddress.Name,
			FromEmail: strings.ToLower(item.From.EmailAddress.Address),
			Subject:   item.Subject,
			Date:      date,
			Body:      item.BodyPreview,
		})
	}
	return result, nil
}

func (s *EmailService) saveEmail(companyID, channelID string, item fetchedEmail) (bool, error) {
	externalID := item.MessageID
	if externalID == "" {
		externalID = fmt.Sprintf("imap:%s:%d", channelID, item.UID)
	}

	var exists bool
	if err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM messages WHERE company_id = $1 AND external_id = $2)", companyID, externalID).Scan(&exists); err == nil && exists {
		return false, nil
	}

	contactID, err := s.getOrCreateEmailContact(companyID, item.FromName, item.FromEmail)
	if err != nil {
		return false, err
	}

	conversationID, isNew, isResolved, err := s.getOrCreateEmailConversation(companyID, contactID, channelID, item.Subject)
	if err != nil {
		return false, err
	}

	msgID := uuid.New().String()
	content := strings.TrimSpace(item.Body)
	if content == "" {
		content = "(email sem texto)"
	}
	metadata, _ := json.Marshal(map[string]interface{}{
		"email_uid":     item.UID,
		"email_subject": item.Subject,
		"email_from":    item.FromEmail,
	})

	createdAt := item.Date
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	_, err = s.db.Exec(`
		INSERT INTO messages (id, conversation_id, company_id, sender_type, sender_id, content, message_type, external_id, status, metadata, created_at)
		VALUES ($1, $2, $3, 'contact', $4, $5, 'email', $6, 'delivered', $7, $8)
	`, msgID, conversationID, companyID, contactID, content, externalID, metadata, createdAt)
	if err != nil {
		return false, err
	}

	preview := content
	if len(preview) > 160 {
		preview = preview[:160]
	}
	if isResolved {
		_, err = s.db.Exec(`
			UPDATE conversations
			SET last_message_at = GREATEST(COALESCE(last_message_at, $1), $1),
			    last_message_preview = $2,
			    subject = COALESCE(NULLIF(subject, ''), $3),
			    updated_at = NOW()
			WHERE id = $4 AND company_id = $5
		`, createdAt, preview, item.Subject, conversationID, companyID)
	} else {
		_, err = s.db.Exec(`
			UPDATE conversations
			SET last_message_at = $1, last_message_preview = $2, unread_count = unread_count + 1, subject = COALESCE(NULLIF(subject, ''), $3), updated_at = NOW()
			WHERE id = $4 AND company_id = $5
		`, createdAt, preview, item.Subject, conversationID, companyID)
	}
	if err != nil {
		return false, err
	}

	if isResolved {
		return true, nil
	}

	messagePayload := map[string]interface{}{
		"id":              msgID,
		"conversation_id": conversationID,
		"company_id":      companyID,
		"sender_type":     "contact",
		"sender_id":       contactID,
		"content":         content,
		"message_type":    "email",
		"external_id":     externalID,
		"status":          "delivered",
		"created_at":      createdAt,
	}
	s.wsHub.BroadcastToCompany(companyID, ws.EventNewMessage, messagePayload)
	if isNew {
		s.wsHub.BroadcastToCompany(companyID, ws.EventConversationNew, map[string]interface{}{"id": conversationID})
	}

	return true, nil
}

func (s *EmailService) getOrCreateEmailContact(companyID, name, email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		email = "sem-email-" + uuid.New().String() + "@local"
	}

	var contactID string
	err := s.db.QueryRow("SELECT id FROM contacts WHERE company_id = $1 AND lower(email) = $2 ORDER BY created_at DESC LIMIT 1", companyID, email).Scan(&contactID)
	if err == nil {
		return contactID, nil
	}

	if strings.TrimSpace(name) == "" {
		name = email
	}
	contactID = uuid.New().String()
	_, err = s.db.Exec(`
		INSERT INTO contacts (id, company_id, name, email, origin)
		VALUES ($1, $2, $3, $4, 'email')
	`, contactID, companyID, name, email)
	return contactID, err
}

func (s *EmailService) getOrCreateEmailConversation(companyID, contactID, channelID, subject string) (string, bool, bool, error) {
	var conversationID string
	err := s.db.QueryRow(`
		SELECT id
		FROM conversations
		WHERE company_id = $1 AND contact_id = $2 AND channel_id = $3 AND status != 'resolved'
		ORDER BY created_at DESC
		LIMIT 1
	`, companyID, contactID, channelID).Scan(&conversationID)
	if err == nil {
		return conversationID, false, false, nil
	}

	normalizedSubject := normalizeEmailSubject(subject)
	if normalizedSubject != "" {
		err = s.db.QueryRow(`
			SELECT id
			FROM conversations
			WHERE company_id = $1
			  AND contact_id = $2
			  AND channel_id = $3
			  AND status = 'resolved'
			  AND lower(regexp_replace(regexp_replace(COALESCE(subject, ''), '^\s*(re|fw|fwd)\s*:\s*', '', 'i'), '\s+', ' ', 'g')) = $4
			ORDER BY resolved_at DESC NULLS LAST, updated_at DESC
			LIMIT 1
		`, companyID, contactID, channelID, normalizedSubject).Scan(&conversationID)
		if err == nil {
			return conversationID, false, true, nil
		}
	}

	conversationID = uuid.New().String()
	_, err = s.db.Exec(`
		INSERT INTO conversations (id, company_id, contact_id, channel_id, status, subject, last_message_at)
		VALUES ($1, $2, $3, $4, 'open', $5, NOW())
	`, conversationID, companyID, contactID, channelID, subject)
	return conversationID, true, false, err
}

func normalizeEmailSubject(subject string) string {
	value := strings.ToLower(strings.TrimSpace(subject))
	for {
		next := regexp.MustCompile(`^\s*(re|fw|fwd)\s*:\s*`).ReplaceAllString(value, "")
		next = strings.TrimSpace(next)
		if next == value {
			break
		}
		value = next
	}
	value = regexp.MustCompile(`\s+`).ReplaceAllString(value, " ")
	return value
}

func parseEmailSettings(raw []byte) (EmailChannelSettings, error) {
	var settings EmailChannelSettings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return settings, fmt.Errorf("invalid email settings: %w", err)
	}
	if settings.Provider == "gmail" || settings.Provider == "outlook" {
		if settings.AccessToken == "" && settings.RefreshToken == "" {
			return settings, fmt.Errorf("conta OAuth sem tokens")
		}
		if settings.MaxImport <= 0 {
			settings.MaxImport = 100
		}
		return settings, nil
	}
	if settings.IMAPHost == "" || settings.Username == "" || settings.Password == "" {
		return settings, fmt.Errorf("IMAP host, user and password are required")
	}
	if settings.IMAPPort == 0 {
		if settings.UseTLS {
			settings.IMAPPort = 993
		} else {
			settings.IMAPPort = 143
		}
	}
	if settings.Mailbox == "" {
		settings.Mailbox = "INBOX"
	}
	if settings.MaxImport <= 0 {
		settings.MaxImport = 500
	}
	return settings, nil
}

func ParseEmailSettingsForHandler(raw []byte) (EmailChannelSettings, error) {
	return parseEmailSettings(raw)
}

func fetchInboxEmails(settings EmailChannelSettings) ([]fetchedEmail, uint32, error) {
	conn, reader, err := dialIMAP(settings)
	if err != nil {
		return nil, settings.LastUID, err
	}
	defer conn.Close()

	if _, err := readIMAPUntil(reader, ""); err != nil {
		return nil, settings.LastUID, err
	}

	if err := sendIMAP(conn, reader, "a001", "LOGIN %s %s", imapQuote(settings.Username), imapQuote(settings.Password)); err != nil {
		return nil, settings.LastUID, err
	}
	if err := sendIMAP(conn, reader, "a002", "SELECT %s", imapQuote(settings.Mailbox)); err != nil {
		return nil, settings.LastUID, err
	}

	searchQuery := "ALL"
	if settings.LastUID > 0 {
		searchQuery = fmt.Sprintf("UID %d:*", settings.LastUID+1)
	}
	if _, err := fmt.Fprintf(conn, "a003 UID SEARCH %s\r\n", searchQuery); err != nil {
		return nil, settings.LastUID, err
	}
	lines, err := readIMAPUntil(reader, "a003")
	if err != nil {
		return nil, settings.LastUID, err
	}

	uids := parseSearchUIDs(lines)
	if len(uids) == 0 {
		_ = sendIMAP(conn, reader, "a999", "LOGOUT")
		return nil, settings.LastUID, nil
	}
	sort.Slice(uids, func(i, j int) bool { return uids[i] < uids[j] })
	if settings.MaxImport > 0 && len(uids) > settings.MaxImport {
		uids = uids[len(uids)-settings.MaxImport:]
	}

	var emails []fetchedEmail
	lastUID := settings.LastUID
	for idx, uid := range uids {
		tag := fmt.Sprintf("a%03d", idx+4)
		if _, err := fmt.Fprintf(conn, "%s UID FETCH %d (RFC822)\r\n", tag, uid); err != nil {
			return emails, lastUID, err
		}
		_, literals, err := readIMAPResponseWithLiterals(reader, tag)
		if err != nil {
			return emails, lastUID, err
		}
		if uid > lastUID {
			lastUID = uid
		}
		if len(literals) == 0 {
			continue
		}
		msg, err := parseRawEmail(uid, literals[0])
		if err == nil && msg.FromEmail != "" {
			emails = append(emails, msg)
		}
	}

	_ = sendIMAP(conn, reader, "a999", "LOGOUT")
	return emails, lastUID, nil
}

func dialIMAP(settings EmailChannelSettings) (net.Conn, *bufio.Reader, error) {
	address := fmt.Sprintf("%s:%d", settings.IMAPHost, settings.IMAPPort)
	var conn net.Conn
	var err error
	if settings.UseTLS {
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: 20 * time.Second}, "tcp", address, &tls.Config{ServerName: settings.IMAPHost, MinVersion: tls.VersionTLS12})
	} else {
		conn, err = net.DialTimeout("tcp", address, 20*time.Second)
	}
	if err != nil {
		return nil, nil, err
	}
	return conn, bufio.NewReader(conn), nil
}

func sendIMAP(conn net.Conn, reader *bufio.Reader, tag, format string, args ...interface{}) error {
	if _, err := fmt.Fprintf(conn, "%s %s\r\n", tag, fmt.Sprintf(format, args...)); err != nil {
		return err
	}
	lines, err := readIMAPUntil(reader, tag)
	if err != nil {
		return err
	}
	last := ""
	if len(lines) > 0 {
		last = lines[len(lines)-1]
	}
	if !strings.Contains(strings.ToUpper(last), tag+" OK") {
		return fmt.Errorf("IMAP command failed: %s", last)
	}
	return nil
}

func readIMAPUntil(reader *bufio.Reader, tag string) ([]string, error) {
	lines, _, err := readIMAPResponseWithLiterals(reader, tag)
	return lines, err
}

func readIMAPResponseWithLiterals(reader *bufio.Reader, tag string) ([]string, [][]byte, error) {
	var lines []string
	var literals [][]byte
	literalPattern := regexp.MustCompile(`\{(\d+)\}$`)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return lines, literals, err
		}
		line = strings.TrimRight(line, "\r\n")
		lines = append(lines, line)

		if matches := literalPattern.FindStringSubmatch(line); len(matches) == 2 {
			size, _ := strconv.Atoi(matches[1])
			if size > 0 {
				buf := make([]byte, size)
				if _, err := io.ReadFull(reader, buf); err != nil {
					return lines, literals, err
				}
				literals = append(literals, buf)
				_, _ = reader.ReadString('\n')
			}
		}

		if tag == "" {
			if strings.HasPrefix(line, "* OK") || strings.HasPrefix(line, "* PREAUTH") {
				return lines, literals, nil
			}
		} else if strings.HasPrefix(line, tag+" ") {
			return lines, literals, nil
		}
	}
}

func parseSearchUIDs(lines []string) []uint32 {
	var uids []uint32
	for _, line := range lines {
		upper := strings.ToUpper(line)
		if !strings.HasPrefix(upper, "* SEARCH") {
			continue
		}
		parts := strings.Fields(line)
		for _, part := range parts[2:] {
			value, err := strconv.ParseUint(part, 10, 32)
			if err == nil {
				uids = append(uids, uint32(value))
			}
		}
	}
	return uids
}

func parseRawEmail(uid uint32, raw []byte) (fetchedEmail, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return fetchedEmail{}, err
	}

	var item fetchedEmail
	item.UID = uid
	item.Subject = decodeHeader(msg.Header.Get("Subject"))
	item.MessageID = strings.Trim(msg.Header.Get("Message-Id"), "<>")
	if item.MessageID == "" {
		item.MessageID = strings.Trim(msg.Header.Get("Message-ID"), "<>")
	}
	if date, err := msg.Header.Date(); err == nil {
		item.Date = date
	}
	if from, err := mail.ParseAddress(msg.Header.Get("From")); err == nil {
		item.FromName = decodeHeader(from.Name)
		item.FromEmail = strings.ToLower(from.Address)
	}

	body, _ := io.ReadAll(io.LimitReader(msg.Body, 1024*1024))
	item.Body = cleanEmailBody(string(body))
	return item, nil
}

func decodeHeader(value string) string {
	decoded, err := (&mime.WordDecoder{}).DecodeHeader(value)
	if err != nil {
		return value
	}
	return decoded
}

func cleanEmailBody(body string) string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	if strings.Contains(strings.ToLower(body), "content-type: text/html") {
		body = stripHTML(body)
	}
	body = stripHTML(body)
	lines := strings.Split(body, "\n")
	var cleaned []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, ">") {
			continue
		}
		if strings.HasPrefix(strings.ToLower(trimmed), "em ") && strings.Contains(strings.ToLower(trimmed), "escreveu") {
			break
		}
		cleaned = append(cleaned, line)
		if len(strings.Join(cleaned, "\n")) > 20000 {
			break
		}
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func stripHTML(value string) string {
	re := regexp.MustCompile(`(?s)<[^>]*>`)
	value = re.ReplaceAllString(value, " ")
	value = strings.ReplaceAll(value, "&nbsp;", " ")
	value = strings.ReplaceAll(value, "&amp;", "&")
	value = strings.ReplaceAll(value, "&lt;", "<")
	value = strings.ReplaceAll(value, "&gt;", ">")
	space := regexp.MustCompile(`[ \t]+`)
	return strings.TrimSpace(space.ReplaceAllString(value, " "))
}

func imapQuote(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}
