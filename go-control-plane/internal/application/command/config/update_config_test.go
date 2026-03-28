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
