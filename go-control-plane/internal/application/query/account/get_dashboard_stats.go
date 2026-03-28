package accountquery

import "context"

type DashboardStatsResult struct {
	Total      int64            `json:"total"`
	ByPlatform map[string]int64 `json:"by_platform"`
	ByStatus   map[string]int64 `json:"by_status"`
}

type DashboardStatsRepository interface {
	GetDashboardStats(context.Context) (DashboardStatsResult, error)
}

type DashboardStatsHandler struct {
	repo DashboardStatsRepository
}

func NewDashboardStatsHandler(repo DashboardStatsRepository) DashboardStatsHandler {
	return DashboardStatsHandler{repo: repo}
}

func (h DashboardStatsHandler) Handle(ctx context.Context) (DashboardStatsResult, error) {
	return h.repo.GetDashboardStats(ctx)
}
