package app

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ai"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/knowledge"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ticket"
	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/config"
	platformhttp "github.com/Haimbeau1o/zld-supportpilot/internal/platform/http"
	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/persistence"
)

var openPostgres = persistence.OpenPostgres

type App struct {
	config  config.Config
	server  *http.Server
	cleanup []func() error
}

func New() (*App, error) {
	cfg := config.Load()

	if cfg.PersistenceMode == "postgres" && strings.TrimSpace(cfg.PostgresDSN) == "" {
		return nil, fmt.Errorf("postgres persistence mode requires POSTGRES_DSN")
	}

	cleanup := make([]func() error, 0, 2)
	bootstrapSucceeded := false
	defer func() {
		if bootstrapSucceeded {
			return
		}
		for index := len(cleanup) - 1; index >= 0; index-- {
			_ = cleanup[index]()
		}
	}()

	var identityService *identity.Service
	var ticketService *ticket.Service
	var knowledgeBaseRepository knowledge.KnowledgeBaseRepository
	var documentRepository knowledge.DocumentRepository
	var taskRepository knowledge.DocumentProcessingTaskRepository
	var chunkRepository knowledge.DocumentChunkRepository
	var intakeRepository ai.UnifiedIntakeRepository
	var feedbackRepository ai.AnswerFeedbackRepository

	switch cfg.PersistenceMode {
	case "memory":
		// memory 模式继续承担默认开发体验，保证项目在无外部依赖时也能直接跑通主链路。
		identityService = identity.NewService(
			identity.NewMemoryUserRepository(),
			identity.NewMemoryMembershipRepository(),
			identity.NewPasswordManager(),
		)
		ticketService = ticket.NewService(
			ticket.NewMemoryTicketRepository(),
			ticket.WithCollaborationDependencies(ticket.CollaborationDependencies{
				CommentRepository: ticket.NewMemoryTicketCommentRepository(),
				AuditRepository:   ticket.NewMemoryTicketAuditEventRepository(),
			}),
		)
		knowledgeBaseRepository = knowledge.NewMemoryKnowledgeBaseRepository()
		documentRepository = knowledge.NewMemoryDocumentRepository()
		taskRepository = knowledge.NewMemoryDocumentProcessingTaskRepository()
		chunkRepository = knowledge.NewMemoryDocumentChunkRepository()
		intakeRepository = ai.NewMemoryUnifiedIntakeRepository()
		feedbackRepository = ai.NewMemoryAnswerFeedbackRepository()
	case "postgres":
		db, err := openPostgres(context.Background(), cfg.PostgresDSN)
		if err != nil {
			return nil, fmt.Errorf("open postgres persistence: %w", err)
		}
		cleanup = append(cleanup, db.Close)

		identityService = identity.NewService(
			identity.NewPostgresUserRepository(db),
			identity.NewPostgresMembershipRepository(db),
			identity.NewPasswordManager(),
		)
		ticketService = ticket.NewService(
			ticket.NewPostgresTicketRepository(db),
			ticket.WithCollaborationDependencies(ticket.CollaborationDependencies{
				CommentRepository: ticket.NewPostgresTicketCommentRepository(db),
				AuditRepository:   ticket.NewPostgresTicketAuditEventRepository(db),
			}),
		)
		knowledgeBaseRepository = knowledge.NewPostgresKnowledgeBaseRepository(db)
		documentRepository = knowledge.NewPostgresDocumentRepository(db)
		taskRepository = knowledge.NewPostgresDocumentProcessingTaskRepository(db)
		chunkRepository = knowledge.NewPostgresDocumentChunkRepository(db)
		intakeRepository = ai.NewPostgresUnifiedIntakeRepository(db)
		feedbackRepository = ai.NewPostgresAnswerFeedbackRepository(db)
	default:
		return nil, fmt.Errorf("unsupported persistence mode: %s", cfg.PersistenceMode)
	}

	var objectStorage knowledge.ObjectStorage
	switch cfg.DocumentStorageMode {
	case "memory":
		objectStorage = knowledge.NewMemoryObjectStorage()
	case "filesystem":
		if strings.TrimSpace(cfg.DocumentStorageRoot) == "" {
			return nil, fmt.Errorf("filesystem document storage mode requires DOCUMENT_STORAGE_ROOT")
		}
		objectStorage = knowledge.NewFileSystemObjectStorage(cfg.DocumentStorageRoot)
	default:
		return nil, fmt.Errorf("unsupported document storage mode: %s", cfg.DocumentStorageMode)
	}

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
	cleanup = append(cleanup, asyncProcessor.Close)

	aiService := ai.NewService(ai.ServiceDependencies{
		ChunkSource:        knowledgeService,
		Retriever:          ai.NewVectorRetriever(ai.NewHashingEmbedder(128)),
		AnswerGenerator:    ai.TemplateAnswerGenerator{},
		MinConfidence:      0.15,
		TicketWorkspace:    ticketService,
		TicketAnalyzer:     ai.TemplateTicketAnalyzer{},
		IntakeRepository:   intakeRepository,
		FeedbackRepository: feedbackRepository,
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

	bootstrapSucceeded = true
	return &App{
		config: cfg,
		server: &http.Server{
			Addr:    cfg.HTTPAddr,
			Handler: mux,
		},
		cleanup: cleanup,
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
