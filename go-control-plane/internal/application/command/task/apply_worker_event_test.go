package taskcommand

import (
	"context"
	"testing"

	domaintask "go-control-plane/internal/domain/task"
)

type fakeWorkerEventRepository struct {
	events []domaintask.WorkerEvent
}

func (f *fakeWorkerEventRepository) ApplyWorkerEvent(_ context.Context, event domaintask.WorkerEvent) error {
	f.events = append(f.events, event)
	return nil
}

func TestApplyWorkerEventHandlerStoresEvent(t *testing.T) {
	repo := &fakeWorkerEventRepository{}
	handler := NewApplyWorkerEventHandler(repo)

	err := handler.Handle(context.Background(), ApplyWorkerEventCommand{
		TaskID:  "task_1",
		Type:    domaintask.WorkerEventLog,
		Message: "hello",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(repo.events) != 1 || repo.events[0].Message != "hello" {
		t.Fatalf("unexpected events: %#v", repo.events)
	}
}
