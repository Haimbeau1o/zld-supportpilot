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
