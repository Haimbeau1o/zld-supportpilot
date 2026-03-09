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
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ticket"
)

type aiService interface {
	AskKnowledgeQuestion(ctx context.Context, actor identity.IdentityContext, input ai.AskKnowledgeQuestionInput) (ai.AnswerResult, error)
	HandleUnifiedIntake(ctx context.Context, actor identity.IdentityContext, input ai.UnifiedIntakeInput) (ai.UnifiedIntakeResult, error)
	SubmitAnswerFeedback(ctx context.Context, actor identity.IdentityContext, input ai.AnswerFeedbackInput) (ai.AnswerFeedbackResult, error)
	GetAnswerFeedbackStats(ctx context.Context, actor identity.IdentityContext) (ai.AnswerFeedbackStats, error)
	GenerateTicketAssist(ctx context.Context, actor identity.IdentityContext, input ai.GenerateTicketAssistInput) (ai.TicketAssistResult, error)
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

type unifiedIntakeRequest struct {
	KnowledgeBaseID string `json:"knowledge_base_id"`
	Question        string `json:"question"`
	TopK            int    `json:"top_k"`
}

type answerFeedbackRequest struct {
	Status  ai.AnswerFeedbackStatus `json:"status"`
	Comment string                  `json:"comment"`
}

type generateTicketAssistRequest struct {
	KnowledgeBaseID string `json:"knowledge_base_id"`
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

type aiUnifiedIntakeResponse struct {
	IntakeID     string                     `json:"intake_id"`
	ResultType   ai.UnifiedIntakeResultType `json:"result_type"`
	AnswerStatus ai.AnswerStatus            `json:"answer_status"`
	Answer       string                     `json:"answer"`
	Confidence   float64                    `json:"confidence"`
	Citations    []aiCitationResponse       `json:"citations"`
	TicketID     string                     `json:"ticket_id,omitempty"`
}

type aiAnswerFeedbackResponse struct {
	FeedbackID   string                        `json:"feedback_id"`
	IntakeID     string                        `json:"intake_id"`
	Status       ai.AnswerFeedbackStatus       `json:"status"`
	TicketAction ai.AnswerFeedbackTicketAction `json:"ticket_action"`
	TicketID     string                        `json:"ticket_id,omitempty"`
}

type aiAnswerFeedbackStatsResponse struct {
	Total      int `json:"total"`
	Resolved   int `json:"resolved"`
	Unresolved int `json:"unresolved"`
	Inaccurate int `json:"inaccurate"`
}

type aiTicketAssistResponse struct {
	CategorySuggestion string               `json:"category_suggestion"`
	Summary            string               `json:"summary"`
	ReplyDraft         string               `json:"reply_draft"`
	AnswerStatus       ai.AnswerStatus      `json:"answer_status"`
	Confidence         float64              `json:"confidence"`
	Citations          []aiCitationResponse `json:"citations"`
	RecordedCommentID  string               `json:"recorded_comment_id"`
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

func (handler AIHandler) HandleUnifiedIntake(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	actor, ok := identityContextFromContext(request.Context())
	if !ok {
		writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "当前请求缺少身份上下文")
		return
	}

	var input unifiedIntakeRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		writeError(writer, stdhttp.StatusBadRequest, "invalid_request", "请求体不是合法的 JSON")
		return
	}

	result, err := handler.aiService.HandleUnifiedIntake(request.Context(), actor, ai.UnifiedIntakeInput{
		KnowledgeBaseID: strings.TrimSpace(input.KnowledgeBaseID),
		Question:        input.Question,
		TopK:            input.TopK,
	})
	if err != nil {
		handleAIError(writer, err)
		return
	}

	writeJSON(writer, stdhttp.StatusOK, newAIUnifiedIntakeResponse(result))
}

func (handler AIHandler) SubmitAnswerFeedback(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	actor, ok := identityContextFromContext(request.Context())
	if !ok {
		writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "当前请求缺少身份上下文")
		return
	}

	var input answerFeedbackRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		writeError(writer, stdhttp.StatusBadRequest, "invalid_request", "请求体不是合法的 JSON")
		return
	}

	result, err := handler.aiService.SubmitAnswerFeedback(request.Context(), actor, ai.AnswerFeedbackInput{
		IntakeID: strings.TrimSpace(request.PathValue("id")),
		Status:   input.Status,
		Comment:  input.Comment,
	})
	if err != nil {
		handleAIError(writer, err)
		return
	}

	writeJSON(writer, stdhttp.StatusOK, newAIAnswerFeedbackResponse(result))
}

