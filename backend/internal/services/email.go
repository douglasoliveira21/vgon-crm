package services

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/mail"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	ws "github.com/evocrm/backend/internal/websocket"
	"github.com/google/uuid"
)

type EmailService struct {
	db    *sql.DB
	wsHub *ws.Hub
}

type EmailChannelSettings struct {
	Provider  string `json:"provider,omitempty"`
	IMAPHost  string `json:"imap_host"`
	IMAPPort  int    `json:"imap_port"`
	Username  string `json:"username"`
	Password  string `json:"password,omitempty"`
	Mailbox   string `json:"mailbox"`
	UseTLS    bool   `json:"use_tls"`
	LastUID   uint32 `json:"last_uid,omitempty"`
	MaxImport int    `json:"max_import,omitempty"`
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

func NewEmailService(db *sql.DB, wsHub *ws.Hub) *EmailService {
	return &EmailService{db: db, wsHub: wsHub}
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

	emails, lastUID, err := fetchInboxEmails(settings)
	if err != nil {
		s.db.Exec("UPDATE channels SET status = 'error', updated_at = NOW() WHERE id = $1 AND company_id = $2", channelID, companyID)
		return 0, err
	}

	imported := 0
	for _, item := range emails {
		if item.UID <= settings.LastUID {
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

	conversationID, isNew, err := s.getOrCreateEmailConversation(companyID, contactID, channelID, item.Subject)
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
	_, err = s.db.Exec(`
		UPDATE conversations
		SET last_message_at = $1, last_message_preview = $2, unread_count = unread_count + 1, subject = COALESCE(NULLIF(subject, ''), $3), updated_at = NOW()
		WHERE id = $4 AND company_id = $5
	`, createdAt, preview, item.Subject, conversationID, companyID)
	if err != nil {
		return false, err
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

func (s *EmailService) getOrCreateEmailConversation(companyID, contactID, channelID, subject string) (string, bool, error) {
	var conversationID string
	err := s.db.QueryRow(`
		SELECT id
		FROM conversations
		WHERE company_id = $1 AND contact_id = $2 AND channel_id = $3 AND status != 'resolved'
		ORDER BY created_at DESC
		LIMIT 1
	`, companyID, contactID, channelID).Scan(&conversationID)
	if err == nil {
		return conversationID, false, nil
	}

	conversationID = uuid.New().String()
	_, err = s.db.Exec(`
		INSERT INTO conversations (id, company_id, contact_id, channel_id, status, subject, last_message_at)
		VALUES ($1, $2, $3, $4, 'open', $5, NOW())
	`, conversationID, companyID, contactID, channelID, subject)
	return conversationID, true, err
}

func parseEmailSettings(raw []byte) (EmailChannelSettings, error) {
	var settings EmailChannelSettings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return settings, fmt.Errorf("invalid email settings: %w", err)
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
