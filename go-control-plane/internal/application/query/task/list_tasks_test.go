package taskquery

import (
	"context"
	"testing"

	domaintask "go-control-plane/internal/domain/task"
)

type fakeTaskRepository struct {
	total int
	items []domaintask.TaskRun
}

func (f fakeTaskRepository) List(ctx context.Context, filter domaintask.ListFilter) (int, []domaintask.TaskRun, error) {
	return f.total, f.items, nil
}

func TestHandlerReturnsPaginatedTasks(t *testing.T) {
	handler := NewHandler(fakeTaskRepository{
		total: 3,
		items: []domaintask.TaskRun{
			{ID: "task_1", Platform: "dummy", Status: "done", ProgressCurrent: 1, ProgressTotal: 1},
		},
	})

	result, err := handler.Handle(context.Background(), Query{Page: 2, PageSize: 1})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Total != 3 {
		t.Fatalf("expected total 3, got %d", result.Total)
	}
	if result.Page != 2 {
		t.Fatalf("expected page 2, got %d", result.Page)
	}
	if len(result.Items) != 1 || result.Items[0].ID != "task_1" {
		t.Fatalf("unexpected items: %#v", result.Items)
	}
}
