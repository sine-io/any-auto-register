package integrationquery

import (
	"context"
	"testing"

	workerport "go-control-plane/internal/ports/outbound/worker"
)

type fakeIntegrationWorker struct{}

func (fakeIntegrationWorker) Register(context.Context, workerport.RegisterRequest) (workerport.RegisterResponse, error) {
	return workerport.RegisterResponse{}, nil
}
func (fakeIntegrationWorker) CheckAccount(context.Context, workerport.CheckAccountRequest) (workerport.CheckAccountResponse, error) {
	return workerport.CheckAccountResponse{}, nil
}
func (fakeIntegrationWorker) ExecuteAction(context.Context, workerport.ExecuteActionRequest) (workerport.ExecuteActionResponse, error) {
	return workerport.ExecuteActionResponse{}, nil
}
func (fakeIntegrationWorker) ListActions(context.Context, string) (workerport.ListActionsResponse, error) {
	return workerport.ListActionsResponse{}, nil
}
func (fakeIntegrationWorker) GetSolverStatus(context.Context) (workerport.SolverStatusResponse, error) {
	return workerport.SolverStatusResponse{}, nil
}
func (fakeIntegrationWorker) RestartSolver(context.Context) (map[string]any, error) {
	return map[string]any{}, nil
}

func (fakeIntegrationWorker) ListIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{Items: []map[string]any{{"name": "grok2api"}}}, nil
}
func (fakeIntegrationWorker) StartAllIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (fakeIntegrationWorker) StopAllIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (fakeIntegrationWorker) StartIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (fakeIntegrationWorker) InstallIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (fakeIntegrationWorker) StopIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (fakeIntegrationWorker) BackfillIntegrations(context.Context, []string) (map[string]any, error) {
	return map[string]any{}, nil
}

func TestListServicesHandlerReturnsItems(t *testing.T) {
	handler := NewListServicesHandler(fakeIntegrationWorker{})
	result, err := handler.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	items, ok := result["items"].([]map[string]any)
	if !ok || len(items) != 1 {
		t.Fatalf("unexpected integration result: %#v", result)
	}
}
