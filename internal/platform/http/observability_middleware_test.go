package http

import (
	"bytes"
	"encoding/json"
	"log"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/config"
)

type httpMetricsSnapshotPayload struct {
	RequestsTotal  uint64            `json:"requests_total"`
	RateLimited    uint64            `json:"rate_limited_total"`
	StatusTotals   map[string]uint64 `json:"status_totals"`
	RouteTotals    map[string]uint64 `json:"route_totals"`
	LatencyBuckets map[string]uint64 `json:"latency_ms_buckets"`
}

func TestHealthzAddsRequestIDAndPublishesMetrics(t *testing.T) {
	handler := NewMux(config.Config{
		AppName:  "zld-supportpilot-test",
		AppEnv:   "test",
		HTTPAddr: ":0",
	})

	request := httptest.NewRequest(stdhttp.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusOK, recorder.Code)
	}

	requestID := recorder.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Fatalf("expected X-Request-ID header to be populated")
	}

	metricsRequest := httptest.NewRequest(stdhttp.MethodGet, "/debug/metrics/http", nil)
	metricsRecorder := httptest.NewRecorder()
	handler.ServeHTTP(metricsRecorder, metricsRequest)

	if metricsRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected metrics status %d, got %d, body=%s", stdhttp.StatusOK, metricsRecorder.Code, metricsRecorder.Body.String())
	}

	var snapshot httpMetricsSnapshotPayload
	if err := json.NewDecoder(metricsRecorder.Body).Decode(&snapshot); err != nil {
		t.Fatalf("decode metrics response: %v", err)
	}

	if snapshot.RequestsTotal < 1 {
		t.Fatalf("expected metrics requests_total to be recorded, got %d", snapshot.RequestsTotal)
	}

	if snapshot.RouteTotals["GET /healthz"] != 1 {
		t.Fatalf("expected route total for healthz to be 1, got %d", snapshot.RouteTotals["GET /healthz"])
	}

	if snapshot.StatusTotals["200"] < 1 {
		t.Fatalf("expected status total for 200 to be recorded")
	}
}

func TestHealthzReusesProvidedRequestID(t *testing.T) {
	handler := NewMux(config.Config{
		AppName:  "zld-supportpilot-test",
		AppEnv:   "test",
		HTTPAddr: ":0",
	})

	request := httptest.NewRequest(stdhttp.MethodGet, "/healthz", nil)
	request.Header.Set("X-Request-ID", "req-fixed-123")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Header().Get("X-Request-ID") != "req-fixed-123" {
		t.Fatalf("expected response to reuse provided request id, got %q", recorder.Header().Get("X-Request-ID"))
	}
}

func TestObservabilityMiddlewareLogsStructuredRequest(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger := log.New(buffer, "", 0)
	metrics := NewHTTPMetrics()
	middleware := NewObservabilityMiddleware(logger, metrics)

	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/auth/me", nil)
	request = request.WithContext(withRequestIDContext(request.Context(), "req-123"))
	request = request.WithContext(withIdentityContext(request.Context(), identity.IdentityContext{
		UserID:         "user-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}))
	recorder := httptest.NewRecorder()

	handler := middleware.Wrap("GET /api/v1/auth/me", stdhttp.HandlerFunc(func(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
		writeJSON(writer, stdhttp.StatusOK, map[string]string{"status": "ok"})
	}))

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusOK, recorder.Code)
	}

	var entry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buffer.Bytes()), &entry); err != nil {
		t.Fatalf("decode log entry: %v, raw=%s", err, buffer.String())
	}

	if entry["request_id"] != "req-123" {
		t.Fatalf("expected request_id req-123, got %#v", entry["request_id"])
	}

	if entry["route"] != "GET /api/v1/auth/me" {
		t.Fatalf("expected route to be logged, got %#v", entry["route"])
	}

	if entry["user_id"] != "user-1" {
		t.Fatalf("expected user_id to be logged, got %#v", entry["user_id"])
	}

	if entry["organization_id"] != "org-1" {
		t.Fatalf("expected organization_id to be logged, got %#v", entry["organization_id"])
	}

	if entry["status"] != float64(stdhttp.StatusOK) {
		t.Fatalf("expected status %d, got %#v", stdhttp.StatusOK, entry["status"])
	}
}
