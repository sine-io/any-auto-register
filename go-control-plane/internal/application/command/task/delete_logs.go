package taskcommand

import "context"

type DeleteTaskLogsCommand struct {
	IDs []int64 `json:"ids"`
}

type DeleteTaskLogsResult struct {
	Deleted        int     `json:"deleted"`
	NotFound       []int64 `json:"not_found"`
	TotalRequested int     `json:"total_requested"`
}

type DeleteTaskLogsRepository interface {
	DeleteTaskLogs(context.Context, []int64) (int, []int64, error)
}

type DeleteTaskLogsHandler struct {
	repo DeleteTaskLogsRepository
}

func NewDeleteTaskLogsHandler(repo DeleteTaskLogsRepository) DeleteTaskLogsHandler {
	return DeleteTaskLogsHandler{repo: repo}
}

func (h DeleteTaskLogsHandler) Handle(ctx context.Context, cmd DeleteTaskLogsCommand) (DeleteTaskLogsResult, error) {
	deleted, notFound, err := h.repo.DeleteTaskLogs(ctx, cmd.IDs)
	if err != nil {
		return DeleteTaskLogsResult{}, err
	}
	return DeleteTaskLogsResult{
		Deleted:        deleted,
		NotFound:       notFound,
		TotalRequested: len(cmd.IDs),
	}, nil
}
