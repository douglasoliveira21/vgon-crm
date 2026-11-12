package services

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestShouldExecuteFlowNode(t *testing.T) {
	tests := []struct {
		name                string
		alreadyCompleted    bool
		resumedFromResponse bool
		step                int
		want                bool
	}{
		{
			name: "new node always executes",
			step: 1,
			want: true,
		},
		{
			name:             "durable retry does not duplicate completed node",
			alreadyCompleted: true,
			step:             1,
			want:             false,
		},
		{
			name:                "resume start does not duplicate completed node",
			alreadyCompleted:    true,
			resumedFromResponse: true,
			step:                1,
			want:                false,
		},
		{
			name:                "loop reached after response executes prompt again",
			alreadyCompleted:    true,
			resumedFromResponse: true,
			step:                4,
			want:                true,
		},
		{
			name:             "ordinary retry remains idempotent after first step",
			alreadyCompleted: true,
			step:             4,
			want:             false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := shouldExecuteFlowNode(test.alreadyCompleted, test.resumedFromResponse, test.step)
			if got != test.want {
				t.Fatalf("shouldExecuteFlowNode(%v, %v, %d) = %v, want %v",
					test.alreadyCompleted, test.resumedFromResponse, test.step, got, test.want)
			}
		})
	}
}

func TestClientInactivityRequiresPublicAttendantReplyForHumanOwnedConversations(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Assigned/team conversations become eligible only when the last public,
	// successful message came from an attendant, rather than a bot queue notice.
	mock.ExpectQuery(regexp.QuoteMeta("(c.assigned_to IS NULL AND c.team_id IS NULL)\n\t\t\t\tOR last_message.sender_type = 'user'")).
		WithArgs("company-a", 30, "", "flow-a").
		WillReturnRows(sqlmock.NewRows([]string{"id", "contact_id", "channel_id", "phone", "instance_name"}))

	engine := NewBotEngine(db, nil, nil)
	conversations, err := engine.findClientInactiveConversations("company-a", "flow-a", "", 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(conversations) != 0 {
		t.Fatalf("got %d inactivity candidates, want none", len(conversations))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
