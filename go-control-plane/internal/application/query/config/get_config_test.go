package configquery

import (
	"context"
	"testing"
)

type fakeConfigRepository struct {
	items map[string]string
}

func (f fakeConfigRepository) GetAll(context.Context, []string) (map[string]string, error) {
	return f.items, nil
}

func TestGetConfigReturnsKnownKeys(t *testing.T) {
	handler := NewGetConfigHandler(fakeConfigRepository{items: map[string]string{
		"mail_provider":   "moemail",
		"yescaptcha_key":  "secret-key",
		"laoudo_email":    "user@example.com",
	}})
	result, err := handler.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result["mail_provider"] != "moemail" {
		t.Fatalf("unexpected config: %#v", result)
	}
	if result["yescaptcha_key"] != MaskedSecretValue {
		t.Fatalf("expected secret values to be masked, got %#v", result)
	}
	if result["laoudo_email"] != "user@example.com" {
		t.Fatalf("expected non-secret values to remain visible, got %#v", result)
	}
	if _, ok := result["yescaptcha_key"]; !ok {
		t.Fatalf("expected known keys to exist, got %#v", result)
	}
}
