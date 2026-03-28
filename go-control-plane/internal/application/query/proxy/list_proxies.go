package proxyquery

import (
	"context"

	domainproxy "go-control-plane/internal/domain/proxy"
)

type ProxyItem struct {
	ID           int64   `json:"id"`
	URL          string  `json:"url"`
	Region       string  `json:"region"`
	SuccessCount int64   `json:"success_count"`
	FailCount    int64   `json:"fail_count"`
	IsActive     bool    `json:"is_active"`
	LastChecked  *string `json:"last_checked"`
}

type Handler struct {
	repo domainproxy.Repository
}

func NewListProxiesHandler(repo domainproxy.Repository) Handler {
	return Handler{repo: repo}
}

func (h Handler) Handle(ctx context.Context) ([]ProxyItem, error) {
	items, err := h.repo.List(ctx, domainproxy.ListFilter{})
	if err != nil {
		return nil, err
	}
	result := make([]ProxyItem, 0, len(items))
	for _, item := range items {
		var lastChecked *string
		if item.LastChecked != nil {
			value := item.LastChecked.Format("2006-01-02T15:04:05Z07:00")
			lastChecked = &value
		}
		result = append(result, ProxyItem{
			ID:           item.ID,
			URL:          item.URL,
			Region:       item.Region,
			SuccessCount: item.SuccessCount,
			FailCount:    item.FailCount,
			IsActive:     item.IsActive,
			LastChecked:  lastChecked,
		})
	}
	return result, nil
}
