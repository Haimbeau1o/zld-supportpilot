package http

import (
	"bytes"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ai"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/knowledge"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ticket"
	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/config"
)

type aiAnswerResponsePayload struct {
	Status     ai.AnswerStatus `json:"status"`
	Answer     string          `json:"answer"`
	Confidence float64         `json:"confidence"`
	Citations  []struct {
		ChunkID    string  `json:"chunk_id"`
		DocumentID string  `json:"document_id"`
		Score      float64 `json:"score"`
		Content    string  `json:"content"`
	} `json:"citations"`
}

type aiTicketAssistResponsePayload struct {
	CategorySuggestion string `json:"category_suggestion"`
	Summary            string `json:"summary"`
	ReplyDraft         string `json:"reply_draft"`
	RecordedCommentID  string `json:"recorded_comment_id"`
	Citations          []struct {
		ChunkID string `json:"chunk_id"`
	} `json:"citations"`
}

func TestAIAnswerQuestion(t *testing.T) {
	harness := newTestAIMux(t, []knowledge.DocumentChunk{{ID: "chunk-1", DocumentID: "doc-1", KnowledgeBaseID: "kb-1", Content: "VPN 无法连接时，请先重置客户端。"}}, 0.15)
	agent := identity.IdentityContext{UserID: "user-agent-1", OrganizationID: "org-1", Role: identity.RoleAgent, Permissions: identity.RolePermissions(identity.RoleAgent)}

	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/ai/knowledge/bases/kb-1/answers", bytes.NewBufferString(`{"question":"VPN 无法连接怎么办？","top_k":2}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+issueKnowledgeToken(t, harness.tokenManager, agent))
	recorder := httptest.NewRecorder()

	harness.handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response aiAnswerResponsePayload
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != ai.AnswerStatusAnswered {
		t.Fatalf("expected answered status, got %q", response.Status)
	}
	if len(response.Citations) == 0 {
		t.Fatalf("expected citations to be returned")
	}
}

func TestAIAnswerQuestionDegradesWhenLowConfidence(t *testing.T) {
	harness := newTestAIMux(t, []knowledge.DocumentChunk{{ID: "chunk-1", DocumentID: "doc-1", KnowledgeBaseID: "kb-1", Content: "薪资审批需要走人力系统。"}}, 0.25)
	agent := identity.IdentityContext{UserID: "user-agent-1", OrganizationID: "org-1", Role: identity.RoleAgent, Permissions: identity.RolePermissions(identity.RoleAgent)}

	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/ai/knowledge/bases/kb-1/answers", bytes.NewBufferString(`{"question":"VPN 无法连接怎么办？","top_k":2}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+issueKnowledgeToken(t, harness.tokenManager, agent))
	recorder := httptest.NewRecorder()

	harness.handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response aiAnswerResponsePayload
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != ai.AnswerStatusDegraded {
		t.Fatalf("expected degraded status, got %q", response.Status)
	}
}

