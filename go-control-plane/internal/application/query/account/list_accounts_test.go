package accountquery

import (
	"context"
	"testing"

	domainaccount "go-control-plane/internal/domain/account"
)

type fakeAccountListRepository struct {
	total int
	items []domainaccount.Account
}

func (f fakeAccountListRepository) List(context.Context, ListAccountsFilter) (int, []domainaccount.Account, error) {
	return f.total, f.items, nil
}

func TestListAccountsHandlerReturnsPaginatedAccounts(t *testing.T) {
	handler := NewListAccountsHandler(fakeAccountListRepository{
		total: 2,
		items: []domainaccount.Account{
			{ID: 1, Platform: "dummy", Email: "user@example.com", Status: "registered"},
		},
	})

	result, err := handler.Handle(context.Background(), ListAccountsQuery{Page: 2, PageSize: 1})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Total != 2 || result.Page != 2 {
		t.Fatalf("unexpected pagination result: %#v", result)
	}
	if len(result.Items) != 1 || result.Items[0].Email != "user@example.com" {
		t.Fatalf("unexpected account items: %#v", result.Items)
	}
}
