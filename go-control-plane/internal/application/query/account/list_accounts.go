package accountquery

import (
	"context"
	"time"

	domainaccount "go-control-plane/internal/domain/account"
)

type ListAccountsFilter struct {
	Platform string
	Status   string
	Email    string
	Page     int
	PageSize int
}

type ListAccountsQuery = ListAccountsFilter

type AccountItem struct {
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

type ListAccountsResult struct {
	Total int           `json:"total"`
	Page  int           `json:"page"`
	Items []AccountItem `json:"items"`
}

type ListAccountsRepository interface {
	List(ctx context.Context, filter ListAccountsFilter) (int, []domainaccount.Account, error)
}

type ListAccountsHandler struct {
	repo ListAccountsRepository
}

func NewListAccountsHandler(repo ListAccountsRepository) ListAccountsHandler {
	return ListAccountsHandler{repo: repo}
}

func (h ListAccountsHandler) Handle(ctx context.Context, query ListAccountsQuery) (ListAccountsResult, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	total, accounts, err := h.repo.List(ctx, ListAccountsFilter{
		Platform: query.Platform,
		Status:   query.Status,
		Email:    query.Email,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return ListAccountsResult{}, err
	}
	items := make([]AccountItem, 0, len(accounts))
	for _, account := range accounts {
		items = append(items, AccountItem{
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
		})
	}
	return ListAccountsResult{Total: total, Page: page, Items: items}, nil
}
