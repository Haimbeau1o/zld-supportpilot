package http

import (
	"bytes"
	"encoding/json"
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/knowledge"
	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/config"
)

type processingTaskResponsePayload struct {
	ID           string                         `json:"id"`
	DocumentID   string                         `json:"document_id"`
	Status       knowledge.ProcessingTaskStatus `json:"status"`
	Stage        knowledge.ProcessingStage      `json:"stage"`
	Attempt      int                            `json:"attempt"`
	MaxAttempts  int                            `json:"max_attempts"`
	ErrorMessage string                         `json:"error_message"`
}

type processingTaskListResponsePayload struct {
	Items []processingTaskResponsePayload `json:"items"`
}

func TestKnowledgeListTasks(t *testing.T) {
	harness := newKnowledgeProcessingHarness(t)
	agent := identity.IdentityContext{UserID: "user-agent-1", OrganizationID: "org-1", Role: identity.RoleAgent, Permissions: identity.RolePermissions(identity.RoleAgent)}
	knowledgeBase := createKnowledgeBaseViaHTTP(t, harness.handler, harness.tokenManager, agent, `{"name":"IT 支持知识库","description":"用于沉淀 IT 文档"}`)
	document := uploadDocumentViaHTTP(t, harness.handler, harness.tokenManager, agent, knowledgeBase.ID, "vpn-guide.txt", []byte("first paragraph\n\nsecond paragraph"))

	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/knowledge/documents/"+document.ID+"/tasks", nil)
	request.Header.Set("Authorization", "Bearer "+issueKnowledgeToken(t, harness.tokenManager, agent))
	recorder := httptest.NewRecorder()

	harness.handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response processingTaskListResponsePayload
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode task list response: %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("expected 1 processing task, got %d", len(response.Items))
	}
	if response.Items[0].Status != knowledge.ProcessingTaskStatusQueued {
		t.Fatalf("expected queued task, got %q", response.Items[0].Status)
	}
	if response.Items[0].Attempt != 1 {
		t.Fatalf("expected attempt 1, got %d", response.Items[0].Attempt)
	}
}

func TestKnowledgeRetryDocument(t *testing.T) {
	harness := newKnowledgeProcessingHarness(t)
	harness.dispatcher.err = errors.New("queue unavailable")

	agent := identity.IdentityContext{UserID: "user-agent-1", OrganizationID: "org-1", Role: identity.RoleAgent, Permissions: identity.RolePermissions(identity.RoleAgent)}
	knowledgeBase := createKnowledgeBaseViaHTTP(t, harness.handler, harness.tokenManager, agent, `{"name":"IT 支持知识库","description":"用于沉淀 IT 文档"}`)
	document := uploadDocumentViaHTTP(t, harness.handler, harness.tokenManager, agent, knowledgeBase.ID, "vpn-guide.txt", []byte("first paragraph"))
	if document.Status != knowledge.DocumentStatusFailed {
		t.Fatalf("expected failed document when enqueue fails, got %q", document.Status)
	}

	harness.dispatcher.err = nil
	retryRequest := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/knowledge/documents/"+document.ID+"/retry", bytes.NewBuffer(nil))
	retryRequest.Header.Set("Authorization", "Bearer "+issueKnowledgeToken(t, harness.tokenManager, agent))
	retryRecorder := httptest.NewRecorder()

	harness.handler.ServeHTTP(retryRecorder, retryRequest)

	if retryRecorder.Code != stdhttp.StatusAccepted {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusAccepted, retryRecorder.Code, retryRecorder.Body.String())
	}

	var response processingTaskResponsePayload
	if err := json.NewDecoder(retryRecorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode retry response: %v", err)
	}
	if response.Attempt != 2 {
		t.Fatalf("expected retry attempt 2, got %d", response.Attempt)
	}
	if response.Status != knowledge.ProcessingTaskStatusQueued {
		t.Fatalf("expected queued retry task, got %q", response.Status)
	}
}

type knowledgeProcessingHarness struct {
	handler      *stdhttp.ServeMux
	tokenManager identity.TokenManager
	dispatcher   *switchableProcessingTaskDispatcher
}

func newKnowledgeProcessingHarness(t *testing.T) knowledgeProcessingHarness {
	t.Helper()

	taskRepository := knowledge.NewMemoryDocumentProcessingTaskRepository()
	chunkRepository := knowledge.NewMemoryDocumentChunkRepository()
	dispatcher := &switchableProcessingTaskDispatcher{}
	knowledgeService := knowledge.NewService(
		knowledge.NewMemoryKnowledgeBaseRepository(),
		knowledge.NewMemoryDocumentRepository(),
		knowledge.NewMemoryObjectStorage(),
		knowledge.WithProcessingPipeline(knowledge.ProcessingDependencies{
			TaskRepository:  taskRepository,
			ChunkRepository: chunkRepository,
			Dispatcher:      dispatcher,
			Parser:          knowledge.PlainTextDocumentParser{},
			Chunker:         knowledge.FixedSizeDocumentChunker{MaxCharacters: 64},
			Indexer:         knowledge.NewMemoryChunkIndexer(chunkRepository),
			MaxAttempts:     3,
		}),
	)
	tokenManager := identity.NewTokenManager("test-signing-key", time.Hour)

	return knowledgeProcessingHarness{
		handler: NewMuxWithRouteDependencies(config.Config{
			AppName:  "zld-supportpilot-test",
			AppEnv:   "test",
			HTTPAddr: ":0",
		}, RouteDependencies{
			Auth: &AuthDependencies{
				TokenManager: tokenManager,
			},
			Knowledge: &KnowledgeDependencies{
				KnowledgeService: knowledgeService,
			},
		}),
		tokenManager: tokenManager,
		dispatcher:   dispatcher,
	}
}

type switchableProcessingTaskDispatcher struct {
	err error
}

func (dispatcher *switchableProcessingTaskDispatcher) Enqueue(string) error {
	return dispatcher.err
}
