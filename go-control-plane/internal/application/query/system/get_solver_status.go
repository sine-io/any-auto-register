package systemquery

import (
	"context"

	workerport "go-control-plane/internal/ports/outbound/worker"
)

type SolverStatusResult struct {
	Running bool `json:"running"`
}

type SolverStatusHandler struct {
	worker workerport.Client
}

func NewSolverStatusHandler(worker workerport.Client) SolverStatusHandler {
	return SolverStatusHandler{worker: worker}
}

func (h SolverStatusHandler) Handle(ctx context.Context) (SolverStatusResult, error) {
	resp, err := h.worker.GetSolverStatus(ctx)
	if err != nil {
		return SolverStatusResult{}, err
	}
	return SolverStatusResult{Running: resp.Running}, nil
}
