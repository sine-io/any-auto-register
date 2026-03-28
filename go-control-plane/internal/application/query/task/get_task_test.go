package taskquery

import (
	"context"
	"testing"
	"time"

	domaintask "go-control-plane/internal/domain/task"
)

type fakeTaskDetailRepository struct {
	item domaintask.TaskRun
}

func (f fakeTaskDetailRepository) GetByID(context.Context, string) (domaintask.TaskRun, error) {
	return f.item, nil
}

func TestGetTaskReturnsProjectedTask(t *testing.T) {
	handler := NewGetTaskHandler(fakeTaskDetailRepository{
		item: domaintask.TaskRun{
			ID:              "task_1",
			Platform:        "dummy",
			Status:          "done",
			ProgressCurrent: 1,
			ProgressTotal:   1,
			SuccessCount:    1,
			ErrorSummary:    "none",
			CashierURLs:     []string{"https://example.com/pay"},
			CreatedAt:       time.Unix(1, 0).UTC(),
		},
	})
	result, err := handler.Handle(context.Background(), GetTaskQuery{TaskID: "task_1"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.ID != "task_1" || result.Progress != "1/1" {
		t.Fatalf("unexpected task result: %#v", result)
	}
	if result.Success != 1 || len(result.CashierURLs) != 1 {
		t.Fatalf("expected task payload fields, got %#v", result)
	}
}
