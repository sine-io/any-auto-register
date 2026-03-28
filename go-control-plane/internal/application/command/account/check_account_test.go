package accountcommand

import (
	"context"
	"testing"

	domainaccount "go-control-plane/internal/domain/account"
	workerport "go-control-plane/internal/ports/outbound/worker"
)

type fakeAccountLookupRepository struct {
	account domainaccount.Account
}

func (f fakeAccountLookupRepository) GetByID(context.Context, int64) (domainaccount.Account, error) {
	return f.account, nil
}

type fakeCheckWorkerClient struct {
	response workerport.CheckAccountResponse
}

func (f fakeCheckWorkerClient) CheckAccount(context.Context, workerport.CheckAccountRequest) (workerport.CheckAccountResponse, error) {
	return f.response, nil
}

func (f fakeCheckWorkerClient) Register(context.Context, workerport.RegisterRequest) (workerport.RegisterResponse, error) {
	return workerport.RegisterResponse{}, nil
}

func (f fakeCheckWorkerClient) ExecuteAction(context.Context, workerport.ExecuteActionRequest) (workerport.ExecuteActionResponse, error) {
	return workerport.ExecuteActionResponse{}, nil
}

func TestCheckAccountUsesStoredPlatform(t *testing.T) {
	handler := NewCheckAccountHandler(
		fakeAccountLookupRepository{account: domainaccount.Account{ID: 1, Platform: "dummy"}},
		fakeCheckWorkerClient{response: workerport.CheckAccountResponse{OK: true, Valid: true}},
	)

	result, err := handler.Handle(context.Background(), CheckAccountCommand{AccountID: 1})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.OK || !result.Valid {
		t.Fatalf("unexpected result: %#v", result)
	}
}
