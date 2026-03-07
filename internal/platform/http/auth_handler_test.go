package http

import (
	"bytes"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/config"
)

type registerResponsePayload struct {
	UserID         string        `json:"user_id"`
	OrganizationID string        `json:"organization_id"`
	Role           identity.Role `json:"role"`
}

type loginResponsePayload struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type meResponsePayload struct {
	UserID         string                `json:"user_id"`
	OrganizationID string                `json:"organization_id"`
	Role           identity.Role         `json:"role"`
	Permissions    []identity.Permission `json:"permissions"`
}

func TestAuthRegister(t *testing.T) {
	handler := newTestAuthMux(t)
	body := bytes.NewBufferString(`{"name":"Alice","email":"alice@example.com","password":"secret123","tenant_name":"中联数据智能服务台"}`)
	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/register", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusCreated, recorder.Code, recorder.Body.String())
	}

	var response registerResponsePayload
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode register response: %v", err)
	}

	if response.UserID == "" {
		t.Fatalf("expected user_id to be returned")
	}

	if response.OrganizationID == "" {
		t.Fatalf("expected organization_id to be returned")
	}

	if response.Role != identity.RoleTenantAdmin {
		t.Fatalf("expected role %q, got %q", identity.RoleTenantAdmin, response.Role)
	}
}

func TestAuthLogin(t *testing.T) {
	handler := newTestAuthMux(t)
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

	loginRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/auth/login",
		bytes.NewBufferString(`{"email":"alice@example.com","password":"secret123"}`),
	)
	loginRequest.Header.Set("Content-Type", "application/json")
	loginRecorder := httptest.NewRecorder()

	handler.ServeHTTP(loginRecorder, loginRequest)

	if loginRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusOK, loginRecorder.Code, loginRecorder.Body.String())
	}

	var response loginResponsePayload
	if err := json.NewDecoder(loginRecorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	if response.AccessToken == "" {
		t.Fatalf("expected access_token to be returned")
	}

	if response.TokenType != "Bearer" {
		t.Fatalf("expected token type Bearer, got %q", response.TokenType)
	}

	if response.ExpiresIn <= 0 {
		t.Fatalf("expected positive expires_in, got %d", response.ExpiresIn)
	}
}

func TestAuthMe(t *testing.T) {
	handler := newTestAuthMux(t)
	accessToken := registerAndLogin(t, handler)

	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/auth/me", nil)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response meResponsePayload
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode me response: %v", err)
	}

	if response.UserID == "" {
		t.Fatalf("expected user_id to be returned")
	}

	if response.OrganizationID == "" {
		t.Fatalf("expected organization_id to be returned")
	}

	if response.Role != identity.RoleTenantAdmin {
		t.Fatalf("expected role %q, got %q", identity.RoleTenantAdmin, response.Role)
	}

	if len(response.Permissions) == 0 {
		t.Fatalf("expected permissions to be returned")
	}
}

func newTestAuthMux(t *testing.T) *stdhttp.ServeMux {
	t.Helper()

	identityService := identity.NewService(
		identity.NewMemoryUserRepository(),
		identity.NewMemoryMembershipRepository(),
		identity.NewPasswordManager(),
	)
	tokenManager := identity.NewTokenManager("test-signing-key", time.Hour)

	return NewMuxWithDependencies(config.Config{
		AppName:  "zld-supportpilot-test",
		AppEnv:   "test",
		HTTPAddr: ":0",
	}, AuthDependencies{
		IdentityService: identityService,
		TokenManager:    tokenManager,
	})
}

func registerAndLogin(t *testing.T, handler stdhttp.Handler) string {
	t.Helper()

	registerRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/auth/register",
		bytes.NewBufferString(`{"name":"Alice","email":"alice@example.com","password":"secret123","tenant_name":"中联数据智能服务台"}`),
	)
	registerRequest.Header.Set("Content-Type", "application/json")
	registerRecorder := httptest.NewRecorder()
	handler.ServeHTTP(registerRecorder, registerRequest)
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
	handler.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected login status %d, got %d, body=%s", stdhttp.StatusOK, loginRecorder.Code, loginRecorder.Body.String())
	}

	var response loginResponsePayload
	if err := json.NewDecoder(loginRecorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	return response.AccessToken
}
