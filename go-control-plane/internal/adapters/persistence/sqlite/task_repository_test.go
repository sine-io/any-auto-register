package sqliteadapter

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	domaintask "go-control-plane/internal/domain/task"
)

func createTaskTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "tasks.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("expected sqlite db to open, got %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE task_runs (
			id TEXT PRIMARY KEY,
			platform TEXT NOT NULL,
			status TEXT NOT NULL,
			progress_current INTEGER NOT NULL,
			progress_total INTEGER NOT NULL,
			success_count INTEGER NOT NULL,
			error_count INTEGER NOT NULL,
			error_summary TEXT NOT NULL,
			errors_json TEXT NOT NULL,
			request_json TEXT NOT NULL,
			cashier_urls_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("expected schema setup to succeed, got %v", err)
	}

	now := time.Now().UTC()
	for i, id := range []string{"task_1", "task_2", "task_3"} {
		_, err = db.Exec(
			`INSERT INTO task_runs (id, platform, status, progress_current, progress_total, success_count, error_count, error_summary, errors_json, request_json, cashier_urls_json, created_at, updated_at)
			 VALUES (?, 'dummy', 'done', 1, 1, 1, 0, '', '[]', '{}', '[]', ?, ?)`,
			id,
			now.Add(time.Duration(i)*time.Minute).Format(time.RFC3339Nano),
			now.Add(time.Duration(i)*time.Minute).Format(time.RFC3339Nano),
		)
		if err != nil {
			t.Fatalf("expected seed insert to succeed, got %v", err)
		}
	}

	return db
}

func TestOpenSupportsAbsoluteSQLiteURL(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "absolute.db")
	uri := "sqlite:////" + dbPath[1:]
	db, err := Open(uri)
	if err != nil {
		t.Fatalf("expected open to succeed, got %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE demo (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("expected absolute sqlite path to be writable, got %v", err)
	}
}

func TestTaskRepositoryListsPaginatedRuns(t *testing.T) {
	db := createTaskTestDB(t)
	repo := NewTaskRepository(db)

	total, items, err := repo.List(context.Background(), domaintask.ListFilter{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total != 3 {
		t.Fatalf("expected total 3, got %d", total)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item on second page, got %d", len(items))
	}
	if items[0].ID != "task_1" {
		t.Fatalf("expected oldest task on page 2, got %s", items[0].ID)
	}
}

func TestTaskRepositoryReturnsEmptyWhenTableMissing(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "empty.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("expected sqlite db to open, got %v", err)
	}

	repo := NewTaskRepository(db)
	total, items, err := repo.List(context.Background(), domaintask.ListFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("expected missing table to degrade gracefully, got %v", err)
	}
	if total != 0 {
		t.Fatalf("expected total 0, got %d", total)
	}
	if len(items) != 0 {
		t.Fatalf("expected no items, got %d", len(items))
	}
}
