package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
)

func TestCanAccessConversationKeepsTenantAndRoleScope(t *testing.T) {
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
