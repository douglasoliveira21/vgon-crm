package services

import (
	"database/sql"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestExtractMessageContent(t *testing.T) {
	tests := []struct {
		name        string
		message     map[string]interface{}
		wantType    string
		wantContent string
		wantMedia   bool
	}{
		{
			name:        "plain conversation",
			message:     map[string]interface{}{"conversation": "Trocou o monitor"},
			wantType:    "text",
			wantContent: "Trocou o monitor",
		},
		{
			name: "audio has no caption but is media",
			message: map[string]interface{}{
				"audioMessage": map[string]interface{}{"url": "https://cdn/a.enc"},
			},
			wantType:  "audio",
			wantMedia: true,
		},
		{
			name: "view once photo is unwrapped",
			message: map[string]interface{}{
				"viewOnceMessageV2": map[string]interface{}{
					"message": map[string]interface{}{
						"imageMessage": map[string]interface{}{"url": "https://cdn/v.enc"},
					},
				},
			},
			wantType:  "image",
			wantMedia: true,
		},
		{
			name: "ephemeral wrapping view once audio",
			message: map[string]interface{}{
				"ephemeralMessage": map[string]interface{}{
					"message": map[string]interface{}{
						"viewOnceMessage": map[string]interface{}{
							"message": map[string]interface{}{
								"audioMessage": map[string]interface{}{"url": "https://cdn/z.enc"},
							},
						},
					},
				},
			},
			wantType:  "audio",
			wantMedia: true,
		},
		{
			name: "ephemeral text",
			message: map[string]interface{}{
				"ephemeralMessage": map[string]interface{}{
					"message": map[string]interface{}{"conversation": "some sozinha"},
				},
			},
			wantType:    "text",
			wantContent: "some sozinha",
		},
		{
			name: "document with caption wrapper",
			message: map[string]interface{}{
				"documentWithCaptionMessage": map[string]interface{}{
					"message": map[string]interface{}{
						"documentMessage": map[string]interface{}{"fileName": "nota.pdf", "url": "https://cdn/d.enc"},
					},
				},
			},
			wantType:    "document",
			wantContent: "nota.pdf",
			wantMedia:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			msgType, content, mediaURL := extractMessageContent(test.message)
			if msgType != test.wantType {
				t.Errorf("type = %q, want %q", msgType, test.wantType)
			}
			if content != test.wantContent {
				t.Errorf("content = %q, want %q", content, test.wantContent)
			}
			if (mediaURL != "") != test.wantMedia {
				t.Errorf("media present = %v, want %v", mediaURL != "", test.wantMedia)
			}
		})
	}
}

// Events that are not messages must come back empty so the webhook can drop
// them instead of storing a blank bubble.
func TestExtractMessageContentLeavesNonMessageEventsEmpty(t *testing.T) {
	events := map[string]map[string]interface{}{
		"reaction":       {"reactionMessage": map[string]interface{}{"text": "👍"}},
		"revoke":         {"protocolMessage": map[string]interface{}{"type": "REVOKE"}},
		"sender key":     {"senderKeyDistributionMessage": map[string]interface{}{"groupId": "x"}},
		"poll update":    {"pollUpdateMessage": map[string]interface{}{}},
		"empty envelope": {"ephemeralMessage": map[string]interface{}{"message": map[string]interface{}{}}},
		"empty":          {},
	}
	for name, event := range events {
		msgType, content, mediaURL := extractMessageContent(event)
		if msgType != "text" || content != "" || mediaURL != "" {
			t.Errorf("%s: got (%q, %q, %q), want an empty text message", name, msgType, content, mediaURL)
		}
	}
	if msgType, content, mediaURL := extractMessageContent(nil); msgType != "text" || content != "" || mediaURL != "" {
		t.Errorf("nil message: got (%q, %q, %q)", msgType, content, mediaURL)
	}
}

func TestMediaPreviewLabelOnlyLabelsRealMedia(t *testing.T) {
	for _, mediaType := range []string{"image", "video", "audio", "document", "sticker", "gif"} {
		if mediaPreviewLabel(mediaType) == "" {
			t.Errorf("%s has no preview label", mediaType)
		}
	}
	// A generic "attachment" label here once made contentless non-media
	// events look like a file the contact had sent.
	if label := mediaPreviewLabel("text"); label != "" {
		t.Errorf("text got preview label %q, want none", label)
	}
}

