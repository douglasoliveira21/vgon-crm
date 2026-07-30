package services

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestJobQueueEnqueueUsesStableIdempotencyKey(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	queue := NewJobQueue(db)
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO durable_jobs")).
		WithArgs(sqlmock.AnyArg(), "tenant-a", "campaign.send", "campaign-a", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("existing-job"))

	id, err := queue.Enqueue("tenant-a", "campaign.send", "campaign-a", map[string]string{"campaign_id": "campaign-a"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if id != "existing-job" {
		t.Fatalf("job id = %q, want existing-job", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRetryDeadLetterRequeuesOriginalJobAtomically(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	queue := NewJobQueue(db)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE durable_jobs").
		WithArgs("dead-a", "tenant-a").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM dead_letter_jobs").
		WithArgs("dead-a", "tenant-a").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := queue.RetryDeadLetter("dead-a", "tenant-a", false); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCampaignJobHasNoFixedExecutionDeadline(t *testing.T) {
	ctx, cancel := jobExecutionContext(context.Background(), "campaign.send")
	defer cancel()

	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		t.Fatal("campaign.send should run until completion, pause, or server shutdown")
	}
}

func TestRegularJobKeepsExecutionDeadline(t *testing.T) {
	ctx, cancel := jobExecutionContext(context.Background(), "campaign.email.send")
	defer cancel()

	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		t.Fatal("regular jobs should keep a bounded execution time")
	}
	remaining := time.Until(deadline)
	if remaining < 14*time.Minute || remaining > 15*time.Minute {
		t.Fatalf("unexpected regular job timeout: %s", remaining)
	}
}
