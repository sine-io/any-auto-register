package sqliteadapter

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	taskquery "go-control-plane/internal/application/query/task"
	domaintask "go-control-plane/internal/domain/task"
)

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) TaskRepository {
	return TaskRepository{db: db}
}

func (r TaskRepository) List(ctx context.Context, filter domaintask.ListFilter) (int, []domaintask.TaskRun, error) {
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM task_runs`).Scan(&total); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return 0, []domaintask.TaskRun{}, nil
		}
		return 0, nil, err
	}

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, platform, status, progress_current, progress_total, success_count, error_count, error_summary, errors_json, cashier_urls_json, created_at, updated_at
		 FROM task_runs
		 ORDER BY datetime(created_at) DESC
		 LIMIT ? OFFSET ?`,
		pageSize,
		(page-1)*pageSize,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return 0, []domaintask.TaskRun{}, nil
		}
		return 0, nil, err
	}
	defer rows.Close()

	items := make([]domaintask.TaskRun, 0)
	for rows.Next() {
		var item domaintask.TaskRun
		var errorsJSON string
		var cashierURLsJSON string
		var createdAt string
		var updatedAt string
		if err := rows.Scan(
			&item.ID,
			&item.Platform,
			&item.Status,
			&item.ProgressCurrent,
			&item.ProgressTotal,
			&item.SuccessCount,
			&item.ErrorCount,
			&item.ErrorSummary,
			&errorsJSON,
			&cashierURLsJSON,
			&createdAt,
			&updatedAt,
		); err != nil {
			return 0, nil, err
		}
		item.Errors = decodeStringSlice(errorsJSON)
		item.CashierURLs = decodeStringSlice(cashierURLsJSON)
		item.CreatedAt = parseTime(createdAt)
		item.UpdatedAt = parseTime(updatedAt)
		items = append(items, item)
	}

	return total, items, rows.Err()
}

func (r TaskRepository) GetByID(ctx context.Context, taskID string) (domaintask.TaskRun, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, platform, status, progress_current, progress_total, success_count, error_count, error_summary, errors_json, cashier_urls_json, created_at, updated_at
		 FROM task_runs WHERE id = ?`,
		taskID,
	)
	var item domaintask.TaskRun
	var errorsJSON string
	var cashierURLsJSON string
	var createdAt string
	var updatedAt string
	if err := row.Scan(&item.ID, &item.Platform, &item.Status, &item.ProgressCurrent, &item.ProgressTotal, &item.SuccessCount, &item.ErrorCount, &item.ErrorSummary, &errorsJSON, &cashierURLsJSON, &createdAt, &updatedAt); err != nil {
		return domaintask.TaskRun{}, err
	}
	item.Errors = decodeStringSlice(errorsJSON)
	item.CashierURLs = decodeStringSlice(cashierURLsJSON)
	item.CreatedAt = parseTime(createdAt)
	item.UpdatedAt = parseTime(updatedAt)
	return item, nil
}

func (r TaskRepository) ListLogs(ctx context.Context, filter taskquery.ListLogsFilter) (int, []taskquery.TaskLogItem, error) {
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}

	clauses := []string{}
	args := []any{}
	if filter.Platform != "" {
		clauses = append(clauses, "platform = ?")
		args = append(args, filter.Platform)
	}
	whereSQL := ""
	if len(clauses) > 0 {
		whereSQL = " WHERE " + strings.Join(clauses, " AND ")
	}

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM task_logs`+whereSQL, args...).Scan(&total); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return 0, []taskquery.TaskLogItem{}, nil
		}
		return 0, nil, err
	}

	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, pageSize, (page-1)*pageSize)
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, platform, email, status, error, created_at FROM task_logs`+whereSQL+` ORDER BY id DESC LIMIT ? OFFSET ?`,
		queryArgs...,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return 0, []taskquery.TaskLogItem{}, nil
		}
		return 0, nil, err
	}
	defer rows.Close()

	items := make([]taskquery.TaskLogItem, 0)
	for rows.Next() {
		var item taskquery.TaskLogItem
		var createdAt string
		if err := rows.Scan(&item.ID, &item.Platform, &item.Email, &item.Status, &item.Error, &createdAt); err != nil {
			return 0, nil, err
		}
		item.CreatedAt = parseTime(createdAt)
		items = append(items, item)
	}
	return total, items, rows.Err()
}

func (r TaskRepository) ListEvents(ctx context.Context, taskID string, sinceID int64) ([]taskquery.TaskEventItem, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, message FROM task_events WHERE task_id = ? AND id > ? ORDER BY id`,
		taskID,
		sinceID,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return []taskquery.TaskEventItem{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	items := make([]taskquery.TaskEventItem, 0)
	for rows.Next() {
		var item taskquery.TaskEventItem
		if err := rows.Scan(&item.ID, &item.Message); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func decodeStringSlice(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []string{}
	}
	return items
}

func parseTime(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return parsed
	}
	if parsed, err := time.Parse("2006-01-02 15:04:05.999999", raw); err == nil {
		return parsed
	}
	return time.Time{}
}
