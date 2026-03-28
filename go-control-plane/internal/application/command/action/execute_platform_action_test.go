package actioncommand

import (
	"context"
	"testing"

	domainaccount "go-control-plane/internal/domain/account"
	workerport "go-control-plane/internal/ports/outbound/worker"
)

type fakeActionAccountRepository struct {
	account domainaccount.Account
}

func (f fakeActionAccountRepository) GetByID(context.Context, int64) (domainaccount.Account, error) {
	return f.account, nil
}

type fakeExecuteActionWorkerClient struct {
	response workerport.ExecuteActionResponse
}

func (f fakeExecuteActionWorkerClient) ExecuteAction(context.Context, workerport.ExecuteActionRequest) (workerport.ExecuteActionResponse, error) {
	return f.response, nil
}

func (f fakeExecuteActionWorkerClient) Register(context.Context, workerport.RegisterRequest) (workerport.RegisterResponse, error) {
	return workerport.RegisterResponse{}, nil
}

func (f fakeExecuteActionWorkerClient) CheckAccount(context.Context, workerport.CheckAccountRequest) (workerport.CheckAccountResponse, error) {
	return workerport.CheckAccountResponse{}, nil
}
func (f fakeExecuteActionWorkerClient) GetSolverStatus(context.Context) (workerport.SolverStatusResponse, error) {
	return workerport.SolverStatusResponse{}, nil
}
func (f fakeExecuteActionWorkerClient) RestartSolver(context.Context) (map[string]any, error) { return map[string]any{}, nil }
func (f fakeExecuteActionWorkerClient) ListIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (f fakeExecuteActionWorkerClient) StartAllIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (f fakeExecuteActionWorkerClient) StopAllIntegrationServices(context.Context) (workerport.IntegrationServicesResponse, error) {
	return workerport.IntegrationServicesResponse{}, nil
}
func (f fakeExecuteActionWorkerClient) StartIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (f fakeExecuteActionWorkerClient) InstallIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (f fakeExecuteActionWorkerClient) StopIntegrationService(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (f fakeExecuteActionWorkerClient) BackfillIntegrations(context.Context, []string) (map[string]any, error) {
	return map[string]any{}, nil
}

func TestExecutePlatformActionReturnsWorkerResponse(t *testing.T) {
	handler := NewExecutePlatformActionHandler(
		fakeActionAccountRepository{account: domainaccount.Account{ID: 1, Platform: "dummy"}},
		fakeExecuteActionWorkerClient{response: workerport.ExecuteActionResponse{
			OK:   true,
			Data: map[string]any{"message": "done"},
		}},
	)

	result, err := handler.Handle(context.Background(), ExecutePlatformActionCommand{
		Platform:  "dummy",
		AccountID: 1,
		ActionID:  "sync_external",
		Params:    map[string]any{},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.OK || result.Data["message"] != "done" {
		t.Fatalf("unexpected result: %#v", result)
	}
}
