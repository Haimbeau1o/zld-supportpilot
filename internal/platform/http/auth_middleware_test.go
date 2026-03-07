package http

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareRejectsMissingToken(t *testing.T) {
	handler := newTestAuthMux(t)
	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/auth/me", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
}
