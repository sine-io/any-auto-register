package taskquery

import (
	"context"
	"testing"
	"time"
)

type fakeTaskLogRepository struct {
	total int
	items []TaskLogItem
}

func (f fakeTaskLogRepository) ListLogs(context.Context, ListLogsFilter) (int, []TaskLogItem, error) {
	return f.total, f.items, nil
}

func TestListTaskLogsReturnsPaginatedLogs(t *testing.T) {
	handler := NewListLogsHandler(fakeTaskLogRepository{
		total: 1,
		items: []TaskLogItem{{ID: 1, Platform: "dummy", Email: "user@example.com", Status: "success", CreatedAt: time.Unix(1, 0).UTC()}},
	})
	result, err := handler.Handle(context.Background(), ListLogsQuery{Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("unexpected log result: %#v", result)
	}
}
