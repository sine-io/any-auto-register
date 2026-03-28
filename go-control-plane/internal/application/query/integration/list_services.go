package integrationquery

import (
	"context"

	workerport "go-control-plane/internal/ports/outbound/worker"
)

type ListServicesHandler struct {
	worker workerport.Client
}

func NewListServicesHandler(worker workerport.Client) ListServicesHandler {
	return ListServicesHandler{worker: worker}
}

func (h ListServicesHandler) Handle(ctx context.Context) (map[string]any, error) {
	resp, err := h.worker.ListIntegrationServices(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": resp.Items}, nil
}
