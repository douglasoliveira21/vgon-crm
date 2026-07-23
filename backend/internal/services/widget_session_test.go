package services

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/evocrm/backend/internal/config"
)

func TestWidgetSessionIsBoundToWidgetConversationAndVisitor(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	auth := NewAuthService(db, &config.Config{})
	token := "signed-visitor-token"

	mock.ExpectQuery("UPDATE widget_sessions").
		WithArgs("widget-a", "conversation-a", "visitor-a", tokenHash(token)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("session-a"))
	if !auth.ValidateWidgetSession("widget-a", "conversation-a", "visitor-a", token) {
		t.Fatal("valid widget session was rejected")
	}

	mock.ExpectQuery("UPDATE widget_sessions").
		WithArgs("widget-b", "conversation-a", "visitor-a", tokenHash(token)).
		WillReturnError(sqlmock.ErrCancelled)
	if auth.ValidateWidgetSession("widget-b", "conversation-a", "visitor-a", token) {
		t.Fatal("widget session was accepted for another widget")
	}
	if auth.ValidateWidgetSession("", "conversation-a", "visitor-a", token) {
		t.Fatal("incomplete widget session was accepted")
	}
}
