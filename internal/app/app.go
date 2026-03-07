package app

import (
	"fmt"
	"net/http"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ai"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/knowledge"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ticket"
	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/config"
	platformhttp "github.com/Haimbeau1o/zld-supportpilot/internal/platform/http"
)

type App struct {
	config  config.Config
	server  *http.Server
	cleanup []func() error
}

func New() (*App, error) {
	cfg := config.Load()

	// 当前阶段先使用内存适配，把认证、租户与 RBAC、工单主流程以及知识处理链路的边界稳定下来，避免数据库、对象存储和消息队列过早侵入主线开发。
	identityService := identity.NewService(
		identity.NewMemoryUserRepository(),
		identity.NewMemoryMembershipRepository(),
		identity.NewPasswordManager(),
	)
	ticketService := ticket.NewService(ticket.NewMemoryTicketRepository())

	knowledgeBaseRepository := knowledge.NewMemoryKnowledgeBaseRepository()
	documentRepository := knowledge.NewMemoryDocumentRepository()
	taskRepository := knowledge.NewMemoryDocumentProcessingTaskRepository()
	chunkRepository := knowledge.NewMemoryDocumentChunkRepository()
	objectStorage := knowledge.NewMemoryObjectStorage()
	processingQueue := knowledge.NewMemoryProcessingQueue(32)
	knowledgeService := knowledge.NewService(
		knowledgeBaseRepository,
		documentRepository,
		objectStorage,
		knowledge.WithProcessingPipeline(knowledge.ProcessingDependencies{
			TaskRepository:  taskRepository,
			ChunkRepository: chunkRepository,
			Dispatcher:      processingQueue,
			Parser:          knowledge.PlainTextDocumentParser{},
			Chunker:         knowledge.FixedSizeDocumentChunker{MaxCharacters: 200},
			Indexer:         knowledge.NewMemoryChunkIndexer(chunkRepository),
			MaxAttempts:     3,
		}),
	)
	asyncProcessor := knowledge.NewAsyncProcessor(processingQueue, 1, knowledgeService.ProcessTask)
	asyncProcessor.Start()
	aiService := ai.NewService(ai.ServiceDependencies{
		ChunkSource:     knowledgeService,
		Retriever:       ai.NewVectorRetriever(ai.NewHashingEmbedder(128)),
		AnswerGenerator: ai.TemplateAnswerGenerator{},
		MinConfidence:   0.15,
	})

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
		AI: &platformhttp.AIDependencies{
			AIService: aiService,
		},
	})

	return &App{
		config: cfg,
		server: &http.Server{
			Addr:    cfg.HTTPAddr,
			Handler: mux,
		},
		cleanup: []func() error{asyncProcessor.Close},
	}, nil
}

func (a *App) Run() error {
	defer a.Close()
	fmt.Printf("%s listening on %s\n", a.config.AppName, a.config.HTTPAddr)
	return a.server.ListenAndServe()
}

func (a *App) Close() error {
	var firstErr error
	for index := len(a.cleanup) - 1; index >= 0; index-- {
		if err := a.cleanup[index](); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
