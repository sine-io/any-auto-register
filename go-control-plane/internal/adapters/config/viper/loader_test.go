package viperconfig

import "testing"

func TestLoadReturnsDefaultServerConfig(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Server.Host != "0.0.0.0" {
		t.Fatalf("expected default host 0.0.0.0, got %s", cfg.Server.Host)
	}

	if cfg.Server.Port != 8080 {
		t.Fatalf("expected default port 8080, got %d", cfg.Server.Port)
	}

	if cfg.Database.URL != "../account_manager.db" {
		t.Fatalf("expected default database url ../account_manager.db, got %s", cfg.Database.URL)
	}

	if len(cfg.Platforms) == 0 {
		t.Fatal("expected default platform manifest to be populated")
	}
}
