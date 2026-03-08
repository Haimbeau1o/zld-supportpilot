package http

import (
	"bytes"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/config"
)

func TestRateLimitMiddlewareRejectsRequestsBeyondBurst(t *testing.T) {
	metrics := NewHTTPMetrics()
	middleware := NewRateLimitMiddleware(config.Config{
		AppName:              "zld-supportpilot-test",
		AppEnv:               "test",
		HTTPAddr:             ":0",
		HTTPRateLimitEnabled: true,
		HTTPRateLimitRPS:     1,
		HTTPRateLimitBurst:   1,
	}, metrics)
	currentTime := time.Unix(1700000000, 0)
	middleware.now = func() time.Time {
		return currentTime
	}

	handler := middleware.Wrap("POST /api/v1/auth/login", clientAddressRateLimitKey, stdhttp.HandlerFunc(func(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
		writeJSON(writer, stdhttp.StatusOK, map[string]string{"status": "ok"})
	}))

	firstRequest := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"email":"alice@example.com","password":"secret123"}`))
	firstRequest.RemoteAddr = "10.0.0.1:1234"
	firstRecorder := httptest.NewRecorder()
	handler.ServeHTTP(firstRecorder, firstRequest)

	if firstRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected first request status %d, got %d", stdhttp.StatusOK, firstRecorder.Code)
	}

	secondRequest := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"email":"alice@example.com","password":"secret123"}`))
	secondRequest.RemoteAddr = "10.0.0.1:1234"
	secondRecorder := httptest.NewRecorder()
	handler.ServeHTTP(secondRecorder, secondRequest)

	if secondRecorder.Code != stdhttp.StatusTooManyRequests {
		t.Fatalf("expected second request status %d, got %d, body=%s", stdhttp.StatusTooManyRequests, secondRecorder.Code, secondRecorder.Body.String())
	}

	if secondRecorder.Header().Get("Retry-After") == "" {
		t.Fatalf("expected Retry-After header to be returned")
	}

	var response errorResponse
	if err := json.NewDecoder(secondRecorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode rate limit response: %v", err)
	}

	if response.Error.Code != "rate_limited" {
		t.Fatalf("expected rate_limited error code, got %q", response.Error.Code)
	}

	if metrics.Snapshot().RateLimitedTotal != 1 {
		t.Fatalf("expected rate limited total 1, got %d", metrics.Snapshot().RateLimitedTotal)
	}
}

func TestAuthLoginIsRateLimitedWhenConfigured(t *testing.T) {
	handler := newTestAuthMuxWithConfig(t, config.Config{
		AppName:              "zld-supportpilot-test",
		AppEnv:               "test",
		HTTPAddr:             ":0",
		HTTPRateLimitEnabled: true,
		HTTPRateLimitRPS:     1,
		HTTPRateLimitBurst:   1,
	})

	registerRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/auth/register",
		bytes.NewBufferString(`{"name":"Alice","email":"alice@example.com","password":"secret123","tenant_name":"中联数据智能服务台"}`),
	)
	registerRequest.Header.Set("Content-Type", "application/json")
	registerRecorder := httptest.NewRecorder()
	handler.ServeHTTP(registerRecorder, registerRequest)
	if registerRecorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected register status %d, got %d", stdhttp.StatusCreated, registerRecorder.Code)
	}

	firstLoginRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/auth/login",
		bytes.NewBufferString(`{"email":"alice@example.com","password":"secret123"}`),
	)
	firstLoginRequest.RemoteAddr = "10.0.0.9:2001"
	firstLoginRequest.Header.Set("Content-Type", "application/json")
	firstLoginRecorder := httptest.NewRecorder()
	handler.ServeHTTP(firstLoginRecorder, firstLoginRequest)
	if firstLoginRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected first login status %d, got %d, body=%s", stdhttp.StatusOK, firstLoginRecorder.Code, firstLoginRecorder.Body.String())
	}

	secondLoginRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/auth/login",
		bytes.NewBufferString(`{"email":"alice@example.com","password":"secret123"}`),
	)
	secondLoginRequest.RemoteAddr = "10.0.0.9:2001"
	secondLoginRequest.Header.Set("Content-Type", "application/json")
	secondLoginRecorder := httptest.NewRecorder()
	handler.ServeHTTP(secondLoginRecorder, secondLoginRequest)

	if secondLoginRecorder.Code != stdhttp.StatusTooManyRequests {
		t.Fatalf("expected second login status %d, got %d, body=%s", stdhttp.StatusTooManyRequests, secondLoginRecorder.Code, secondLoginRecorder.Body.String())
	}
}
