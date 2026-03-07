package app

import (
	"fmt"
	"net/http"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/knowledge"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ticket"
	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/config"
	platformhttp "github.com/Haimbeau1o/zld-supportpilot/internal/platform/http"
)

type App struct {
	config config.Config
	server *http.Server
}

func New() (*App, error) {
	cfg := config.Load()

	// 当前阶段先使用内存适配，把认证、租户与 RBAC、工单主流程以及知识上传链路的边界稳定下来，避免数据库和真实对象存储过早侵入主线开发。
	identityService := identity.NewService(
		identity.NewMemoryUserRepository(),
		identity.NewMemoryMembershipRepository(),
		identity.NewPasswordManager(),
	)
	ticketService := ticket.NewService(ticket.NewMemoryTicketRepository())
	knowledgeService := knowledge.NewService(
		knowledge.NewMemoryKnowledgeBaseRepository(),
		knowledge.NewMemoryDocumentRepository(),
		knowledge.NewMemoryObjectStorage(),
	)
	tokenManager := identity.NewTokenManager(cfg.AuthSigningKey, cfg.AuthTokenTTL)
	mux := platformhttp.NewMuxWithRouteDependencies(cfg, platformhttp.RouteDependencies{
		Auth: &platformhttp.AuthDependencies{
			IdentityService: identityService,
			TokenManager:    tokenManager,
		},
		Ticket: &platformhttp.TicketDependencies{
			TicketService: ticketService,
		},
		Knowledge: &platformhttp.KnowledgeDependencies{
			KnowledgeService: knowledgeService,
		},
	})

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
