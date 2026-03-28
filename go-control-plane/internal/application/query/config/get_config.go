package configquery

import "context"

var KnownKeys = []string{
	"laoudo_auth", "laoudo_email", "laoudo_account_id",
	"yescaptcha_key", "twocaptcha_key",
	"default_executor", "default_captcha_solver",
	"duckmail_api_url", "duckmail_provider_url", "duckmail_bearer",
	"freemail_api_url", "freemail_admin_token", "freemail_username", "freemail_password",
	"moemail_api_url",
	"mail_provider",
	"cfworker_api_url", "cfworker_admin_token", "cfworker_domain", "cfworker_fingerprint",
	"cpa_api_url", "cpa_api_key",
	"team_manager_url", "team_manager_key",
	"cliproxyapi_management_key",
	"grok2api_url", "grok2api_app_key", "grok2api_pool", "grok2api_quota",
	"kiro_manager_path", "kiro_manager_exe",
}

type Repository interface {
	GetAll(ctx context.Context, keys []string) (map[string]string, error)
}

type GetConfigHandler struct {
	repo Repository
}

func NewGetConfigHandler(repo Repository) GetConfigHandler {
	return GetConfigHandler{repo: repo}
}

func (h GetConfigHandler) Handle(ctx context.Context) (map[string]string, error) {
	items, err := h.repo.GetAll(ctx, KnownKeys)
	if err != nil {
		return nil, err
	}
	for _, key := range KnownKeys {
		if _, ok := items[key]; !ok {
			items[key] = ""
		}
	}
	return items, nil
}
