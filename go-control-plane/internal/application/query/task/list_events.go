package taskquery

import "context"

type TaskEventItem struct {
	ID      int64  `json:"id"`
	Message string `json:"message"`
}

type ListEventsQuery struct {
	TaskID  string
	SinceID int64
}

type ListEventsResult struct {
	Items []TaskEventItem `json:"items"`
}

type ListEventsRepository interface {
	ListEvents(context.Context, string, int64) ([]TaskEventItem, error)
}

type ListEventsHandler struct {
	repo ListEventsRepository
}

func NewListEventsHandler(repo ListEventsRepository) ListEventsHandler {
	return ListEventsHandler{repo: repo}
}

func (h ListEventsHandler) Handle(ctx context.Context, query ListEventsQuery) (ListEventsResult, error) {
	items, err := h.repo.ListEvents(ctx, query.TaskID, query.SinceID)
	if err != nil {
		return ListEventsResult{}, err
	}
	return ListEventsResult{Items: items}, nil
}
