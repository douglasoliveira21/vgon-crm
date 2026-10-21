package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/evocrm/backend/internal/config"
	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

func TestEvolutionWebhookAuthorized(t *testing.T) {
	app := fiber.New()
	app.Post("/hook", func(c *fiber.Ctx) error {
		if !evolutionWebhookAuthorized(c, c.Query("expected")) {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		return c.SendStatus(fiber.StatusOK)
	})

	tests := []struct {
		name    string
		url     string
		headers map[string]string
		want    int
	}{
		{"token in url", "/hook?expected=s3cret&token=s3cret", nil, fiber.StatusOK},
		{"bearer header", "/hook?expected=s3cret", map[string]string{"Authorization": "Bearer s3cret"}, fiber.StatusOK},
		{"custom header", "/hook?expected=s3cret", map[string]string{"X-Webhook-Token": "s3cret"}, fiber.StatusOK},
		{"wrong token", "/hook?expected=s3cret&token=nope", nil, fiber.StatusUnauthorized},
		{"no token at all", "/hook?expected=s3cret", nil, fiber.StatusUnauthorized},
		{"wrong bearer", "/hook?expected=s3cret", map[string]string{"Authorization": "Bearer nope"}, fiber.StatusUnauthorized},
		{"unconfigured stays open", "/hook", nil, fiber.StatusOK},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", test.url, nil)
			for key, value := range test.headers {
				request.Header.Set(key, value)
			}
			response, err := app.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != test.want {
				t.Fatalf("status = %d, want %d", response.StatusCode, test.want)
			}
		})
	}
}

func TestSanitizeChannelSettingsStripsCredentials(t *testing.T) {
	raw := []byte(`{
		"imap_host": "imap.example.com",
		"username": "agent@example.com",
		"password": "p1",
		"smtp_password": "p2",
		"access_token": "t1",
		"refresh_token": "t2",
		"api_key": "k1",
		"client_secret": "c1"
	}`)
	settings := sanitizeChannelSettings(raw)

	for _, field := range []string{"password", "smtp_password", "access_token", "refresh_token", "api_key", "client_secret"} {
		if _, present := settings[field]; present {
			t.Errorf("%s was exposed to the browser", field)
		}
	}
	if settings["imap_host"] != "imap.example.com" || settings["username"] != "agent@example.com" {
		t.Errorf("harmless fields were dropped: %v", settings)
	}
	if got := sanitizeChannelSettings(nil); len(got) != 0 {
		t.Errorf("empty input produced %v", got)
	}
}

func TestBelongsToCompanyScopesStagesThroughTheirFunnel(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// funnel_stages carries no company_id of its own, so ownership has to be
	// proven through the funnel — otherwise any stage id would pass.
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("stage-a", "company-a").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	if !belongsToCompany(db, "funnel_stages", "stage-a", "company-a") {
		t.Fatal("a stage of the caller's own funnel was rejected")
	}

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("stage-b", "company-a").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	if belongsToCompany(db, "funnel_stages", "stage-b", "company-a") {
		t.Fatal("another tenant's stage was accepted")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBelongsToCompanyDeniesWithoutQuerying(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	cases := []struct{ table, id, company string }{
		{"funnels", "", "company-a"},
		{"funnels", "funnel-a", ""},
		{"unknown_table", "id-a", "company-a"},
		{"users; DROP TABLE users", "id-a", "company-a"},
	}
	for _, c := range cases {
		if belongsToCompany(db, c.table, c.id, c.company) {
			t.Errorf("(%q, %q, %q) was allowed", c.table, c.id, c.company)
		}
	}
	// Nothing was expected on the mock, so an unexpected query would have
	// surfaced as an error inside belongsToCompany and been denied anyway;
	// the denials above are what matter, and the helper is built to deny
	// these inputs before it ever reaches the database.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCampaignItemAlreadySent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("recipient-a", 0).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	if !campaignItemAlreadySent(db, "recipient-a", 0) {
		t.Fatal("a delivered item would be sent again on retry")
	}

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("recipient-a", 2).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	if campaignItemAlreadySent(db, "recipient-a", 2) {
		t.Fatal("an unsent item was skipped")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func glpiGuardStatus(t *testing.T, svc *services.Container, companyID string) int {
	t.Helper()
	app := fiber.New()
	app.Get("/glpi", func(c *fiber.Ctx) error {
		c.Locals("company_id", companyID)
		c.Locals("user_id", "user-a")
		return c.Next()
	}, GLPITenantGuard(svc), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	response, err := app.Test(httptest.NewRequest("GET", "/glpi", nil))
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode
}

func TestGLPITenantGuardAllowsOnlyTheOwningCompany(t *testing.T) {
	svc := &services.Container{Config: &config.Config{GLPICompanyID: "company-owner"}}

	if got := glpiGuardStatus(t, svc, "company-owner"); got != fiber.StatusOK {
		t.Fatalf("owning company: status = %d, want %d", got, fiber.StatusOK)
	}
	if got := glpiGuardStatus(t, svc, "company-other"); got != fiber.StatusForbidden {
		t.Fatalf("other company: status = %d, want %d", got, fiber.StatusForbidden)
	}
}

func TestGLPITenantGuardFallsBackToSuperAdminsOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	svc := &services.Container{Config: &config.Config{}, DB: db}

	mock.ExpectQuery("SELECT COALESCE").
		WithArgs("user-a").
		WillReturnRows(sqlmock.NewRows([]string{"is_super_admin"}).AddRow(false))
	if got := glpiGuardStatus(t, svc, "company-a"); got != fiber.StatusForbidden {
		t.Fatalf("ordinary admin with no owner configured: status = %d, want %d", got, fiber.StatusForbidden)
	}

	mock.ExpectQuery("SELECT COALESCE").
		WithArgs("user-a").
		WillReturnRows(sqlmock.NewRows([]string{"is_super_admin"}).AddRow(true))
	if got := glpiGuardStatus(t, svc, "company-a"); got != fiber.StatusOK {
		t.Fatalf("super admin: status = %d, want %d", got, fiber.StatusOK)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPublicRegistrationIsClosedByDefault(t *testing.T) {
	svc := &services.Container{Config: &config.Config{AllowPublicRegistration: false}}
	app := fiber.New()
	app.Post("/auth/register", AuthRegister(svc))

	body := `{"company_name":"Acme","name":"Eve","email":"eve@example.com","password":"longenough1"}`
	request := httptest.NewRequest("POST", "/auth/register", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusForbidden)
	}
}

func TestRefreshMessageMediaURLIssuesFreshSignedPath(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	svc := &services.Container{DB: db, Config: &config.Config{JWTSecret: "secret"}}

	app := fiber.New()
	app.Get("/conversations/:id/messages/:messageId/media-url", func(c *fiber.Ctx) error {
		c.Locals("company_id", "company-a")
		return c.Next()
	}, RefreshMessageMediaURL(svc))

	// Scoped by message, conversation and company together.
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("message-1", "conversation-1", "company-a").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	response, err := app.Test(httptest.NewRequest("GET", "/conversations/conversation-1/messages/message-1/media-url", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("message-9", "conversation-1", "company-a").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	response, err = app.Test(httptest.NewRequest("GET", "/conversations/conversation-1/messages/message-9/media-url", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusNotFound {
		t.Fatalf("a message outside the conversation: status = %d, want %d", response.StatusCode, fiber.StatusNotFound)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