func TestAITicketAssist(t *testing.T) {
	harness := newTestAIMux(t, []knowledge.DocumentChunk{{ID: "chunk-1", DocumentID: "doc-1", KnowledgeBaseID: "kb-1", Content: "VPN 无法连接时，请先重置客户端，然后重新登录。"}}, 0.15)
	endUser := identity.IdentityContext{UserID: "user-end-1", OrganizationID: "org-1", Role: identity.RoleEndUser, Permissions: identity.RolePermissions(identity.RoleEndUser)}
	agent := identity.IdentityContext{UserID: "user-agent-1", OrganizationID: "org-1", Role: identity.RoleAgent, Permissions: identity.RolePermissions(identity.RoleAgent)}

	createdTicket, err := harness.ticketService.CreateTicket(endUser, ticket.CreateTicketInput{
		Title:       "VPN 无法连接",
		Description: "今天上午开始无法连接公司 VPN，重试后仍失败。",
		Category:    "network",
		Priority:    ticket.TicketPriorityHigh,
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/ai/tickets/"+createdTicket.ID+"/assist", bytes.NewBufferString(`{"knowledge_base_id":"kb-1"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+issueKnowledgeToken(t, harness.tokenManager, agent))
	recorder := httptest.NewRecorder()

	harness.handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response aiTicketAssistResponsePayload
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode assist response: %v", err)
	}
	if response.CategorySuggestion == "" || response.Summary == "" || response.ReplyDraft == "" {
		t.Fatalf("expected structured ticket assist payload, got %+v", response)
	}
	if response.RecordedCommentID == "" {
		t.Fatalf("expected recorded comment id")
	}
}

func TestAIEndUserCannotTriggerTicketAssist(t *testing.T) {
	harness := newTestAIMux(t, []knowledge.DocumentChunk{{ID: "chunk-1", DocumentID: "doc-1", KnowledgeBaseID: "kb-1", Content: "VPN 无法连接时，请先重置客户端。"}}, 0.15)
	endUser := identity.IdentityContext{UserID: "user-end-1", OrganizationID: "org-1", Role: identity.RoleEndUser, Permissions: identity.RolePermissions(identity.RoleEndUser)}

	createdTicket, err := harness.ticketService.CreateTicket(endUser, ticket.CreateTicketInput{
		Title:       "VPN 无法连接",
		Description: "今天上午开始无法连接公司 VPN，重试后仍失败。",
		Category:    "network",
		Priority:    ticket.TicketPriorityHigh,
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/ai/tickets/"+createdTicket.ID+"/assist", bytes.NewBufferString(`{"knowledge_base_id":"kb-1"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+issueKnowledgeToken(t, harness.tokenManager, endUser))
	recorder := httptest.NewRecorder()

	harness.handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusForbidden {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusForbidden, recorder.Code, recorder.Body.String())
	}
}

func TestAITicketAssistIsRateLimitedWhenConfigured(t *testing.T) {
	harness := newTestAIMuxWithConfig(t, []knowledge.DocumentChunk{{ID: "chunk-1", DocumentID: "doc-1", KnowledgeBaseID: "kb-1", Content: "VPN 无法连接时，请先重置客户端，然后重新登录。"}}, 0.15, config.Config{
		AppName:              "zld-supportpilot-test",
		AppEnv:               "test",
		HTTPAddr:             ":0",
		HTTPRateLimitEnabled: true,
		HTTPRateLimitRPS:     1,
		HTTPRateLimitBurst:   1,
	})
	endUser := identity.IdentityContext{UserID: "user-end-1", OrganizationID: "org-1", Role: identity.RoleEndUser, Permissions: identity.RolePermissions(identity.RoleEndUser)}
	agent := identity.IdentityContext{UserID: "user-agent-1", OrganizationID: "org-1", Role: identity.RoleAgent, Permissions: identity.RolePermissions(identity.RoleAgent)}

	createdTicket, err := harness.ticketService.CreateTicket(endUser, ticket.CreateTicketInput{
		Title:       "VPN 无法连接",
		Description: "今天上午开始无法连接公司 VPN，重试后仍失败。",
		Category:    "network",
		Priority:    ticket.TicketPriorityHigh,
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	firstRequest := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/ai/tickets/"+createdTicket.ID+"/assist", bytes.NewBufferString(`{"knowledge_base_id":"kb-1"}`))
	firstRequest.Header.Set("Content-Type", "application/json")
	firstRequest.Header.Set("Authorization", "Bearer "+issueKnowledgeToken(t, harness.tokenManager, agent))
	firstRecorder := httptest.NewRecorder()
	harness.handler.ServeHTTP(firstRecorder, firstRequest)
	if firstRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected first assist status %d, got %d, body=%s", stdhttp.StatusOK, firstRecorder.Code, firstRecorder.Body.String())
	}

	secondRequest := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/ai/tickets/"+createdTicket.ID+"/assist", bytes.NewBufferString(`{"knowledge_base_id":"kb-1"}`))
	secondRequest.Header.Set("Content-Type", "application/json")
	secondRequest.Header.Set("Authorization", "Bearer "+issueKnowledgeToken(t, harness.tokenManager, agent))
	secondRecorder := httptest.NewRecorder()
	harness.handler.ServeHTTP(secondRecorder, secondRequest)
	if secondRecorder.Code != stdhttp.StatusTooManyRequests {
		t.Fatalf("expected second assist status %d, got %d, body=%s", stdhttp.StatusTooManyRequests, secondRecorder.Code, secondRecorder.Body.String())
	}
}

func TestAIRequiresAuthentication(t *testing.T) {
	harness := newTestAIMux(t, nil, 0.15)
	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/ai/knowledge/bases/kb-1/answers", bytes.NewBufferString(`{"question":"VPN?"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	harness.handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
}

type aiTestHarness struct {
	handler       *stdhttp.ServeMux
	tokenManager  identity.TokenManager
	ticketService *ticket.Service
}

func newTestAIMux(t *testing.T, chunks []knowledge.DocumentChunk, minConfidence float64) aiTestHarness {
	t.Helper()

	return newTestAIMuxWithConfig(t, chunks, minConfidence, config.Config{
		AppName:  "zld-supportpilot-test",
		AppEnv:   "test",
		HTTPAddr: ":0",
	})
}

func newTestAIMuxWithConfig(t *testing.T, chunks []knowledge.DocumentChunk, minConfidence float64, cfg config.Config) aiTestHarness {
	t.Helper()

	if cfg.AppName == "" {
		cfg.AppName = "zld-supportpilot-test"
	}
	if cfg.AppEnv == "" {
		cfg.AppEnv = "test"
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":0"
	}

	ticketService := ticket.NewService(
		ticket.NewMemoryTicketRepository(),
		ticket.WithCollaborationDependencies(ticket.CollaborationDependencies{
			CommentRepository: ticket.NewMemoryTicketCommentRepository(),
			AuditRepository:   ticket.NewMemoryTicketAuditEventRepository(),
		}),
	)
	aiService := ai.NewService(ai.ServiceDependencies{
		ChunkSource:     aiTestChunkSource{chunks: chunks},
		Retriever:       ai.NewVectorRetriever(ai.NewHashingEmbedder(64)),
		AnswerGenerator: ai.TemplateAnswerGenerator{},
		MinConfidence:   minConfidence,
		TicketWorkspace: ticketService,
		TicketAnalyzer:  ai.TemplateTicketAnalyzer{},
	})
	tokenManager := identity.NewTokenManager("test-signing-key", time.Hour)

	return aiTestHarness{
		handler: NewMuxWithRouteDependencies(cfg, RouteDependencies{
			Auth: &AuthDependencies{
				TokenManager: tokenManager,
			},
			Ticket: &TicketDependencies{
				TicketService: ticketService,
			},
			AI: &AIDependencies{
				AIService: aiService,
			},
		}),
		tokenManager:  tokenManager,
		ticketService: ticketService,
	}
}

type aiTestChunkSource struct {
	chunks []knowledge.DocumentChunk
}

func (source aiTestChunkSource) ListIndexedChunks(identity.IdentityContext, string) ([]knowledge.DocumentChunk, error) {
	return source.chunks, nil
}
