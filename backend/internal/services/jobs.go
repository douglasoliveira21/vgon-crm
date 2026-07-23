package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

type JobHandler func(context.Context, json.RawMessage) error

type JobQueue struct {
	db       *sql.DB
	workerID string
	handlers map[string]JobHandler
	mu       sync.RWMutex
	started  time.Time
	lastPoll time.Time
	lastErr  string
}

func NewJobQueue(db *sql.DB) *JobQueue {
	host, _ := os.Hostname()
	return &JobQueue{
		db: db, workerID: fmt.Sprintf("%s-%s", host, uuid.New().String()[:8]),
		handlers: make(map[string]JobHandler),
	}
}

func (q *JobQueue) Register(jobType string, handler JobHandler) {
	q.mu.Lock()
	q.handlers[jobType] = handler
	q.mu.Unlock()
}

func (q *JobQueue) Enqueue(companyID, jobType, idempotencyKey string, payload interface{}, availableAt time.Time) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	if availableAt.IsZero() {
		availableAt = time.Now()
	}
	id := uuid.New().String()
	err = q.db.QueryRow(`
		INSERT INTO durable_jobs
			(id, company_id, job_type, idempotency_key, payload, available_at)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6)
		ON CONFLICT (job_type, idempotency_key) DO UPDATE SET
			available_at = LEAST(durable_jobs.available_at, EXCLUDED.available_at),
			status = CASE WHEN durable_jobs.status IN ('completed', 'dead') THEN 'pending' ELSE durable_jobs.status END,
			attempts = CASE WHEN durable_jobs.status IN ('completed', 'dead') THEN 0 ELSE durable_jobs.attempts END,
			completed_at = CASE WHEN durable_jobs.status IN ('completed', 'dead') THEN NULL ELSE durable_jobs.completed_at END,
			last_error = CASE WHEN durable_jobs.status IN ('completed', 'dead') THEN NULL ELSE durable_jobs.last_error END,
			payload = EXCLUDED.payload,
			updated_at = NOW()
		RETURNING id
	`, id, companyID, jobType, idempotencyKey, raw, availableAt).Scan(&id)
	return id, err
}

func (q *JobQueue) Start(ctx context.Context, workers int) {
	if workers < 1 {
		workers = 1
	}
	q.mu.Lock()
	q.started = time.Now()
	q.mu.Unlock()
	q.recoverStaleJobs(ctx)
	for index := 0; index < workers; index++ {
		go q.runWorker(ctx)
	}
}

func (q *JobQueue) runWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			q.mu.Lock()
			q.lastPoll = time.Now()
			q.mu.Unlock()
			q.recoverStaleJobs(ctx)
			if err := q.processOne(ctx); err != nil && err != sql.ErrNoRows {
				q.mu.Lock()
				q.lastErr = err.Error()
				q.mu.Unlock()
				log.Printf("[JOBS] worker %s: %v", q.workerID, err)
			}
		}
	}
}

func (q *JobQueue) recoverStaleJobs(ctx context.Context) {
	_, _ = q.db.ExecContext(ctx, `
		UPDATE durable_jobs
		SET status = 'pending', locked_at = NULL, locked_by = NULL,
			available_at = NOW(), updated_at = NOW()
		WHERE status = 'processing' AND locked_at < NOW() - INTERVAL '2 minutes'
	`)
}

