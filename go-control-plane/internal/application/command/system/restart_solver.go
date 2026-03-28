package systemcommand

import (
	"context"

	workerport "go-control-plane/internal/ports/outbound/worker"
)

type RestartSolverHandler struct {
	worker workerport.Client
}

func NewRestartSolverHandler(worker workerport.Client) RestartSolverHandler {
	return RestartSolverHandler{worker: worker}
}

func (h RestartSolverHandler) Handle(ctx context.Context) (map[string]any, error) {
	return h.worker.RestartSolver(ctx)
}
