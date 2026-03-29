package actionquery

import (
	"context"
	"testing"

	workerport "go-control-plane/internal/ports/outbound/worker"
)

type fakeActionListWorker struct{}

func (fakeActionListWorker) Register(context.Context, workerport.RegisterRequest) (workerport.RegisterResponse, error) {
	return workerport.RegisterResponse{}, nil
}
func (fakeActionListWorker) CheckAccount(context.Context, workerport.CheckAccountRequest) (workerport.CheckAccountResponse, error) {
	return workerport.CheckAccountResponse{}, nil
}
func (fakeActionListWorker) ExecuteAction(context.Context, workerport.ExecuteActionRequest) (workerport.ExecuteActionResponse, error) {
	return workerport.ExecuteActionResponse{}, nil
}
func (fakeActionListWorker) ListActions(context.Context, string) (workerport.ListActionsResponse, error) {
	return workerport.ListActionsResponse{
		Actions: []map[string]any{{"id": "sync_external", "available": true}},
	}, nil
}
func (fakeActionListWorker) GetSolverStatus(context.Context) (workerport.SolverStatusResponse, error) {
	return workerport.SolverStatusResponse{}, nil
}
func (fakeActionListWorker) RestartSolver(context.Context) (map[string]any, error) {
	return map[string]any{}, nil
}
func (fakeActionListWorker) ListIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (fakeActionListWorker) StartAllIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (fakeActionListWorker) StopAllIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (fakeActionListWorker) StartIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (fakeActionListWorker) InstallIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (fakeActionListWorker) StopIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (fakeActionListWorker) BackfillIntegrations(context.Context, []string) (map[string]any, error) {
	return map[string]any{}, nil
}

func TestListActionsReturnsWorkerMetadata(t *testing.T) {
	handler := NewListActionsHandler(fakeActionListWorker{})

	result, err := handler.Handle(context.Background(), "dummy")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Actions) != 1 || result.Actions[0]["id"] != "sync_external" {
		t.Fatalf("unexpected actions result: %#v", result)
	}
}
