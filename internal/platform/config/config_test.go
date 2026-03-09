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

func TestLoadProvidesDefaultHTTPRateLimitConfig(t *testing.T) {
	t.Setenv("HTTP_RATE_LIMIT_ENABLED", "")
	t.Setenv("HTTP_RATE_LIMIT_RPS", "")
	t.Setenv("HTTP_RATE_LIMIT_BURST", "")

	cfg := Load()

	if !cfg.HTTPRateLimitEnabled {
		t.Fatalf("expected default rate limit to be enabled")
	}

	if cfg.HTTPRateLimitRPS != 5 {
		t.Fatalf("expected default rate limit rps 5, got %v", cfg.HTTPRateLimitRPS)
	}

	if cfg.HTTPRateLimitBurst != 10 {
		t.Fatalf("expected default rate limit burst 10, got %d", cfg.HTTPRateLimitBurst)
	}
}

func TestLoadReadsHTTPRateLimitConfigFromEnvironment(t *testing.T) {
	t.Setenv("HTTP_RATE_LIMIT_ENABLED", "false")
	t.Setenv("HTTP_RATE_LIMIT_RPS", "2.5")
	t.Setenv("HTTP_RATE_LIMIT_BURST", "6")

	cfg := Load()

	if cfg.HTTPRateLimitEnabled {
		t.Fatalf("expected rate limit enabled to be false")
	}

	if cfg.HTTPRateLimitRPS != 2.5 {
		t.Fatalf("expected rate limit rps 2.5, got %v", cfg.HTTPRateLimitRPS)
	}

	if cfg.HTTPRateLimitBurst != 6 {
		t.Fatalf("expected rate limit burst 6, got %d", cfg.HTTPRateLimitBurst)
	}
}

func TestLoadFallsBackWhenHTTPRateLimitConfigInvalid(t *testing.T) {
	t.Setenv("HTTP_RATE_LIMIT_ENABLED", "not-a-bool")
	t.Setenv("HTTP_RATE_LIMIT_RPS", "-1")
	t.Setenv("HTTP_RATE_LIMIT_BURST", "0")

	cfg := Load()

	if !cfg.HTTPRateLimitEnabled {
		t.Fatalf("expected invalid bool to fall back to default true")
	}

	if cfg.HTTPRateLimitRPS != 5 {
		t.Fatalf("expected invalid rate limit rps to fall back to 5, got %v", cfg.HTTPRateLimitRPS)
	}

	if cfg.HTTPRateLimitBurst != 10 {
		t.Fatalf("expected invalid rate limit burst to fall back to 10, got %d", cfg.HTTPRateLimitBurst)
	}
}

func TestLoadProvidesDefaultPersistenceConfig(t *testing.T) {
	t.Setenv("APP_PERSISTENCE_MODE", "")
	t.Setenv("POSTGRES_DSN", "")
	t.Setenv("DOCUMENT_STORAGE_MODE", "")
	t.Setenv("DOCUMENT_STORAGE_ROOT", "")

	cfg := Load()

	if cfg.PersistenceMode != "memory" {
		t.Fatalf("expected default persistence mode memory, got %q", cfg.PersistenceMode)
	}

	if cfg.PostgresDSN != "" {
		t.Fatalf("expected default postgres dsn empty, got %q", cfg.PostgresDSN)
	}

	if cfg.DocumentStorageMode != "memory" {
		t.Fatalf("expected default document storage mode memory, got %q", cfg.DocumentStorageMode)
	}

	if cfg.DocumentStorageRoot != "data/documents" {
		t.Fatalf("expected default document storage root %q, got %q", "data/documents", cfg.DocumentStorageRoot)
	}
}

func TestLoadReadsPersistenceConfigFromEnvironment(t *testing.T) {
	t.Setenv("APP_PERSISTENCE_MODE", "postgres")
	t.Setenv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/supportpilot?sslmode=disable")
	t.Setenv("DOCUMENT_STORAGE_MODE", "filesystem")
	t.Setenv("DOCUMENT_STORAGE_ROOT", "/tmp/supportpilot-documents")

	cfg := Load()

	if cfg.PersistenceMode != "postgres" {
		t.Fatalf("expected persistence mode postgres, got %q", cfg.PersistenceMode)
	}

	if cfg.PostgresDSN != "postgres://postgres:postgres@localhost:5432/supportpilot?sslmode=disable" {
		t.Fatalf("expected postgres dsn from env, got %q", cfg.PostgresDSN)
	}

	if cfg.DocumentStorageMode != "filesystem" {
		t.Fatalf("expected document storage mode filesystem, got %q", cfg.DocumentStorageMode)
	}

	if cfg.DocumentStorageRoot != "/tmp/supportpilot-documents" {
		t.Fatalf("expected document storage root from env, got %q", cfg.DocumentStorageRoot)
	}
}
