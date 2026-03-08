package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	stdhttp "net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type HTTPMetricsSnapshot struct {
	RequestsTotal    uint64            `json:"requests_total"`
	RateLimitedTotal uint64            `json:"rate_limited_total"`
	StatusTotals     map[string]uint64 `json:"status_totals"`
	RouteTotals      map[string]uint64 `json:"route_totals"`
	LatencyBuckets   map[string]uint64 `json:"latency_ms_buckets"`
}

type HTTPMetrics struct {
	mu               sync.RWMutex
	requestsTotal    uint64
	rateLimitedTotal uint64
	statusTotals     map[string]uint64
	routeTotals      map[string]uint64
	latencyBuckets   map[string]uint64
}

type ObservabilityMiddleware struct {
	logger  *log.Logger
	metrics *HTTPMetrics
}

type responseRecorder struct {
	stdhttp.ResponseWriter
	statusCode int
	bytes      int
}

func NewHTTPMetrics() *HTTPMetrics {
	return &HTTPMetrics{
		statusTotals:   map[string]uint64{},
		routeTotals:    map[string]uint64{},
		latencyBuckets: map[string]uint64{},
	}
}

func (metrics *HTTPMetrics) Observe(route string, statusCode int, duration time.Duration) {
	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	metrics.requestsTotal++
	metrics.routeTotals[route]++
	metrics.statusTotals[strconv.Itoa(statusCode)]++
	metrics.latencyBuckets[latencyBucket(duration)]++
}

func (metrics *HTTPMetrics) RecordRateLimited() {
	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	metrics.rateLimitedTotal++
}

func (metrics *HTTPMetrics) Snapshot() HTTPMetricsSnapshot {
	metrics.mu.RLock()
	defer metrics.mu.RUnlock()

	return HTTPMetricsSnapshot{
		RequestsTotal:    metrics.requestsTotal,
		RateLimitedTotal: metrics.rateLimitedTotal,
		StatusTotals:     cloneMetricMap(metrics.statusTotals),
		RouteTotals:      cloneMetricMap(metrics.routeTotals),
		LatencyBuckets:   cloneMetricMap(metrics.latencyBuckets),
	}
}

func (metrics *HTTPMetrics) Handler() stdhttp.Handler {
	return stdhttp.HandlerFunc(func(writer stdhttp.ResponseWriter, _ *stdhttp.Request) {
		writeJSON(writer, stdhttp.StatusOK, metrics.Snapshot())
	})
}

func NewObservabilityMiddleware(logger *log.Logger, metrics *HTTPMetrics) ObservabilityMiddleware {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	if metrics == nil {
		metrics = NewHTTPMetrics()
	}

	return ObservabilityMiddleware{logger: logger, metrics: metrics}
}

func (middleware ObservabilityMiddleware) Wrap(route string, next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
		startedAt := time.Now()
		recorder := &responseRecorder{ResponseWriter: writer, statusCode: stdhttp.StatusOK}
		next.ServeHTTP(recorder, request)

		duration := time.Since(startedAt)
		middleware.metrics.Observe(route, recorder.statusCode, duration)

		entry := map[string]any{
			"message":        "http_request",
			"request_id":     requestIDOrFallback(request.Context()),
			"route":          route,
			"method":         request.Method,
			"path":           request.URL.Path,
			"status":         recorder.statusCode,
			"duration_ms":    duration.Milliseconds(),
			"response_bytes": recorder.bytes,
		}
		if identityContext, ok := identityContextFromContext(request.Context()); ok {
			entry["user_id"] = identityContext.UserID
			entry["organization_id"] = identityContext.OrganizationID
		}

		payload, err := json.Marshal(entry)
		if err != nil {
			middleware.logger.Printf(`{"message":"http_request_log_failed","route":%q}`, route)
			return
		}

		middleware.logger.Print(string(payload))
	})
}

func requestIDMiddleware(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
		requestID := normalizedRequestID(request.Header.Get("X-Request-ID"))
		if requestID == "" {
			// 当前阶段用请求 ID 作为最小链路追踪锚点：哪怕还没有完整 trace 系统，也能把一次请求在响应、日志和指标之间串起来。
			requestID = generateRequestID()
		}

		writer.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(writer, request.WithContext(withRequestIDContext(request.Context(), requestID)))
	})
}

func newDiscardLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}

func requestIDOrFallback(ctx context.Context) string {
	requestID, ok := requestIDFromContext(ctx)
	if !ok {
		return ""
	}

	return requestID
}

func generateRequestID() string {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	return hex.EncodeToString(buffer)
}

func normalizedRequestID(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || len(trimmed) > 128 {
		return ""
	}

	return trimmed
}

func cloneMetricMap(source map[string]uint64) map[string]uint64 {
	cloned := make(map[string]uint64, len(source))
	for key, value := range source {
		cloned[key] = value
	}

	return cloned
}

func latencyBucket(duration time.Duration) string {
	milliseconds := duration.Milliseconds()
	switch {
	case milliseconds <= 10:
		return "le_10ms"
	case milliseconds <= 50:
		return "le_50ms"
	case milliseconds <= 100:
		return "le_100ms"
	case milliseconds <= 250:
		return "le_250ms"
	case milliseconds <= 500:
		return "le_500ms"
	case milliseconds <= 1000:
		return "le_1000ms"
	default:
		return "gt_1000ms"
	}
}

func (recorder *responseRecorder) WriteHeader(statusCode int) {
	recorder.statusCode = statusCode
	recorder.ResponseWriter.WriteHeader(statusCode)
}

func (recorder *responseRecorder) Write(payload []byte) (int, error) {
	written, err := recorder.ResponseWriter.Write(payload)
	recorder.bytes += written
	return written, err
}
