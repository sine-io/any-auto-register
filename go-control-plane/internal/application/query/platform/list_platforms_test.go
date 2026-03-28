package platformquery

import (
	"context"
	"testing"

	domainplatform "go-control-plane/internal/domain/platform"
)

type fakePlatformRepository struct {
	items []domainplatform.Platform
}

func (f fakePlatformRepository) List(context.Context) ([]domainplatform.Platform, error) {
	return f.items, nil
}

func TestHandlerReturnsPlatforms(t *testing.T) {
	handler := NewHandler(fakePlatformRepository{
		items: []domainplatform.Platform{
			{Name: "trae", DisplayName: "Trae.ai", SupportedExecutors: []string{"protocol", "headed"}},
		},
	})

	result, err := handler.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Items) != 1 || result.Items[0].Name != "trae" {
		t.Fatalf("unexpected result: %#v", result.Items)
	}
}
