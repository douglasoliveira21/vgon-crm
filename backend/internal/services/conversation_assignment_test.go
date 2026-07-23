package services

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/evocrm/backend/internal/websocket"
)

func expectAutomationPause(mock sqlmock.Sqlmock, conversationID string) {
	mock.ExpectExec("DELETE FROM glpi_flow_states").
		WithArgs(conversationID).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("UPDATE bot_executions").
		WithArgs(conversationID).
		WillReturnResult(sqlmock.NewResult(0, 0))
}

func TestTransferConversationToTeamPreservesAssignedAgent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := &MessageService{db: db, wsHub: websocket.NewHub()}
	teamID := "team-a"
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE conversations SET team_id = $1, updated_at = NOW()")).
		WithArgs(teamID, "conversation-a", "tenant-a").
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectAutomationPause(mock, "conversation-a")
	mock.ExpectCommit()

	if err := service.TransferConversation("conversation-a", "tenant-a", nil, &teamID, false); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTransferConversationToAgentPreservesAssignedTeam(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := &MessageService{db: db, wsHub: websocket.NewHub()}
	userID := "user-a"
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE conversations SET assigned_to = $1, status = 'in_progress', updated_at = NOW()")).
		WithArgs(userID, "conversation-a", "tenant-a").
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectAutomationPause(mock, "conversation-a")
	mock.ExpectCommit()

	if err := service.TransferConversation("conversation-a", "tenant-a", &userID, nil, false); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBotTeamTransferPreservesAssignedAgent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	engine := &BotEngine{db: db}
	node := BotNode{Data: map[string]interface{}{"team_id": "team-a"}}
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("team-a", "tenant-a").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE conversations SET team_id = $1, updated_at = NOW()")).
		WithArgs("team-a", "conversation-a", "tenant-a").
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectAutomationPause(mock, "conversation-a")

	if err := engine.nodeTransferTeam(node, "tenant-a", "conversation-a"); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
