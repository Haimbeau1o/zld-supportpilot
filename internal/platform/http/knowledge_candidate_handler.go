package http

import (
	"encoding/json"
	stdhttp "net/http"
	"strings"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/knowledge"
)

type createKnowledgeCandidateRequest struct {
	TicketID        string `json:"ticket_id"`
	KnowledgeBaseID string `json:"knowledge_base_id"`
	Title           string `json:"title"`
	Summary         string `json:"summary"`
	Content         string `json:"content"`
}

type reviewKnowledgeCandidateRequest struct {
	Action  knowledge.ReviewKnowledgeCandidateAction `json:"action"`
	Comment string                                   `json:"comment"`
}

type knowledgeCandidateResponse struct {
	ID                 string                             `json:"id"`
	KnowledgeBaseID    string                             `json:"knowledge_base_id"`
	OrganizationID     string                             `json:"organization_id"`
	SourceTicketID     string                             `json:"source_ticket_id"`
	Title              string                             `json:"title"`
	Summary            string                             `json:"summary"`
	Content            string                             `json:"content"`
	Status             knowledge.KnowledgeCandidateStatus `json:"status"`
	CreatedBy          string                             `json:"created_by"`
	ReviewedBy         string                             `json:"reviewed_by,omitempty"`
	ReviewComment      string                             `json:"review_comment,omitempty"`
	ApprovedDocumentID string                             `json:"approved_document_id,omitempty"`
}

type knowledgeCandidateListResponse struct {
	Items []knowledgeCandidateResponse `json:"items"`
}

func (handler KnowledgeHandler) CreateKnowledgeCandidate(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	actor, ok := identityContextFromContext(request.Context())
	if !ok {
		writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "当前请求缺少身份上下文")
		return
	}

	var input createKnowledgeCandidateRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		writeError(writer, stdhttp.StatusBadRequest, "invalid_request", "请求体不是合法的 JSON")
		return
	}

	candidate, err := handler.knowledgeService.CreateKnowledgeCandidateFromTicket(actor, knowledge.CreateKnowledgeCandidateFromTicketInput{
		TicketID:        input.TicketID,
		KnowledgeBaseID: input.KnowledgeBaseID,
		Title:           input.Title,
		Summary:         input.Summary,
		Content:         input.Content,
	})
	if err != nil {
		handleKnowledgeError(writer, err, "创建知识候选失败")
		return
	}

	writeJSON(writer, stdhttp.StatusCreated, newKnowledgeCandidateResponse(candidate))
}

func (handler KnowledgeHandler) ListKnowledgeCandidates(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	actor, ok := identityContextFromContext(request.Context())
	if !ok {
		writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "当前请求缺少身份上下文")
		return
	}

	candidates, err := handler.knowledgeService.ListKnowledgeCandidates(actor, knowledge.ListKnowledgeCandidatesInput{
		KnowledgeBaseID: strings.TrimSpace(request.URL.Query().Get("knowledge_base_id")),
	})
	if err != nil {
		handleKnowledgeError(writer, err, "查询知识候选失败")
		return
	}

	items := make([]knowledgeCandidateResponse, 0, len(candidates))
	for _, item := range candidates {
		items = append(items, newKnowledgeCandidateResponse(item))
	}

	writeJSON(writer, stdhttp.StatusOK, knowledgeCandidateListResponse{Items: items})
}

func (handler KnowledgeHandler) ReviewKnowledgeCandidate(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	actor, ok := identityContextFromContext(request.Context())
	if !ok {
		writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "当前请求缺少身份上下文")
		return
	}

	var input reviewKnowledgeCandidateRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		writeError(writer, stdhttp.StatusBadRequest, "invalid_request", "请求体不是合法的 JSON")
		return
	}

	candidate, err := handler.knowledgeService.ReviewKnowledgeCandidate(actor, knowledge.ReviewKnowledgeCandidateInput{
		CandidateID: strings.TrimSpace(request.PathValue("id")),
		Action:      input.Action,
		Comment:     input.Comment,
	})
	if err != nil {
		handleKnowledgeError(writer, err, "审核知识候选失败")
		return
	}

	writeJSON(writer, stdhttp.StatusOK, newKnowledgeCandidateResponse(candidate))
}

func newKnowledgeCandidateResponse(candidate knowledge.KnowledgeCandidate) knowledgeCandidateResponse {
	return knowledgeCandidateResponse{
		ID:                 candidate.ID,
		KnowledgeBaseID:    candidate.KnowledgeBaseID,
		OrganizationID:     candidate.OrganizationID,
		SourceTicketID:     candidate.SourceTicketID,
		Title:              candidate.Title,
		Summary:            candidate.Summary,
		Content:            candidate.Content,
		Status:             candidate.Status,
		CreatedBy:          candidate.CreatedBy,
		ReviewedBy:         candidate.ReviewedBy,
		ReviewComment:      candidate.ReviewComment,
		ApprovedDocumentID: candidate.ApprovedDocumentID,
	}
}
