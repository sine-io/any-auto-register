package systemquery

import (
	"context"
	"testing"

	workerport "go-control-plane/internal/ports/outbound/worker"
)

type fakeSystemWorker struct{}

func (fakeSystemWorker) Register(context.Context, workerport.RegisterRequest) (workerport.RegisterResponse, error) {
	return workerport.RegisterResponse{}, nil
}
func (fakeSystemWorker) CheckAccount(context.Context, workerport.CheckAccountRequest) (workerport.CheckAccountResponse, error) {
	return workerport.CheckAccountResponse{}, nil
}
func (fakeSystemWorker) ExecuteAction(context.Context, workerport.ExecuteActionRequest) (workerport.ExecuteActionResponse, error) {
	return workerport.ExecuteActionResponse{}, nil
}
func (fakeSystemWorker) ListActions(context.Context, string) (workerport.ListActionsResponse, error) {
	return workerport.ListActionsResponse{}, nil
}
func (fakeSystemWorker) GetSolverStatus(context.Context) (workerport.SolverStatusResponse, error) {
	return workerport.SolverStatusResponse{Running: true, Status: "running", Reason: ""}, nil
}
func (fakeSystemWorker) RestartSolver(context.Context) (map[string]any, error) {
	return map[string]any{}, nil
}
func (fakeSystemWorker) ListIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (fakeSystemWorker) StartAllIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (fakeSystemWorker) StopAllIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (fakeSystemWorker) StartIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (fakeSystemWorker) InstallIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (fakeSystemWorker) StopIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (fakeSystemWorker) BackfillIntegrations(context.Context, []string) (map[string]any, error) {
	return map[string]any{}, nil
}

func TestSolverStatusHandlerReturnsWorkerState(t *testing.T) {
	handler := NewSolverStatusHandler(fakeSystemWorker{})
	result, err := handler.Handle(context.Background())
	if err != nil || !result.Running {
		t.Fatalf("unexpected solver result: %#v err=%v", result, err)
	}
	if result.Status != "running" || result.Reason != "" {
		t.Fatalf("expected rich solver state, got %#v", result)
	}
}
