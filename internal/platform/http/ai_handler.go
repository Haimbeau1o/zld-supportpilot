package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	stdhttp "net/http"
	"strings"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ai"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/knowledge"
)

type aiService interface {
	AskKnowledgeQuestion(ctx context.Context, actor identity.IdentityContext, input ai.AskKnowledgeQuestionInput) (ai.AnswerResult, error)
}

type AIDependencies struct {
	AIService aiService
}

type AIHandler struct {
	aiService aiService
}

type askKnowledgeQuestionRequest struct {
	Question string `json:"question"`
	TopK     int    `json:"top_k"`
}

type aiCitationResponse struct {
	ChunkID    string  `json:"chunk_id"`
	DocumentID string  `json:"document_id"`
	Score      float64 `json:"score"`
	Content    string  `json:"content"`
}

type aiAnswerResponse struct {
	Status     ai.AnswerStatus      `json:"status"`
	Answer     string               `json:"answer"`
	Confidence float64              `json:"confidence"`
	Citations  []aiCitationResponse `json:"citations"`
}

func NewAIHandler(aiService aiService) AIHandler {
	return AIHandler{aiService: aiService}
}

func (handler AIHandler) AskKnowledgeQuestion(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	actor, ok := identityContextFromContext(request.Context())
	if !ok {
		writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "当前请求缺少身份上下文")
		return
	}

	var input askKnowledgeQuestionRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		writeError(writer, stdhttp.StatusBadRequest, "invalid_request", "请求体不是合法的 JSON")
		return
	}

	result, err := handler.aiService.AskKnowledgeQuestion(request.Context(), actor, ai.AskKnowledgeQuestionInput{
		KnowledgeBaseID: strings.TrimSpace(request.PathValue("id")),
		Question:        input.Question,
		TopK:            input.TopK,
	})
	if err != nil {
		handleAIError(writer, err)
		return
	}

	writeJSON(writer, stdhttp.StatusOK, newAIAnswerResponse(result))
}

func newAIAnswerResponse(result ai.AnswerResult) aiAnswerResponse {
	citations := make([]aiCitationResponse, 0, len(result.Citations))
	for _, citation := range result.Citations {
		citations = append(citations, aiCitationResponse{
			ChunkID:    citation.ChunkID,
			DocumentID: citation.DocumentID,
			Score:      citation.Score,
			Content:    citation.Content,
		})
	}

	return aiAnswerResponse{
		Status:     result.Status,
		Answer:     result.Answer,
		Confidence: result.Confidence,
		Citations:  citations,
	}
}

func handleAIError(writer stdhttp.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ai.ErrInvalidAIInput):
		writeError(writer, stdhttp.StatusBadRequest, "invalid_ai_request", err.Error())
	case errors.Is(err, knowledge.ErrKnowledgeForbidden):
		writeError(writer, stdhttp.StatusForbidden, "forbidden", "当前身份没有权限执行该 AI 问答操作")
	case errors.Is(err, knowledge.ErrKnowledgeBaseNotFound):
		writeError(writer, stdhttp.StatusNotFound, "knowledge_base_not_found", "知识库不存在")
	case errors.Is(err, knowledge.ErrKnowledgeProcessingDisabled):
		writeError(writer, stdhttp.StatusServiceUnavailable, "knowledge_processing_unavailable", "当前知识库尚未准备好供检索使用")
	default:
		writeError(writer, stdhttp.StatusInternalServerError, "internal_error", "AI 问答失败")
	}
}

func registerAIRoutes(mux *stdhttp.ServeMux, authDependencies AuthDependencies, aiDependencies AIDependencies) error {
	if authDependencies.TokenManager == nil {
		return fmt.Errorf("token manager is required for ai routes")
	}
	if aiDependencies.AIService == nil {
		return fmt.Errorf("ai service is required")
	}

	authMiddleware := NewAuthMiddleware(authDependencies.TokenManager)
	aiHandler := NewAIHandler(aiDependencies.AIService)

	// AI 回答接口同样必须运行在鉴权之后，保证知识库归属和 AI 访问边界一致。
	mux.Handle("POST /api/v1/ai/knowledge/bases/{id}/answers", authMiddleware.RequireIdentity(stdhttp.HandlerFunc(aiHandler.AskKnowledgeQuestion)))
	return nil
}
