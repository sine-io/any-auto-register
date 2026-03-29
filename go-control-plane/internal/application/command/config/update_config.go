package configcommand

import (
	"context"

	configmeta "go-control-plane/internal/application/configmeta"
)

const MaskedSecretValue = configmeta.MaskedSecretValue

var allowedConfigKeys = map[string]struct{}{
	"laoudo_auth": {}, "laoudo_email": {}, "laoudo_account_id": {},
	"yescaptcha_key": {}, "twocaptcha_key": {},
	"default_executor": {}, "default_captcha_solver": {},
	"duckmail_api_url": {}, "duckmail_provider_url": {}, "duckmail_bearer": {},
	"freemail_api_url": {}, "freemail_admin_token": {}, "freemail_username": {}, "freemail_password": {},
	"moemail_api_url": {},
	"mail_provider": {},
	"cfworker_api_url": {}, "cfworker_admin_token": {}, "cfworker_domain": {}, "cfworker_fingerprint": {},
	"cpa_api_url": {}, "cpa_api_key": {},
	"team_manager_url": {}, "team_manager_key": {},
	"cliproxyapi_management_key": {},
	"grok2api_url": {}, "grok2api_app_key": {}, "grok2api_pool": {}, "grok2api_quota": {},
	"kiro_manager_path": {}, "kiro_manager_exe": {},
}

type UpdateConfigCommand struct {
	Data map[string]string `json:"data"`
}

type UpdateConfigResult struct {
	OK      bool     `json:"ok"`
	Updated []string `json:"updated"`
}

type Repository interface {
	SetMany(ctx context.Context, data map[string]string) error
}

type UpdateConfigHandler struct {
	repo Repository
}

func NewUpdateConfigHandler(repo Repository) UpdateConfigHandler {
	return UpdateConfigHandler{repo: repo}
}

func (h UpdateConfigHandler) Handle(ctx context.Context, cmd UpdateConfigCommand) (UpdateConfigResult, error) {
	safe := make(map[string]string)
	updated := make([]string, 0)
	for key, value := range cmd.Data {
		if _, ok := allowedConfigKeys[key]; !ok {
			continue
		}
		if configmeta.IsSecretKey(key) && value == MaskedSecretValue {
			continue
		}
		safe[key] = value
		updated = append(updated, key)
	}
	if err := h.repo.SetMany(ctx, safe); err != nil {
		return UpdateConfigResult{}, err
	}
	return UpdateConfigResult{OK: true, Updated: updated}, nil
}
