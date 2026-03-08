package http

import (
	"encoding/json"
	"log"
	stdhttp "net/http"
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/config"
)

type RouteDependencies struct {
	Auth      *AuthDependencies
	Ticket    *TicketDependencies
	Knowledge *KnowledgeDependencies
	AI        *AIDependencies
}

type routeRuntime struct {
	observer    ObservabilityMiddleware
	rateLimiter *RateLimitMiddleware
	metrics     *HTTPMetrics
}

type healthzResponse struct {
	Name      string `json:"name"`
	Env       string `json:"env"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

func NewMux(cfg config.Config) *stdhttp.ServeMux {
	return newMux(cfg, nil)
}

func NewMuxWithDependencies(cfg config.Config, authDependencies AuthDependencies) *stdhttp.ServeMux {
	return newMux(cfg, &RouteDependencies{Auth: &authDependencies})
}

func NewMuxWithRouteDependencies(cfg config.Config, routeDependencies RouteDependencies) *stdhttp.ServeMux {
	return newMux(cfg, &routeDependencies)
}

func newMux(cfg config.Config, routeDependencies *RouteDependencies) *stdhttp.ServeMux {
	mux := stdhttp.NewServeMux()
	runtime := newRouteRuntime(cfg)

	mux.Handle("GET /healthz", runtime.public("GET /healthz", stdhttp.HandlerFunc(func(writer stdhttp.ResponseWriter, _ *stdhttp.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(stdhttp.StatusOK)

		_ = json.NewEncoder(writer).Encode(healthzResponse{
			Name:      cfg.AppName,
			Env:       cfg.AppEnv,
			Status:    "ok",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	})))
	mux.Handle("GET /debug/metrics/http", runtime.public("GET /debug/metrics/http", runtime.metrics.Handler()))

	if routeDependencies != nil && routeDependencies.Auth != nil {
		if routeDependencies.Auth.IdentityService != nil && routeDependencies.Auth.TokenManager != nil {
			if err := registerAuthRoutes(mux, runtime, *routeDependencies.Auth); err != nil {
				log.Printf("skip auth route registration: %v", err)
			}
		}
	}

	if routeDependencies != nil && routeDependencies.Ticket != nil {
		if routeDependencies.Auth == nil || routeDependencies.Auth.TokenManager == nil {
			log.Printf("skip ticket route registration: auth token manager is required")
		} else if err := registerTicketRoutes(mux, runtime, *routeDependencies.Auth, *routeDependencies.Ticket); err != nil {
			log.Printf("skip ticket route registration: %v", err)
		}
	}

	if routeDependencies != nil && routeDependencies.Knowledge != nil {
		if routeDependencies.Auth == nil || routeDependencies.Auth.TokenManager == nil {
			log.Printf("skip knowledge route registration: auth token manager is required")
		} else if err := registerKnowledgeRoutes(mux, runtime, *routeDependencies.Auth, *routeDependencies.Knowledge); err != nil {
			log.Printf("skip knowledge route registration: %v", err)
		}
	}

	if routeDependencies != nil && routeDependencies.AI != nil {
		if routeDependencies.Auth == nil || routeDependencies.Auth.TokenManager == nil {
			log.Printf("skip ai route registration: auth token manager is required")
		} else if err := registerAIRoutes(mux, runtime, *routeDependencies.Auth, *routeDependencies.AI); err != nil {
			log.Printf("skip ai route registration: %v", err)
		}
	}

	return mux
}

func newRouteRuntime(cfg config.Config) routeRuntime {
	logger := log.Default()
	if cfg.AppEnv == "test" {
		logger = newDiscardLogger()
	}

	metrics := NewHTTPMetrics()
	return routeRuntime{
		observer:    NewObservabilityMiddleware(logger, metrics),
		rateLimiter: NewRateLimitMiddleware(cfg, metrics),
		metrics:     metrics,
	}
}

func (runtime routeRuntime) public(route string, handler stdhttp.Handler) stdhttp.Handler {
	return requestIDMiddleware(runtime.observer.Wrap(route, handler))
}

func (runtime routeRuntime) publicRateLimited(route string, keyExtractor rateLimitKeyExtractor, handler stdhttp.Handler) stdhttp.Handler {
	return requestIDMiddleware(runtime.observer.Wrap(route, runtime.rateLimiter.Wrap(route, keyExtractor, handler)))
}

func (runtime routeRuntime) protected(route string, authMiddleware AuthMiddleware, handler stdhttp.Handler) stdhttp.Handler {
	return requestIDMiddleware(authMiddleware.RequireIdentity(runtime.observer.Wrap(route, handler)))
}

func (runtime routeRuntime) protectedRateLimited(route string, authMiddleware AuthMiddleware, keyExtractor rateLimitKeyExtractor, handler stdhttp.Handler) stdhttp.Handler {
	return requestIDMiddleware(authMiddleware.RequireIdentity(runtime.observer.Wrap(route, runtime.rateLimiter.Wrap(route, keyExtractor, handler))))
}
