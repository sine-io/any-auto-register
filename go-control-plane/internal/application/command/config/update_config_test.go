package configcommand

import (
	"context"
	"testing"
)

type fakeConfigWriteRepository struct {
	data map[string]string
}

func (f *fakeConfigWriteRepository) SetMany(_ context.Context, data map[string]string) error {
	f.data = data
	return nil
}

func TestUpdateConfigFiltersUnknownKeys(t *testing.T) {
	repo := &fakeConfigWriteRepository{}
	handler := NewUpdateConfigHandler(repo)

	result, err := handler.Handle(context.Background(), UpdateConfigCommand{
		Data: map[string]string{
			"mail_provider": "moemail",
			"unknown":       "value",
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.OK || len(result.Updated) != 1 || result.Updated[0] != "mail_provider" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if _, ok := repo.data["unknown"]; ok {
		t.Fatalf("expected unknown key to be filtered, got %#v", repo.data)
	}
}

func TestUpdateConfigSkipsMaskedSecretPlaceholder(t *testing.T) {
	repo := &fakeConfigWriteRepository{}
	handler := NewUpdateConfigHandler(repo)

	result, err := handler.Handle(context.Background(), UpdateConfigCommand{
		Data: map[string]string{
			"yescaptcha_key": MaskedSecretValue,
			"mail_provider":  "duckmail",
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.OK {
		t.Fatalf("expected successful update result, got %#v", result)
	}
	if _, ok := repo.data["yescaptcha_key"]; ok {
		t.Fatalf("expected masked secret placeholder to be skipped, got %#v", repo.data)
	}
	if repo.data["mail_provider"] != "duckmail" {
		t.Fatalf("expected non-secret field to be updated, got %#v", repo.data)
	}
	if len(result.Updated) != 1 || result.Updated[0] != "mail_provider" {
		t.Fatalf("expected only mail_provider to be marked updated, got %#v", result)
	}
}