func TestBrazilianPhoneNinthDigitVariant(t *testing.T) {
	tests := map[string]string{
		"5531987654321": "553187654321",
		"553187654321":  "5531987654321",
		"5511912345678": "551112345678",
		"551144445555":  "5511944445555",
		"14155550123":   "",
		"":              "",
	}
	for input, want := range tests {
		if got := BrazilianPhoneNinthDigitVariant(input); got != want {
			t.Errorf("variant(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeEvolutionPhone(t *testing.T) {
	tests := map[string]string{
		"5531987654321@s.whatsapp.net": "5531987654321",
		"(31) 98765-4321":              "5531987654321",
		" 31 8765-4321 ":               "553187654321",
		"5531987654321":                "5531987654321",
		"120363@g.us":                  "120363@g.us",
		"":                             "",
	}
	for input, want := range tests {
		if got := NormalizeEvolutionPhone(input); got != want {
			t.Errorf("normalize(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFindContactIDByPhoneChecksBothNinthDigitForms(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Both forms go into a single lookup, ordered by age, so every caller
	// settles on the same contact when duplicates already exist.
	mock.ExpectQuery("SELECT id FROM contacts").
		WithArgs("company-a", "5531987654321", "553187654321").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("oldest-contact"))

	id, found := FindContactIDByPhone(db, "company-a", "(31) 98765-4321")
	if !found || id != "oldest-contact" {
		t.Fatalf("got (%q, %v), want (oldest-contact, true)", id, found)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFindContactIDByPhoneReportsMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT id FROM contacts").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	if id, found := FindContactIDByPhone(db, "company-a", "5531987654321"); found || id != "" {
		t.Fatalf("got (%q, %v), want a miss", id, found)
	}

	// An empty number must not even reach the database.
	if _, found := FindContactIDByPhone(db, "company-a", ""); found {
		t.Fatal("an empty phone matched a contact")
	}
}

func TestGetOrCreateContactUsesWinnerOfInsertRace(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := &EvolutionService{db: db}

	// Two webhooks for the same brand new contact: both miss the lookup, one
	// insert is rejected by the unique index. The loser must take the
	// winner's row instead of returning "" and dropping the message.
	mock.ExpectQuery("SELECT id FROM contacts").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec("INSERT INTO contacts").
		WillReturnError(errors.New("duplicate key value violates unique constraint"))
	mock.ExpectQuery("SELECT id FROM contacts").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("winner-contact"))

	got := service.getOrCreateContact("company-a", "5531987654321", map[string]interface{}{"pushName": "Ana"}, "")
	if got != "winner-contact" {
		t.Fatalf("contact = %q, want winner-contact", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetOrCreateConversationDoesNotDuplicateOnLookupError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := &EvolutionService{db: db}
	channel := "channel-a"

	// Only "no rows" means there is no conversation yet. Any other failure
	// used to start a second chat for a contact who already had one.
	mock.ExpectQuery("SELECT id FROM conversations").
		WillReturnError(errors.New("connection reset"))
	if got := service.getOrCreateConversation("company-a", "contact-a", &channel); got != "" {
		t.Fatalf("conversation = %q after a lookup error, want none", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("an insert was attempted after a lookup error: %v", err)
	}
}

func TestGetOrCreateConversationCreatesWhenNoneExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := &EvolutionService{db: db}
	channel := "channel-a"

	mock.ExpectQuery("SELECT id FROM conversations").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("INSERT INTO conversations").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if got := service.getOrCreateConversation("company-a", "contact-a", &channel); got == "" {
		t.Fatal("no conversation was created when none existed")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestIsMessageAlreadyStoredIsScopedByCompany(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := &EvolutionService{db: db}

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("company-a", "wa-123").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	if !service.isMessageAlreadyStored("company-a", "wa-123") {
		t.Fatal("a redelivered message was not recognised")
	}

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("company-b", "wa-123").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	if service.isMessageAlreadyStored("company-b", "wa-123") {
		t.Fatal("another tenant's message id masked a new message")
	}

	// A blank id can't be deduplicated and must not query.
	if service.isMessageAlreadyStored("company-a", "  ") {
		t.Fatal("a blank id was treated as stored")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestIsInternalIP(t *testing.T) {
	internal := []string{"127.0.0.1", "10.0.0.5", "172.16.0.9", "192.168.1.10", "169.254.169.254", "0.0.0.0", "::1", "fe80::1", "fc00::1"}
	for _, address := range internal {
		if !isInternalIP(net.ParseIP(address)) {
			t.Errorf("%s was not treated as internal", address)
		}
	}
	public := []string{"8.8.8.8", "1.1.1.1", "142.250.80.46", "2606:4700:4700::1111"}
	for _, address := range public {
		if isInternalIP(net.ParseIP(address)) {
			t.Errorf("%s was treated as internal", address)
		}
	}
}

func TestValidatePublicURL(t *testing.T) {
	rejected := []string{
		"http://127.0.0.1/admin",
		"http://169.254.169.254/latest/meta-data",
		"http://10.0.0.1:8080/",
		"http://[::1]/",
		"file:///etc/passwd",
		"ftp://example.com/file",
		"gopher://example.com",
		"http:///nohost",
		"not a url",
		"",
	}
	for _, raw := range rejected {
		if err := ValidatePublicURL(raw); err == nil {
			t.Errorf("%q was accepted", raw)
		}
	}
	for _, raw := range []string{"https://example.com/hook", "http://example.com:8080/x?y=1"} {
		if err := ValidatePublicURL(raw); err != nil {
			t.Errorf("%q was rejected: %v", raw, err)
		}
	}
}

func TestEmailSettingsAreEncryptedAtRest(t *testing.T) {
	const seed = "test-seed"
	settings := EmailChannelSettings{
		Provider:     "imap",
		IMAPHost:     "imap.example.com",
		Username:     "agent@example.com",
		Password:     "imap-password-123",
		SMTPPassword: "smtp-password-456",
		AccessToken:  "oauth-access-789",
		RefreshToken: "oauth-refresh-000",
	}

	raw, err := MarshalEmailSettings(settings, seed)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{settings.Password, settings.SMTPPassword, settings.AccessToken, settings.RefreshToken} {
		if strings.Contains(string(raw), secret) {
			t.Errorf("stored settings leak %q", secret)
		}
	}

	parsed, err := parseEmailSettings(raw, seed)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Password != settings.Password || parsed.SMTPPassword != settings.SMTPPassword ||
		parsed.AccessToken != settings.AccessToken || parsed.RefreshToken != settings.RefreshToken {
		t.Fatalf("round trip lost a credential: %+v", parsed)
	}
}

func TestParseEmailSettingsStillReadsPlaintextRows(t *testing.T) {
	// Rows written before encryption was introduced must keep working until
	// they are next saved.
	legacy := []byte(`{"provider":"imap","imap_host":"imap.example.com","username":"agent@example.com","password":"plain-secret"}`)
	parsed, err := parseEmailSettings(legacy, "test-seed")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Password != "plain-secret" {
		t.Fatalf("password = %q, want the legacy plaintext value", parsed.Password)
	}
}

func TestEncryptingEmailSettingsTwiceDoesNotDoubleEncrypt(t *testing.T) {
	settings := EmailChannelSettings{Provider: "imap", IMAPHost: "h", Username: "u", Password: "secret"}
	first, err := MarshalEmailSettings(settings, "seed")
	if err != nil {
		t.Fatal(err)
	}
	reread, err := parseEmailSettings(first, "seed")
	if err != nil {
		t.Fatal(err)
	}
	second, err := MarshalEmailSettings(reread, "seed")
	if err != nil {
		t.Fatal(err)
	}
	final, err := parseEmailSettings(second, "seed")
	if err != nil {
		t.Fatal(err)
	}
	if final.Password != "secret" {
		t.Fatalf("password after two save cycles = %q, want secret", final.Password)
	}
}

func TestStripMessageSignature(t *testing.T) {
	tests := map[string]string{
		"*Juliana Bernardes - (Gestão de Processo):*\n1": "1",
		"*Darlene Simões - (Diretoria):*\r\n2":           "2",
		"*Ana:*\nquero falar com o financeiro":           "quero falar com o financeiro",
		"1":                                              "1",
		"*1*":                                            "*1*",
		"*Só a assinatura:*\n":                           "*Só a assinatura:*\n",
		"bom dia\n1":                                     "bom dia\n1",
		"*Negrito comum*\n1":                             "*Negrito comum*\n1",
	}
	for input, want := range tests {
		if got := stripMessageSignature(input); got != want {
			t.Errorf("stripMessageSignature(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestChooseNextNodeIgnoresSenderSignature(t *testing.T) {
	outgoing := []BotEdge{{ID: "a", Target: "suporte"}, {ID: "b", Target: "comercial"}, {ID: "c", Target: "financeiro"}}
	for response, want := range map[string]string{
		"*Juliana Bernardes - (Gestão de Processo):*\n1": "suporte",
		"*Juliana Bernardes - (Gestão de Processo):*\n3": "financeiro",
		"*2*": "comercial",
		"2":   "comercial",
	} {
		if got := chooseNextNodeIDForResponse(outgoing, response); got != want {
			t.Errorf("response %q chose %q, want %q", response, got, want)
		}
	}
}

func TestPublishConversationStateIncludesNames(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// The contact panel shows names, so the broadcast must carry them and
	// not just the ids the old events sent.
	mock.ExpectQuery("SELECT COALESCE").
		WithArgs("conversation-a", "company-a").
		WillReturnRows(sqlmock.NewRows([]string{"status", "assigned_to", "name", "team_id", "name"}).
			AddRow("in_progress", "user-1", "Maria", "team-1", "Suporte"))

	// A nil hub must be tolerated (bot tests build the engine without one),
	// and must not even touch the database.
	publishConversationState(db, nil, "company-a", "conversation-a")
	if err := mock.ExpectationsWereMet(); err == nil {
		t.Fatal("a nil hub still queried the database")
	}
}
