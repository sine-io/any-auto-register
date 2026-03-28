package integrationcommand

import (
	"context"
	"testing"

	workerport "go-control-plane/internal/ports/outbound/worker"
)

type fakeIntegrationCommandWorker struct{}

func (fakeIntegrationCommandWorker) Register(context.Context, workerport.RegisterRequest) (workerport.RegisterResponse, error) {
	return workerport.RegisterResponse{}, nil
}
func (fakeIntegrationCommandWorker) CheckAccount(context.Context, workerport.CheckAccountRequest) (workerport.CheckAccountResponse, error) {
	return workerport.CheckAccountResponse{}, nil
}
func (fakeIntegrationCommandWorker) ExecuteAction(context.Context, workerport.ExecuteActionRequest) (workerport.ExecuteActionResponse, error) {
	return workerport.ExecuteActionResponse{}, nil
}
func (fakeIntegrationCommandWorker) GetSolverStatus(context.Context) (workerport.SolverStatusResponse, error) {
	return workerport.SolverStatusResponse{}, nil
}
func (fakeIntegrationCommandWorker) RestartSolver(context.Context) (map[string]any, error) { return map[string]any{}, nil }
func (fakeIntegrationCommandWorker) ListIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}

func (fakeIntegrationCommandWorker) StartAllIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{Items: []map[string]any{{"name": "grok2api"}}}, nil
}
func (fakeIntegrationCommandWorker) StopAllIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{Items: []map[string]any{}}, nil
}
func (fakeIntegrationCommandWorker) StartIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{"name": "grok2api"}, nil
}
func (fakeIntegrationCommandWorker) InstallIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{"name": "grok2api"}, nil
}
func (fakeIntegrationCommandWorker) StopIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{"name": "grok2api"}, nil
}
func (fakeIntegrationCommandWorker) BackfillIntegrations(context.Context, []string) (map[string]any, error) {
	return map[string]any{"total": 1}, nil
}

func TestIntegrationHandlerStartAll(t *testing.T) {
	handler := NewHandler(fakeIntegrationCommandWorker{})
	result, err := handler.StartAll(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := result["items"]; !ok {
		t.Fatalf("unexpected result: %#v", result)
	}
}
