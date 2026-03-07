package http

import (
	"encoding/json"
	"log"
	stdhttp "net/http"
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/config"
)

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
	return newMux(cfg, &authDependencies)
}

func newMux(cfg config.Config, authDependencies *AuthDependencies) *stdhttp.ServeMux {
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

	if authDependencies != nil {
		if err := registerAuthRoutes(mux, *authDependencies); err != nil {
			log.Printf("skip auth route registration: %v", err)
		}
	}

	return mux
}
