package actionquery

import (
	"context"

	workerport "go-control-plane/internal/ports/outbound/worker"
)

type ListActionsResult struct {
	Actions []map[string]any `json:"actions"`
}

type ListActionsHandler struct {
	worker workerport.Client
}

func NewListActionsHandler(worker workerport.Client) ListActionsHandler {
	return ListActionsHandler{worker: worker}
}

func (h ListActionsHandler) Handle(ctx context.Context, platform string) (ListActionsResult, error) {
	resp, err := h.worker.ListActions(ctx, platform)
	if err != nil {
		return ListActionsResult{}, err
	}
	return ListActionsResult{Actions: resp.Actions}, nil
}
