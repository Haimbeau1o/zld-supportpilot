package config

import (
	"testing"
	"time"
)

func TestLoadProvidesDefaultAuthConfig(t *testing.T) {
	t.Setenv("AUTH_SIGNING_KEY", "")
	t.Setenv("AUTH_TOKEN_TTL_SECONDS", "")

	cfg := Load()

	if cfg.AuthSigningKey == "" {
		t.Fatalf("expected default auth signing key to be populated")
	}

	if cfg.AuthTokenTTL != time.Hour {
		t.Fatalf("expected default auth token ttl %v, got %v", time.Hour, cfg.AuthTokenTTL)
	}
}

func TestLoadReadsAuthConfigFromEnvironment(t *testing.T) {
	t.Setenv("AUTH_SIGNING_KEY", "custom-signing-key")
	t.Setenv("AUTH_TOKEN_TTL_SECONDS", "7200")

	cfg := Load()

	if cfg.AuthSigningKey != "custom-signing-key" {
		t.Fatalf("expected auth signing key from env, got %q", cfg.AuthSigningKey)
	}

	if cfg.AuthTokenTTL != 2*time.Hour {
		t.Fatalf("expected auth token ttl %v, got %v", 2*time.Hour, cfg.AuthTokenTTL)
	}
}
