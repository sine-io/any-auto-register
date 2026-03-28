package systemcommand

import (
	"context"
	"testing"

	workerport "go-control-plane/internal/ports/outbound/worker"
)

type fakeSystemCommandWorker struct{}

func (fakeSystemCommandWorker) Register(context.Context, workerport.RegisterRequest) (workerport.RegisterResponse, error) {
	return workerport.RegisterResponse{}, nil
}
func (fakeSystemCommandWorker) CheckAccount(context.Context, workerport.CheckAccountRequest) (workerport.CheckAccountResponse, error) {
	return workerport.CheckAccountResponse{}, nil
}
func (fakeSystemCommandWorker) ExecuteAction(context.Context, workerport.ExecuteActionRequest) (workerport.ExecuteActionResponse, error) {
	return workerport.ExecuteActionResponse{}, nil
}
func (fakeSystemCommandWorker) GetSolverStatus(context.Context) (workerport.SolverStatusResponse, error) {
	return workerport.SolverStatusResponse{}, nil
}

func (fakeSystemCommandWorker) RestartSolver(context.Context) (map[string]any, error) {
	return map[string]any{"message": "重启中"}, nil
}
func (fakeSystemCommandWorker) ListIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (fakeSystemCommandWorker) StartAllIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (fakeSystemCommandWorker) StopAllIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (fakeSystemCommandWorker) StartIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (fakeSystemCommandWorker) InstallIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (fakeSystemCommandWorker) StopIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (fakeSystemCommandWorker) BackfillIntegrations(context.Context, []string) (map[string]any, error) {
	return map[string]any{}, nil
}

func TestRestartSolverHandlerReturnsWorkerMessage(t *testing.T) {
	handler := NewRestartSolverHandler(fakeSystemCommandWorker{})
	result, err := handler.Handle(context.Background())
	if err != nil || result["message"] != "重启中" {
		t.Fatalf("unexpected restart result: %#v err=%v", result, err)
	}
}
