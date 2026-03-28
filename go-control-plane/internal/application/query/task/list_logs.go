package taskquery

import (
	"context"
	"time"
)

type TaskLogItem struct {
	ID        int64     `json:"id"`
	Platform  string    `json:"platform"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	Error     string    `json:"error"`
	CreatedAt time.Time `json:"created_at"`
}

type ListLogsFilter struct {
	Platform string
	Page     int
	PageSize int
}

type ListLogsQuery = ListLogsFilter

type ListLogsResult struct {
	Total int           `json:"total"`
	Page  int           `json:"page"`
	Items []TaskLogItem `json:"items"`
}

type ListLogsRepository interface {
	ListLogs(context.Context, ListLogsFilter) (int, []TaskLogItem, error)
}

type ListLogsHandler struct {
	repo ListLogsRepository
}

func NewListLogsHandler(repo ListLogsRepository) ListLogsHandler {
	return ListLogsHandler{repo: repo}
}

func (h ListLogsHandler) Handle(ctx context.Context, query ListLogsQuery) (ListLogsResult, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	total, items, err := h.repo.ListLogs(ctx, ListLogsFilter{
		Platform: query.Platform,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return ListLogsResult{}, err
	}
	return ListLogsResult{Total: total, Page: page, Items: items}, nil
}