func (handler AIHandler) GetAnswerFeedbackStats(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	actor, ok := identityContextFromContext(request.Context())
	if !ok {
		writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "当前请求缺少身份上下文")
		return
	}

	result, err := handler.aiService.GetAnswerFeedbackStats(request.Context(), actor)
	if err != nil {
		handleAIError(writer, err)
		return
	}

	writeJSON(writer, stdhttp.StatusOK, newAIAnswerFeedbackStatsResponse(result))
}

func (handler AIHandler) GenerateTicketAssist(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	actor, ok := identityContextFromContext(request.Context())
	if !ok {
		writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "当前请求缺少身份上下文")
		return
	}

	var input generateTicketAssistRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		writeError(writer, stdhttp.StatusBadRequest, "invalid_request", "请求体不是合法的 JSON")
		return
	}

	result, err := handler.aiService.GenerateTicketAssist(request.Context(), actor, ai.GenerateTicketAssistInput{
		TicketID:        strings.TrimSpace(request.PathValue("id")),
		KnowledgeBaseID: strings.TrimSpace(input.KnowledgeBaseID),
	})
	if err != nil {
		handleAIError(writer, err)
		return
	}

	writeJSON(writer, stdhttp.StatusOK, newAITicketAssistResponse(result))
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

func newAIUnifiedIntakeResponse(result ai.UnifiedIntakeResult) aiUnifiedIntakeResponse {
	citations := make([]aiCitationResponse, 0, len(result.Citations))
	for _, citation := range result.Citations {
		citations = append(citations, aiCitationResponse{
			ChunkID:    citation.ChunkID,
			DocumentID: citation.DocumentID,
			Score:      citation.Score,
			Content:    citation.Content,
		})
	}

	return aiUnifiedIntakeResponse{
		IntakeID:     result.IntakeID,
		ResultType:   result.ResultType,
		AnswerStatus: result.AnswerStatus,
		Answer:       result.Answer,
		Confidence:   result.Confidence,
		Citations:    citations,
		TicketID:     result.TicketID,
	}
}

func newAIAnswerFeedbackResponse(result ai.AnswerFeedbackResult) aiAnswerFeedbackResponse {
	return aiAnswerFeedbackResponse{
		FeedbackID:   result.FeedbackID,
		IntakeID:     result.IntakeID,
		Status:       result.Status,
		TicketAction: result.TicketAction,
		TicketID:     result.TicketID,
	}
}

func newAIAnswerFeedbackStatsResponse(result ai.AnswerFeedbackStats) aiAnswerFeedbackStatsResponse {
	return aiAnswerFeedbackStatsResponse{
		Total:      result.Total,
		Resolved:   result.Resolved,
		Unresolved: result.Unresolved,
		Inaccurate: result.Inaccurate,
	}
}

func newAITicketAssistResponse(result ai.TicketAssistResult) aiTicketAssistResponse {
	citations := make([]aiCitationResponse, 0, len(result.Citations))
	for _, citation := range result.Citations {
		citations = append(citations, aiCitationResponse{
			ChunkID:    citation.ChunkID,
			DocumentID: citation.DocumentID,
			Score:      citation.Score,
			Content:    citation.Content,
		})
	}

	return aiTicketAssistResponse{
		CategorySuggestion: result.CategorySuggestion,
		Summary:            result.Summary,
		ReplyDraft:         result.ReplyDraft,
		AnswerStatus:       result.AnswerStatus,
		Confidence:         result.Confidence,
		Citations:          citations,
		RecordedCommentID:  result.RecordedCommentID,
	}
}

