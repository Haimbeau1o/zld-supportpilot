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
	mux.HandleFunc("/healthz", func(writer stdhttp.ResponseWriter, _ *stdhttp.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(stdhttp.StatusOK)

		_ = json.NewEncoder(writer).Encode(healthzResponse{
			Name:      cfg.AppName,
			Env:       cfg.AppEnv,
			Status:    "ok",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	})

	if routeDependencies != nil && routeDependencies.Auth != nil {
		if routeDependencies.Auth.IdentityService != nil && routeDependencies.Auth.TokenManager != nil {
			if err := registerAuthRoutes(mux, *routeDependencies.Auth); err != nil {
				log.Printf("skip auth route registration: %v", err)
			}
		}
	}

	if routeDependencies != nil && routeDependencies.Ticket != nil {
		if routeDependencies.Auth == nil || routeDependencies.Auth.TokenManager == nil {
			log.Printf("skip ticket route registration: auth token manager is required")
		} else if err := registerTicketRoutes(mux, *routeDependencies.Auth, *routeDependencies.Ticket); err != nil {
			log.Printf("skip ticket route registration: %v", err)
		}
	}

	if routeDependencies != nil && routeDependencies.Knowledge != nil {
		if routeDependencies.Auth == nil || routeDependencies.Auth.TokenManager == nil {
			log.Printf("skip knowledge route registration: auth token manager is required")
		} else if err := registerKnowledgeRoutes(mux, *routeDependencies.Auth, *routeDependencies.Knowledge); err != nil {
			log.Printf("skip knowledge route registration: %v", err)
		}
	}

	return mux
}
