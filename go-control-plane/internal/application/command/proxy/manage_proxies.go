package proxycommand

import "context"

type AddProxyCommand struct {
	URL    string `json:"url"`
	Region string `json:"region"`
}

type BulkAddProxiesCommand struct {
	Proxies []string `json:"proxies"`
	Region  string   `json:"region"`
}

type ToggleProxyCommand struct {
	ProxyID int64
}

type DeleteProxyCommand struct {
	ProxyID int64
}

type CheckProxiesCommand struct{}

type Repository interface {
	Add(ctx context.Context, url, region string) (int64, error)
	BulkAdd(ctx context.Context, proxies []string, region string) (int64, error)
	Toggle(ctx context.Context, proxyID int64) (bool, error)
	Delete(ctx context.Context, proxyID int64) error
	CheckAll(ctx context.Context) error
}

type Handler struct {
	repo Repository
}

func NewProxyCommandHandler(repo Repository) Handler {
	return Handler{repo: repo}
}

func (h Handler) Add(ctx context.Context, cmd AddProxyCommand) (map[string]any, error) {
	id, err := h.repo.Add(ctx, cmd.URL, cmd.Region)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": id}, nil
}

func (h Handler) BulkAdd(ctx context.Context, cmd BulkAddProxiesCommand) (map[string]any, error) {
	added, err := h.repo.BulkAdd(ctx, cmd.Proxies, cmd.Region)
	if err != nil {
		return nil, err
	}
	return map[string]any{"added": added}, nil
}

func (h Handler) Toggle(ctx context.Context, cmd ToggleProxyCommand) (map[string]any, error) {
	isActive, err := h.repo.Toggle(ctx, cmd.ProxyID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"is_active": isActive}, nil
}

func (h Handler) Delete(ctx context.Context, cmd DeleteProxyCommand) (map[string]any, error) {
	if err := h.repo.Delete(ctx, cmd.ProxyID); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

func (h Handler) Check(ctx context.Context, cmd CheckProxiesCommand) (map[string]any, error) {
	go h.repo.CheckAll(context.Background())
	return map[string]any{"message": "检测任务已启动"}, nil
}
