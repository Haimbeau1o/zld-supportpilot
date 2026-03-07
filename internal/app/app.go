package app

import (
	"fmt"
	"net/http"

	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/config"
	platformhttp "github.com/Haimbeau1o/zld-supportpilot/internal/platform/http"
)

type App struct {
	config config.Config
	server *http.Server
}

func New() (*App, error) {
	cfg := config.Load()
	mux := platformhttp.NewMux(cfg)

	return &App{
		config: cfg,
		server: &http.Server{
			Addr:    cfg.HTTPAddr,
			Handler: mux,
		},
	}, nil
}

func (a *App) Run() error {
	fmt.Printf("%s listening on %s\n", a.config.AppName, a.config.HTTPAddr)
	return a.server.ListenAndServe()
}