func (q *JobQueue) processOne(ctx context.Context) error {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var id, jobType string
	var payload json.RawMessage
	var attempts, maxAttempts int
	err = tx.QueryRowContext(ctx, `
		SELECT id, job_type, payload, attempts, max_attempts
		FROM durable_jobs
		WHERE status = 'pending' AND available_at <= NOW()
		ORDER BY available_at, created_at
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`).Scan(&id, &jobType, &payload, &attempts, &maxAttempts)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE durable_jobs SET status = 'processing', locked_at = NOW(),
			locked_by = $2, attempts = attempts + 1, updated_at = NOW()
		WHERE id = $1
	`, id, q.workerID)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}

	q.mu.RLock()
	handler := q.handlers[jobType]
	q.mu.RUnlock()
	if handler == nil {
		return q.failJob(id, jobType, payload, attempts+1, maxAttempts, fmt.Errorf("no handler registered"))
	}
	runCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	heartbeatDone := make(chan struct{})
	go q.heartbeatJob(runCtx, id, heartbeatDone)
	if err := handler(runCtx, payload); err != nil {
		close(heartbeatDone)
		return q.failJob(id, jobType, payload, attempts+1, maxAttempts, err)
	}
	close(heartbeatDone)
	_, err = q.db.ExecContext(ctx, `
		UPDATE durable_jobs SET status = 'completed', completed_at = NOW(),
			locked_at = NULL, locked_by = NULL, last_error = NULL, updated_at = NOW()
		WHERE id = $1
	`, id)
	return err
}

func (q *JobQueue) heartbeatJob(ctx context.Context, id string, done <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			_, _ = q.db.ExecContext(ctx, `
				UPDATE durable_jobs SET locked_at = NOW(), updated_at = NOW()
				WHERE id = $1 AND status = 'processing' AND locked_by = $2
			`, id, q.workerID)
		}
	}
}

func (q *JobQueue) failJob(id, jobType string, payload json.RawMessage, attempts, maxAttempts int, runErr error) error {
	if attempts >= maxAttempts {
		tx, err := q.db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		_, err = tx.Exec(`
			INSERT INTO dead_letter_jobs
				(id, original_job_id, company_id, job_type, payload, attempts, last_error)
			SELECT $2, id, company_id, job_type, payload, attempts, $3
			FROM durable_jobs WHERE id = $1
		`, id, uuid.New().String(), runErr.Error())
		if err != nil {
			return err
		}
		_, err = tx.Exec(`
			UPDATE durable_jobs SET status = 'dead', last_error = $2,
				locked_at = NULL, locked_by = NULL, updated_at = NOW()
			WHERE id = $1
		`, id, runErr.Error())
		if err != nil {
			return err
		}
		return tx.Commit()
	}
	delay := time.Duration(math.Pow(2, float64(attempts-1))) * time.Minute
	_, err := q.db.Exec(`
		UPDATE durable_jobs SET status = 'pending', last_error = $2,
			available_at = NOW() + ($3 * INTERVAL '1 second'),
			locked_at = NULL, locked_by = NULL, updated_at = NOW()
		WHERE id = $1
	`, id, runErr.Error(), int(delay.Seconds()))
	return err
}

func (q *JobQueue) Health(ctx context.Context) map[string]interface{} {
	var pending, processing, dead int
	err := q.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'pending'),
			COUNT(*) FILTER (WHERE status = 'processing'),
			COUNT(*) FILTER (WHERE status = 'dead')
		FROM durable_jobs
	`).Scan(&pending, &processing, &dead)
	q.mu.RLock()
	started, lastPoll, lastErr := q.started, q.lastPoll, q.lastErr
	q.mu.RUnlock()
	healthy := err == nil && !started.IsZero() && (lastPoll.IsZero() || time.Since(lastPoll) < 10*time.Second)
	status := "ok"
	if !healthy {
		status = "error"
	}
	return map[string]interface{}{
		"status":    status,
		"worker_id": q.workerID, "started_at": started, "last_poll_at": lastPoll,
		"pending": pending, "processing": processing, "dead": dead,
		"last_error": lastErr, "database_error": errorString(err),
	}
}

func (q *JobQueue) RetryDeadLetter(deadLetterID, companyID string, allowAnyCompany bool) error {
	tx, err := q.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	filter := "AND (dead.company_id = NULLIF($2, '')::uuid)"
	if allowAnyCompany {
		filter = ""
	}
	query := `
		UPDATE durable_jobs job SET
			status = 'pending', attempts = 0, available_at = NOW(), last_error = NULL,
			locked_at = NULL, locked_by = NULL, completed_at = NULL, updated_at = NOW()
		FROM dead_letter_jobs dead
		WHERE dead.id = $1 AND job.id = dead.original_job_id ` + filter
	args := []interface{}{deadLetterID}
	if !allowAnyCompany {
		args = append(args, companyID)
	}
	result, err := tx.Exec(query, args...)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return sql.ErrNoRows
	}
	if allowAnyCompany {
		_, err = tx.Exec(`DELETE FROM dead_letter_jobs WHERE id = $1`, deadLetterID)
	} else {
		_, err = tx.Exec(`DELETE FROM dead_letter_jobs WHERE id = $1 AND company_id = $2`, deadLetterID, companyID)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
