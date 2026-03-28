package taskquery

import (
	"context"

	domaintask "go-control-plane/internal/domain/task"
)

type Query struct {
	Page     int
	PageSize int
}

type TaskItem struct {
	ID          string   `json:"id"`
	Platform    string   `json:"platform"`
	Status      string   `json:"status"`
	Progress    string   `json:"progress"`
	Success     int      `json:"success"`
	Errors      []string `json:"errors"`
	Error       string   `json:"error"`
	CashierURLs []string `json:"cashier_urls"`
}

type Result struct {
	Total int        `json:"total"`
	Page  int        `json:"page"`
	Items []TaskItem `json:"items"`
}

type Handler struct {
	repo domaintask.Repository
}

func NewHandler(repo domaintask.Repository) Handler {
	return Handler{repo: repo}
}

func (h Handler) Handle(ctx context.Context, query Query) (Result, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}

	total, tasks, err := h.repo.List(ctx, domaintask.ListFilter{Page: page, PageSize: pageSize})
	if err != nil {
		return Result{}, err
	}

	items := make([]TaskItem, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, TaskItem{
			ID:          task.ID,
			Platform:    task.Platform,
			Status:      task.Status,
			Progress:    task.Progress(),
			Success:     task.SuccessCount,
			Errors:      task.Errors,
			Error:       task.ErrorSummary,
			CashierURLs: task.CashierURLs,
		})
	}

	return Result{
		Total: total,
		Page:  page,
		Items: items,
	}, nil
}
