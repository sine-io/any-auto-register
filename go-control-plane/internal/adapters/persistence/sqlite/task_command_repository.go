package sqliteadapter

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	taskcommand "go-control-plane/internal/application/command/task"
	domaintask "go-control-plane/internal/domain/task"
)

type TaskCommandRepository struct {
	db *sql.DB
}

func NewTaskCommandRepository(db *sql.DB) TaskCommandRepository {
	return TaskCommandRepository{db: db}
}

func (r TaskCommandRepository) Create(ctx context.Context, run domaintask.TaskRun, requestJSON string, errorsJSON string, cashierURLsJSON string) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO task_runs (id, platform, status, progress_current, progress_total, success_count, error_count, error_summary, errors_json, request_json, cashier_urls_json, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID,
		run.Platform,
		run.Status,
		run.ProgressCurrent,
		run.ProgressTotal,
		run.SuccessCount,
		run.ErrorCount,
		run.ErrorSummary,
		errorsJSON,
		requestJSON,
		cashierURLsJSON,
		run.CreatedAt.Format(time.RFC3339Nano),
		run.UpdatedAt.Format(time.RFC3339Nano),
	)
	return err
}

func (r TaskCommandRepository) AppendEvents(ctx context.Context, taskID string, messages []string) error {
	for _, message := range messages {
		if _, err := r.db.ExecContext(
			ctx,
			`INSERT INTO task_events (task_id, level, message, created_at) VALUES (?, 'info', ?, ?)`,
			taskID,
			message,
			time.Now().UTC().Format(time.RFC3339Nano),
		); err != nil {
			return err
		}
	}
	return nil
}

func (r TaskCommandRepository) UpdateResult(ctx context.Context, result taskcommand.UpdateResult) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE task_runs
		 SET status = ?, progress_current = ?, progress_total = ?, success_count = ?, error_count = ?, error_summary = ?, errors_json = ?, cashier_urls_json = ?, updated_at = ?
		 WHERE id = ?`,
		result.Status,
		result.ProgressCurrent,
		result.ProgressTotal,
		result.SuccessCount,
		result.ErrorCount,
		result.ErrorSummary,
		result.ErrorsJSON,
		result.CashierURLsJSON,
		result.UpdatedAt.Format(time.RFC3339Nano),
		result.TaskID,
	)
	return err
}

func (r TaskCommandRepository) ApplyWorkerEvent(ctx context.Context, event domaintask.WorkerEvent) error {
	switch event.Type {
	case domaintask.WorkerEventStarted:
		_, err := r.db.ExecContext(
			ctx,
			`UPDATE task_runs SET status = ?, updated_at = ? WHERE id = ?`,
			"running",
			event.OccurredAt.Format(time.RFC3339Nano),
			event.TaskID,
		)
		return err
	case domaintask.WorkerEventProgress:
		_, err := r.db.ExecContext(
			ctx,
			`UPDATE task_runs SET progress_current = ?, progress_total = ?, updated_at = ? WHERE id = ?`,
			event.ProgressCurrent,
			event.ProgressTotal,
			event.OccurredAt.Format(time.RFC3339Nano),
			event.TaskID,
		)
		return err
	case domaintask.WorkerEventLog:
		_, err := r.db.ExecContext(
			ctx,
			`INSERT INTO task_events (task_id, level, message, created_at) VALUES (?, 'info', ?, ?)`,
			event.TaskID,
			event.Message,
			event.OccurredAt.Format(time.RFC3339Nano),
		)
		return err
	case domaintask.WorkerEventSucceeded, domaintask.WorkerEventFailed:
		status := "done"
		if event.Type == domaintask.WorkerEventFailed {
			status = "failed"
		}
		errorsJSON, _ := json.Marshal(event.Errors)
		cashierURLsJSON, _ := json.Marshal(event.CashierURLs)
		_, err := r.db.ExecContext(
			ctx,
			`UPDATE task_runs
			 SET status = ?, success_count = ?, error_count = ?, error_summary = ?, errors_json = ?, cashier_urls_json = ?, updated_at = ?
			 WHERE id = ?`,
			status,
			event.SuccessCount,
			event.ErrorCount,
			event.ErrorSummary,
			string(errorsJSON),
			string(cashierURLsJSON),
			event.OccurredAt.Format(time.RFC3339Nano),
			event.TaskID,
		)
		return err
	default:
		return nil
	}
}
