package accountquery

import (
	"context"
	"testing"
)

type fakeStatsRepository struct {
	result DashboardStatsResult
}

func (f fakeStatsRepository) GetDashboardStats(context.Context) (DashboardStatsResult, error) {
	return f.result, nil
}

func TestGetDashboardStatsReturnsAggregates(t *testing.T) {
	handler := NewDashboardStatsHandler(fakeStatsRepository{
		result: DashboardStatsResult{
			Total:      3,
			ByPlatform: map[string]int64{"trae": 2},
			ByStatus:   map[string]int64{"registered": 3},
		},
	})

	result, err := handler.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Total != 3 || result.ByPlatform["trae"] != 2 {
		t.Fatalf("unexpected stats: %#v", result)
	}
}
