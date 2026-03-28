package proxycommand

import (
	"context"
	"testing"
)

type fakeProxyWriteRepository struct{}

func (fakeProxyWriteRepository) Add(context.Context, string, string) (int64, error)         { return 1, nil }
func (fakeProxyWriteRepository) BulkAdd(context.Context, []string, string) (int64, error)   { return 2, nil }
func (fakeProxyWriteRepository) Toggle(context.Context, int64) (bool, error)                { return false, nil }
func (fakeProxyWriteRepository) Delete(context.Context, int64) error                         { return nil }
func (fakeProxyWriteRepository) CheckAll(context.Context) error                              { return nil }

func TestProxyCommandHandlerAddAndToggle(t *testing.T) {
	handler := NewProxyCommandHandler(fakeProxyWriteRepository{})
	added, err := handler.Add(context.Background(), AddProxyCommand{URL: "http://1.1.1.1:8080", Region: "US"})
	if err != nil || added["id"] != int64(1) {
		t.Fatalf("unexpected add result: %#v err=%v", added, err)
	}
	toggled, err := handler.Toggle(context.Background(), ToggleProxyCommand{ProxyID: 1})
	if err != nil || toggled["is_active"] != false {
		t.Fatalf("unexpected toggle result: %#v err=%v", toggled, err)
	}
}
