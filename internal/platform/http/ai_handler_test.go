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

func TestAIAnswerQuestion(t *testing.T) {
	handler, tokenManager := newTestAIMux(t, []knowledge.DocumentChunk{{ID: "chunk-1", DocumentID: "doc-1", KnowledgeBaseID: "kb-1", Content: "VPN 无法连接时，请先重置客户端。"}}, 0.15)
	agent := identity.IdentityContext{UserID: "user-agent-1", OrganizationID: "org-1", Role: identity.RoleAgent, Permissions: identity.RolePermissions(identity.RoleAgent)}

	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/ai/knowledge/bases/kb-1/answers", bytes.NewBufferString(`{"question":"VPN 无法连接怎么办？","top_k":2}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+issueKnowledgeToken(t, tokenManager, agent))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

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
	handler, tokenManager := newTestAIMux(t, []knowledge.DocumentChunk{{ID: "chunk-1", DocumentID: "doc-1", KnowledgeBaseID: "kb-1", Content: "薪资审批需要走人力系统。"}}, 0.25)
	agent := identity.IdentityContext{UserID: "user-agent-1", OrganizationID: "org-1", Role: identity.RoleAgent, Permissions: identity.RolePermissions(identity.RoleAgent)}

	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/ai/knowledge/bases/kb-1/answers", bytes.NewBufferString(`{"question":"VPN 无法连接怎么办？","top_k":2}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+issueKnowledgeToken(t, tokenManager, agent))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

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

func TestAIRequiresAuthentication(t *testing.T) {
	handler, _ := newTestAIMux(t, nil, 0.15)
	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/ai/knowledge/bases/kb-1/answers", bytes.NewBufferString(`{"question":"VPN?"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
}

func newTestAIMux(t *testing.T, chunks []knowledge.DocumentChunk, minConfidence float64) (*stdhttp.ServeMux, identity.TokenManager) {
	t.Helper()

	aiService := ai.NewService(ai.ServiceDependencies{
		ChunkSource:     aiTestChunkSource{chunks: chunks},
		Retriever:       ai.NewVectorRetriever(ai.NewHashingEmbedder(64)),
		AnswerGenerator: ai.TemplateAnswerGenerator{},
		MinConfidence:   minConfidence,
	})
	tokenManager := identity.NewTokenManager("test-signing-key", time.Hour)

	return NewMuxWithRouteDependencies(config.Config{
		AppName:  "zld-supportpilot-test",
		AppEnv:   "test",
		HTTPAddr: ":0",
	}, RouteDependencies{
		Auth: &AuthDependencies{
			TokenManager: tokenManager,
		},
		AI: &AIDependencies{
			AIService: aiService,
		},
	}), tokenManager
}

type aiTestChunkSource struct {
	chunks []knowledge.DocumentChunk
}

func (source aiTestChunkSource) ListIndexedChunks(identity.IdentityContext, string) ([]knowledge.DocumentChunk, error) {
	return source.chunks, nil
}
