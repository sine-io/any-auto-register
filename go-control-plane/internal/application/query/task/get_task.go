package taskquery

import (
	"context"

	domaintask "go-control-plane/internal/domain/task"
)

type GetTaskQuery struct {
	TaskID string
}

type GetTaskRepository interface {
	GetByID(context.Context, string) (domaintask.TaskRun, error)
}

type GetTaskHandler struct {
	repo GetTaskRepository
}

func NewGetTaskHandler(repo GetTaskRepository) GetTaskHandler {
	return GetTaskHandler{repo: repo}
}

func (h GetTaskHandler) Handle(ctx context.Context, query GetTaskQuery) (TaskItem, error) {
	task, err := h.repo.GetByID(ctx, query.TaskID)
	if err != nil {
		return TaskItem{}, err
	}
	return TaskItem{
		ID:          task.ID,
		Platform:    task.Platform,
		Status:      task.Status,
		Progress:    task.Progress(),
		Success:     task.SuccessCount,
		Errors:      task.Errors,
		Error:       task.ErrorSummary,
		CashierURLs: task.CashierURLs,
	}, nil
}