func handleAIError(writer stdhttp.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ai.ErrInvalidAIInput):
		writeError(writer, stdhttp.StatusBadRequest, "invalid_ai_request", err.Error())
	case errors.Is(err, ticket.ErrTicketForbidden), errors.Is(err, knowledge.ErrKnowledgeForbidden):
		writeError(writer, stdhttp.StatusForbidden, "forbidden", "当前身份没有权限执行该 AI 操作")
	case errors.Is(err, ai.ErrUnifiedIntakeNotFound):
		writeError(writer, stdhttp.StatusNotFound, "intake_not_found", "AI 受理记录不存在")
	case errors.Is(err, ticket.ErrTicketNotFound):
		writeError(writer, stdhttp.StatusNotFound, "ticket_not_found", "工单不存在")
	case errors.Is(err, knowledge.ErrKnowledgeBaseNotFound):
		writeError(writer, stdhttp.StatusNotFound, "knowledge_base_not_found", "知识库不存在")
	case errors.Is(err, knowledge.ErrKnowledgeProcessingDisabled):
		writeError(writer, stdhttp.StatusServiceUnavailable, "knowledge_processing_unavailable", "当前知识库尚未准备好供检索使用")
	default:
		writeError(writer, stdhttp.StatusInternalServerError, "internal_error", "AI 能力执行失败")
	}
}

func registerAIRoutes(mux *stdhttp.ServeMux, runtime routeRuntime, authDependencies AuthDependencies, aiDependencies AIDependencies) error {
	if authDependencies.TokenManager == nil {
		return fmt.Errorf("token manager is required for ai routes")
	}
	if aiDependencies.AIService == nil {
		return fmt.Errorf("ai service is required")
	}

	authMiddleware := NewAuthMiddleware(authDependencies.TokenManager)
	aiHandler := NewAIHandler(aiDependencies.AIService)

	// AI 接口统一运行在鉴权之后，保证知识库与工单的租户归属边界不被绕开。
	// AI 问答、统一受理、反馈回流与工单辅助都属于高成本能力，因此当前阶段统一纳入限流边界，避免演示环境被连续请求打爆。
	mux.Handle("POST /api/v1/ai/knowledge/bases/{id}/answers", runtime.protectedRateLimited("POST /api/v1/ai/knowledge/bases/{id}/answers", authMiddleware, identityOrClientRateLimitKey, stdhttp.HandlerFunc(aiHandler.AskKnowledgeQuestion)))
	mux.Handle("POST /api/v1/ai/intakes", runtime.protectedRateLimited("POST /api/v1/ai/intakes", authMiddleware, identityOrClientRateLimitKey, stdhttp.HandlerFunc(aiHandler.HandleUnifiedIntake)))
	mux.Handle("POST /api/v1/ai/intakes/{id}/feedback", runtime.protectedRateLimited("POST /api/v1/ai/intakes/{id}/feedback", authMiddleware, identityOrClientRateLimitKey, stdhttp.HandlerFunc(aiHandler.SubmitAnswerFeedback)))
	mux.Handle("GET /api/v1/ai/feedback/stats", runtime.protectedRateLimited("GET /api/v1/ai/feedback/stats", authMiddleware, identityOrClientRateLimitKey, stdhttp.HandlerFunc(aiHandler.GetAnswerFeedbackStats)))
	mux.Handle("POST /api/v1/ai/tickets/{id}/assist", runtime.protectedRateLimited("POST /api/v1/ai/tickets/{id}/assist", authMiddleware, identityOrClientRateLimitKey, stdhttp.HandlerFunc(aiHandler.GenerateTicketAssist)))
	return nil
}
