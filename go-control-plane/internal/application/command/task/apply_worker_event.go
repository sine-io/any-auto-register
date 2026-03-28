package taskcommand

import (
	"context"
	"time"

	domaintask "go-control-plane/internal/domain/task"
)

type ApplyWorkerEventCommand struct {
	TaskID          string
	Type            domaintask.WorkerEventType
	Message         string
	ProgressCurrent int
	ProgressTotal   int
	SuccessCount    int
	ErrorCount      int
	ErrorSummary    string
	Errors          []string
	CashierURLs     []string
}

type WorkerEventRepository interface {
	ApplyWorkerEvent(ctx context.Context, event domaintask.WorkerEvent) error
}

type ApplyWorkerEventHandler struct {
	repo WorkerEventRepository
	now  func() time.Time
}

func NewApplyWorkerEventHandler(repo WorkerEventRepository, now ...func() time.Time) ApplyWorkerEventHandler {
	clock := func() time.Time { return time.Now().UTC() }
	if len(now) > 0 && now[0] != nil {
		clock = now[0]
	}
	return ApplyWorkerEventHandler{repo: repo, now: clock}
}

func (h ApplyWorkerEventHandler) Handle(ctx context.Context, cmd ApplyWorkerEventCommand) error {
	return h.repo.ApplyWorkerEvent(ctx, domaintask.WorkerEvent{
		TaskID:          cmd.TaskID,
		Type:            cmd.Type,
		Message:         cmd.Message,
		ProgressCurrent: cmd.ProgressCurrent,
		ProgressTotal:   cmd.ProgressTotal,
		SuccessCount:    cmd.SuccessCount,
		ErrorCount:      cmd.ErrorCount,
		ErrorSummary:    cmd.ErrorSummary,
		Errors:          cmd.Errors,
		CashierURLs:     cmd.CashierURLs,
		OccurredAt:      h.now(),
	})
}
