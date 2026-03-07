package app

import (
	"bytes"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
)

func TestNewWiresAuthRoutes(t *testing.T) {
	t.Setenv("AUTH_SIGNING_KEY", "test-signing-key")
	t.Setenv("AUTH_TOKEN_TTL_SECONDS", "3600")

	application, err := New()
	if err != nil {
		t.Fatalf("expected app bootstrap success, got error: %v", err)
	}

	request := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/auth/register",
		bytes.NewBufferString(`{"name":"Alice","email":"alice@example.com","password":"secret123","tenant_name":"中联数据智能服务台"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	application.server.Handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusCreated, recorder.Code, recorder.Body.String())
	}
}
