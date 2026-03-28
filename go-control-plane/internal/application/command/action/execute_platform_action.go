package actioncommand

import (
	"context"

	domainaccount "go-control-plane/internal/domain/account"
	workerport "go-control-plane/internal/ports/outbound/worker"
)

type ExecutePlatformActionCommand struct {
	Platform  string         `json:"platform"`
	AccountID int64          `json:"account_id"`
	ActionID  string         `json:"action_id"`
	Params    map[string]any `json:"params"`
}

type ExecutePlatformActionResult struct {
	OK    bool           `json:"ok"`
	Data  map[string]any `json:"data,omitempty"`
	Error string         `json:"error,omitempty"`
}

type ExecutePlatformActionHandler struct {
	repo   domainaccount.Repository
	worker workerport.Client
}

func NewExecutePlatformActionHandler(repo domainaccount.Repository, worker workerport.Client) ExecutePlatformActionHandler {
	return ExecutePlatformActionHandler{repo: repo, worker: worker}
}

func (h ExecutePlatformActionHandler) Handle(ctx context.Context, cmd ExecutePlatformActionCommand) (ExecutePlatformActionResult, error) {
	account, err := h.repo.GetByID(ctx, cmd.AccountID)
	if err != nil {
		return ExecutePlatformActionResult{}, err
	}
	platform := cmd.Platform
	if platform == "" {
		platform = account.Platform
	}
	resp, err := h.worker.ExecuteAction(ctx, workerport.ExecuteActionRequest{
		Platform:  platform,
		AccountID: cmd.AccountID,
		ActionID:  cmd.ActionID,
		Params:    cmd.Params,
	})
	if err != nil {
		return ExecutePlatformActionResult{}, err
	}
	return ExecutePlatformActionResult{OK: resp.OK, Data: resp.Data, Error: resp.Error}, nil
}
