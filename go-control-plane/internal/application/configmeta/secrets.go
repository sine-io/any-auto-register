package configmeta

const MaskedSecretValue = "********"

var secretConfigKeys = map[string]struct{}{
	"laoudo_auth":                 {},
	"yescaptcha_key":              {},
	"twocaptcha_key":              {},
	"duckmail_bearer":             {},
	"freemail_admin_token":        {},
	"freemail_password":           {},
	"cfworker_admin_token":        {},
	"cpa_api_key":                 {},
	"team_manager_key":            {},
	"cliproxyapi_management_key":  {},
	"grok2api_app_key":            {},
}

func IsSecretKey(key string) bool {
	_, ok := secretConfigKeys[key]
	return ok
}

func MaskValue(key string, value string) string {
	if value == "" {
		return ""
	}
	if IsSecretKey(key) {
		return MaskedSecretValue
	}
	return value
}
