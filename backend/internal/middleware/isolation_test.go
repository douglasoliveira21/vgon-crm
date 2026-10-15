package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
)

// The access decision is cached process-wide, so each test starts clean or it
// would be answered by whatever an earlier test stored under the same key.
func resetConversationAccessCache() {
	conversationAccessMu.Lock()
	defer conversationAccessMu.Unlock()
	conversationAccessCache = map[string]conversationAccessEntry{}
}

func TestCanAccessConversationKeepsTenantAndRoleScope(t *testing.T) {
	resetConversationAccessCache()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("conversation-a", "tenant-a", "agent-a").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	if CanAccessConversation(db, "conversation-a", "tenant-a", "agent-a", "agent") {
		t.Fatal("agent gained access to a conversation outside its assignment scope")
	}

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("conversation-b", "tenant-a", "supervisor-a").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	if !CanAccessConversation(db, "conversation-b", "tenant-a", "supervisor-a", "supervisor") {
		t.Fatal("supervisor was denied a conversation in the supervised team")
	}

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("conversation-c", "tenant-b").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	if CanAccessConversation(db, "conversation-c", "tenant-b", "admin-a", "admin") {
		t.Fatal("administrator gained cross-tenant conversation access")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestConversationAccessReturnsForbiddenWithoutScope(t *testing.T) {
	resetConversationAccessCache()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("conversation-a", "tenant-a", "agent-a").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	app := fiber.New()
	app.Get("/conversations/:id", func(c *fiber.Ctx) error {
		c.Locals("role_slug", "agent")
		c.Locals("company_id", "tenant-a")
		c.Locals("user_id", "agent-a")
		return c.Next()
	}, ConversationAccess(db), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	response, err := app.Test(httptest.NewRequest("GET", "/conversations/conversation-a", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusForbidden)
	}
}

func TestCanAccessConversationReusesDecisionWithinTTL(t *testing.T) {
	resetConversationAccessCache()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// A single query is expected: the later calls must come from the cache,
	// which is what keeps a broadcast to many agents from querying per client.
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("conversation-a", "tenant-a", "agent-a").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	for attempt := 0; attempt < 3; attempt++ {
		if !CanAccessConversation(db, "conversation-a", "tenant-a", "agent-a", "agent") {
			t.Fatalf("attempt %d: access was denied", attempt)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCanAccessConversationDoesNotCacheDatabaseErrors(t *testing.T) {
	resetConversationAccessCache()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// A transient failure must not turn into a cached denial for the TTL.
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("conversation-a", "tenant-a").
		WillReturnError(sqlmock.ErrCancelled)
	if CanAccessConversation(db, "conversation-a", "tenant-a", "admin-a", "admin") {
		t.Fatal("access was granted despite a database error")
	}

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("conversation-a", "tenant-a").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	if !CanAccessConversation(db, "conversation-a", "tenant-a", "admin-a", "admin") {
		t.Fatal("a cached error kept denying access after the database recovered")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
