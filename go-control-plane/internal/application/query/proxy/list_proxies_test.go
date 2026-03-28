package proxyquery

import (
	"context"
	"testing"

	domainproxy "go-control-plane/internal/domain/proxy"
)

type fakeProxyRepository struct {
	items []domainproxy.Proxy
}

func (f fakeProxyRepository) List(context.Context, domainproxy.ListFilter) ([]domainproxy.Proxy, error) {
	return f.items, nil
}

func TestListProxiesHandlerReturnsItems(t *testing.T) {
	handler := NewListProxiesHandler(fakeProxyRepository{
		items: []domainproxy.Proxy{{ID: 1, URL: "http://1.1.1.1:8080", Region: "US", IsActive: true}},
	})

	items, err := handler.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(items) != 1 || items[0].URL != "http://1.1.1.1:8080" {
		t.Fatalf("unexpected items: %#v", items)
	}
}
