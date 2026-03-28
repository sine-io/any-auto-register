package platformquery

import (
	"context"

	domainplatform "go-control-plane/internal/domain/platform"
)

type PlatformItem struct {
	Name               string   `json:"name"`
	DisplayName        string   `json:"display_name"`
	Version            string   `json:"version"`
	SupportedExecutors []string `json:"supported_executors"`
	Available          bool     `json:"available"`
	AvailabilityReason string   `json:"availability_reason"`
}

type Result struct {
	Items []PlatformItem `json:"items"`
}

type Handler struct {
	repo domainplatform.Repository
}

func NewHandler(repo domainplatform.Repository) Handler {
	return Handler{repo: repo}
}

func (h Handler) Handle(ctx context.Context) (Result, error) {
	platforms, err := h.repo.List(ctx)
	if err != nil {
		return Result{}, err
	}

	items := make([]PlatformItem, 0, len(platforms))
	for _, platform := range platforms {
		items = append(items, PlatformItem{
			Name:               platform.Name,
			DisplayName:        platform.DisplayName,
			Version:            platform.Version,
			SupportedExecutors: platform.SupportedExecutors,
			Available:          platform.Available,
			AvailabilityReason: platform.AvailabilityReason,
		})
	}
	return Result{Items: items}, nil
}
