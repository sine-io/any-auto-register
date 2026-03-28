package integrationcommand

import (
	"context"

	workerport "go-control-plane/internal/ports/outbound/worker"
)

type Handler struct {
	worker workerport.Client
}

func NewHandler(worker workerport.Client) Handler {
	return Handler{worker: worker}
}

func (h Handler) StartAll(ctx context.Context) (map[string]any, error) {
	resp, err := h.worker.StartAllIntegrationServices(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": resp.Items}, nil
}

func (h Handler) StopAll(ctx context.Context) (map[string]any, error) {
	resp, err := h.worker.StopAllIntegrationServices(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": resp.Items}, nil
}

func (h Handler) Start(ctx context.Context, name string) (map[string]any, error) {
	return h.worker.StartIntegrationService(ctx, name)
}

func (h Handler) Install(ctx context.Context, name string) (map[string]any, error) {
	return h.worker.InstallIntegrationService(ctx, name)
}

func (h Handler) Stop(ctx context.Context, name string) (map[string]any, error) {
	return h.worker.StopIntegrationService(ctx, name)
}

func (h Handler) Backfill(ctx context.Context, platforms []string) (map[string]any, error) {
	return h.worker.BackfillIntegrations(ctx, platforms)
}
