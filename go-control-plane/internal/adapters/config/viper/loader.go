package viperconfig

import (
	"strings"

	"github.com/spf13/viper"
)

type AppConfig struct {
	Server    ServerConfig    `mapstructure:"server"`
	Internal  InternalConfig  `mapstructure:"internal"`
	Log       LogConfig       `mapstructure:"log"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Worker    WorkerConfig    `mapstructure:"worker"`
	Platforms []PlatformEntry `mapstructure:"platforms"`
}

type ServerConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	PublicBaseURL   string `mapstructure:"public_base_url"`
	CallbackBaseURL string `mapstructure:"callback_base_url"`
}

type InternalConfig struct {
	CallbackToken string `mapstructure:"callback_token"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

type DatabaseConfig struct {
	URL string `mapstructure:"url"`
}

type WorkerConfig struct {
	BaseURL string `mapstructure:"base_url"`
}

type PlatformEntry struct {
	Name               string   `mapstructure:"name"`
	DisplayName        string   `mapstructure:"display_name"`
	Version            string   `mapstructure:"version"`
	SupportedExecutors []string `mapstructure:"supported_executors"`
	Available          bool     `mapstructure:"available"`
	AvailabilityReason string   `mapstructure:"availability_reason"`
}

func Load(configPath string) (AppConfig, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetEnvPrefix("AAR")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.public_base_url", "http://127.0.0.1:8080")
	v.SetDefault("server.callback_base_url", "")
	v.SetDefault("internal.callback_token", "")
	v.SetDefault("log.level", "info")
	v.SetDefault("database.url", "../account_manager.db")
	v.SetDefault("worker.base_url", "http://127.0.0.1:8000")
	v.SetDefault("platforms", []map[string]any{
		{
			"name":                "trae",
			"display_name":        "Trae.ai",
			"version":             "1.0.0",
			"supported_executors": []string{"protocol", "headless", "headed"},
			"available":           true,
			"availability_reason": "",
		},
		{
			"name":                "cursor",
			"display_name":        "Cursor",
			"version":             "1.0.0",
			"supported_executors": []string{"protocol"},
			"available":           true,
			"availability_reason": "",
		},
		{
			"name":                "kiro",
			"display_name":        "Kiro (AWS Builder ID)",
			"version":             "1.0.0",
			"supported_executors": []string{"protocol"},
			"available":           true,
			"availability_reason": "",
		},
		{
			"name":                "grok",
			"display_name":        "Grok",
			"version":             "1.0.0",
			"supported_executors": []string{"protocol"},
			"available":           true,
			"availability_reason": "",
		},
	})

	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return AppConfig{}, err
		}
	}

	var cfg AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return AppConfig{}, err
	}
	return cfg, nil
}
