package accountcommand

import (
	"context"

	domainaccount "go-control-plane/internal/domain/account"
	workerport "go-control-plane/internal/ports/outbound/worker"
)

type CheckAccountCommand struct {
	AccountID int64
}

type CheckAccountResult struct {
	OK    bool   `json:"ok"`
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

type CheckAccountHandler struct {
	repo   domainaccount.Repository
	worker workerport.Client
}

func NewCheckAccountHandler(repo domainaccount.Repository, worker workerport.Client) CheckAccountHandler {
	return CheckAccountHandler{repo: repo, worker: worker}
}

func (h CheckAccountHandler) Handle(ctx context.Context, cmd CheckAccountCommand) (CheckAccountResult, error) {
	account, err := h.repo.GetByID(ctx, cmd.AccountID)
	if err != nil {
		return CheckAccountResult{}, err
	}
	resp, err := h.worker.CheckAccount(ctx, workerport.CheckAccountRequest{
		Platform:  account.Platform,
		AccountID: cmd.AccountID,
	})
	if err != nil {
		return CheckAccountResult{}, err
	}
	return CheckAccountResult{OK: resp.OK, Valid: resp.Valid, Error: resp.Error}, nil
}
