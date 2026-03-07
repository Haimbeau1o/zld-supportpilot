package app

import (
	"bytes"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
)

type appLoginResponse struct {
	AccessToken string `json:"access_token"`
}

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

func TestNewWiresTicketRoutes(t *testing.T) {
	t.Setenv("AUTH_SIGNING_KEY", "test-signing-key")
	t.Setenv("AUTH_TOKEN_TTL_SECONDS", "3600")

	application, err := New()
	if err != nil {
		t.Fatalf("expected app bootstrap success, got error: %v", err)
	}

	registerRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/auth/register",
		bytes.NewBufferString(`{"name":"Alice","email":"alice@example.com","password":"secret123","tenant_name":"中联数据智能服务台"}`),
	)
	registerRequest.Header.Set("Content-Type", "application/json")
	registerRecorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(registerRecorder, registerRequest)
	if registerRecorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected register status %d, got %d, body=%s", stdhttp.StatusCreated, registerRecorder.Code, registerRecorder.Body.String())
	}

	loginRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/auth/login",
		bytes.NewBufferString(`{"email":"alice@example.com","password":"secret123"}`),
	)
	loginRequest.Header.Set("Content-Type", "application/json")
	loginRecorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected login status %d, got %d, body=%s", stdhttp.StatusOK, loginRecorder.Code, loginRecorder.Body.String())
	}

	var loginResponse appLoginResponse
	if err := json.NewDecoder(loginRecorder.Body).Decode(&loginResponse); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	createTicketRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/tickets",
		bytes.NewBufferString(`{"title":"VPN 无法连接","description":"今天上午开始无法连接公司 VPN","category":"network","priority":"high"}`),
	)
	createTicketRequest.Header.Set("Content-Type", "application/json")
	createTicketRequest.Header.Set("Authorization", "Bearer "+loginResponse.AccessToken)
	createTicketRecorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(createTicketRecorder, createTicketRequest)

	if createTicketRecorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusCreated, createTicketRecorder.Code, createTicketRecorder.Body.String())
	}
}
