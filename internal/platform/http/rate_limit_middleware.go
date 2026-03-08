package http

import (
	"math"
	"net"
	stdhttp "net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/config"
)

type rateLimitKeyExtractor func(*stdhttp.Request) string

type rateLimitBucket struct {
	tokens     float64
	lastRefill time.Time
}

type RateLimitMiddleware struct {
	enabled bool
	rps     float64
	burst   int
	now     func() time.Time
	metrics *HTTPMetrics

	mu      sync.Mutex
	buckets map[string]rateLimitBucket
}

func NewRateLimitMiddleware(cfg config.Config, metrics *HTTPMetrics) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		enabled: cfg.HTTPRateLimitEnabled,
		rps:     cfg.HTTPRateLimitRPS,
		burst:   cfg.HTTPRateLimitBurst,
		now:     time.Now,
		metrics: metrics,
		buckets: map[string]rateLimitBucket{},
	}
}

func (middleware *RateLimitMiddleware) Wrap(route string, keyExtractor rateLimitKeyExtractor, next stdhttp.Handler) stdhttp.Handler {
	if middleware == nil || !middleware.enabled || middleware.rps <= 0 || middleware.burst <= 0 {
		return next
	}

	return stdhttp.HandlerFunc(func(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
		key := "anonymous"
		if keyExtractor != nil {
			if extracted := keyExtractor(request); extracted != "" {
				key = extracted
			}
		}

		allowed, retryAfter := middleware.allow(route + "|" + key)
		if allowed {
			next.ServeHTTP(writer, request)
			return
		}

		if middleware.metrics != nil {
			middleware.metrics.RecordRateLimited()
		}

		writer.Header().Set("Retry-After", strconv.Itoa(maxInt(1, int(math.Ceil(retryAfter.Seconds())))))
		writeError(writer, stdhttp.StatusTooManyRequests, "rate_limited", "请求频率过高，请稍后再试")
	})
}

func (middleware *RateLimitMiddleware) allow(key string) (bool, time.Duration) {
	now := middleware.now()

	middleware.mu.Lock()
	defer middleware.mu.Unlock()

	bucket, ok := middleware.buckets[key]
	if !ok {
		bucket = rateLimitBucket{
			tokens:     float64(middleware.burst),
			lastRefill: now,
		}
	}

	elapsed := now.Sub(bucket.lastRefill).Seconds()
	if elapsed > 0 {
		bucket.tokens = math.Min(float64(middleware.burst), bucket.tokens+elapsed*middleware.rps)
		bucket.lastRefill = now
	}

	if bucket.tokens >= 1 {
		bucket.tokens--
		middleware.buckets[key] = bucket
		return true, 0
	}

	middleware.buckets[key] = bucket
	requiredTokens := 1 - bucket.tokens
	retryAfter := time.Duration((requiredTokens / middleware.rps) * float64(time.Second))
	return false, retryAfter
}

func clientAddressRateLimitKey(request *stdhttp.Request) string {
	forwardedFor := strings.TrimSpace(strings.Split(request.Header.Get("X-Forwarded-For"), ",")[0])
	if forwardedFor != "" {
		return forwardedFor
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(request.RemoteAddr))
	if err == nil && host != "" {
		return host
	}

	return strings.TrimSpace(request.RemoteAddr)
}

func identityOrClientRateLimitKey(request *stdhttp.Request) string {
	if identityContext, ok := identityContextFromContext(request.Context()); ok {
		// 限流键优先绑定组织 + 用户身份边界，这样比单纯按 IP 更贴近真实租户系统，也便于解释“谁在消耗高成本接口配额”。
		if identityContext.OrganizationID != "" || identityContext.UserID != "" {
			return identityContext.OrganizationID + ":" + identityContext.UserID
		}
	}

	return clientAddressRateLimitKey(request)
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}

	return right
}
