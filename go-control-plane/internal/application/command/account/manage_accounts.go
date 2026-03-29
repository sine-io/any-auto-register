package accountcommand

import (
	"context"
	"errors"
	"time"

	domainaccount "go-control-plane/internal/domain/account"
)

var ErrAccountNotFound = errors.New("account not found")

type CreateAccountCommand struct {
	Platform     string `json:"platform"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	Status       string `json:"status"`
	Token        string `json:"token"`
	TrialEndTime int64  `json:"trial_end_time"`
	CashierURL   string `json:"cashier_url"`
}

type UpdateAccountCommand struct {
	AccountID    int64   `json:"-"`
	Status       *string `json:"status,omitempty"`
	Token        *string `json:"token,omitempty"`
	TrialEndTime *int64  `json:"trial_end_time,omitempty"`
	CashierURL   *string `json:"cashier_url,omitempty"`
}

type DeleteAccountCommand struct {
	AccountID int64
}

type BatchDeleteAccountsCommand struct {
	IDs []int64 `json:"ids"`
}

type ImportAccountsCommand struct {
	Platform string   `json:"platform"`
	Lines    []string `json:"lines"`
}

type AccountMutationResult struct {
	ID           int64     `json:"id"`
	Platform     string    `json:"platform"`
	Email        string    `json:"email"`
	Password     string    `json:"password"`
	Region       string    `json:"region"`
	Token        string    `json:"token"`
	Status       string    `json:"status"`
	CashierURL   string    `json:"cashier_url"`
	TrialEndTime int64     `json:"trial_end_time"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type BatchDeleteAccountsResult struct {
	Deleted        int     `json:"deleted"`
	NotFound       []int64 `json:"not_found"`
	TotalRequested int     `json:"total_requested"`
}

type ImportAccountsResult struct {
	Created int `json:"created"`
}

type ManagementRepository interface {
	CreateAccount(context.Context, CreateAccountCommand) (domainaccount.Account, error)
	UpdateAccount(context.Context, UpdateAccountCommand) (domainaccount.Account, error)
	DeleteAccount(context.Context, int64) error
	BatchDeleteAccounts(context.Context, []int64) (int, []int64, error)
	ImportAccounts(context.Context, string, []string) (int, error)
}

type CreateAccountHandler struct {
	repo ManagementRepository
}

func NewCreateAccountHandler(repo ManagementRepository) CreateAccountHandler {
	return CreateAccountHandler{repo: repo}
}

func (h CreateAccountHandler) Handle(ctx context.Context, cmd CreateAccountCommand) (AccountMutationResult, error) {
	account, err := h.repo.CreateAccount(ctx, cmd)
	if err != nil {
		return AccountMutationResult{}, err
	}
	return toAccountMutationResult(account), nil
}

type UpdateAccountHandler struct {
	repo ManagementRepository
}

func NewUpdateAccountHandler(repo ManagementRepository) UpdateAccountHandler {
	return UpdateAccountHandler{repo: repo}
}

func (h UpdateAccountHandler) Handle(ctx context.Context, cmd UpdateAccountCommand) (AccountMutationResult, error) {
	account, err := h.repo.UpdateAccount(ctx, cmd)
	if err != nil {
		return AccountMutationResult{}, err
	}
	return toAccountMutationResult(account), nil
}

type DeleteAccountHandler struct {
	repo ManagementRepository
}

func NewDeleteAccountHandler(repo ManagementRepository) DeleteAccountHandler {
	return DeleteAccountHandler{repo: repo}
}

func (h DeleteAccountHandler) Handle(ctx context.Context, cmd DeleteAccountCommand) (map[string]any, error) {
	if err := h.repo.DeleteAccount(ctx, cmd.AccountID); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

type BatchDeleteAccountsHandler struct {
	repo ManagementRepository
}

func NewBatchDeleteAccountsHandler(repo ManagementRepository) BatchDeleteAccountsHandler {
	return BatchDeleteAccountsHandler{repo: repo}
}

func (h BatchDeleteAccountsHandler) Handle(ctx context.Context, cmd BatchDeleteAccountsCommand) (BatchDeleteAccountsResult, error) {
	deleted, notFound, err := h.repo.BatchDeleteAccounts(ctx, cmd.IDs)
	if err != nil {
		return BatchDeleteAccountsResult{}, err
	}
	return BatchDeleteAccountsResult{
		Deleted:        deleted,
		NotFound:       notFound,
		TotalRequested: len(cmd.IDs),
	}, nil
}

type ImportAccountsHandler struct {
	repo ManagementRepository
}

func NewImportAccountsHandler(repo ManagementRepository) ImportAccountsHandler {
	return ImportAccountsHandler{repo: repo}
}

func (h ImportAccountsHandler) Handle(ctx context.Context, cmd ImportAccountsCommand) (ImportAccountsResult, error) {
	created, err := h.repo.ImportAccounts(ctx, cmd.Platform, cmd.Lines)
	if err != nil {
		return ImportAccountsResult{}, err
	}
	return ImportAccountsResult{Created: created}, nil
}

func toAccountMutationResult(account domainaccount.Account) AccountMutationResult {
	return AccountMutationResult{
		ID:           account.ID,
		Platform:     account.Platform,
		Email:        account.Email,
		Password:     account.Password,
		Region:       account.Region,
		Token:        account.Token,
		Status:       account.Status,
		CashierURL:   account.CashierURL,
		TrialEndTime: account.TrialEndTime,
		CreatedAt:    account.CreatedAt,
		UpdatedAt:    account.UpdatedAt,
	}
}
