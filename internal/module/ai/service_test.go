package ai

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/knowledge"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ticket"
)

func TestAskKnowledgeQuestionAnswered(t *testing.T) {
	service := NewService(ServiceDependencies{
		ChunkSource: stubChunkSource{chunks: []knowledge.DocumentChunk{
			{ID: "chunk-1", DocumentID: "doc-1", KnowledgeBaseID: "kb-1", Content: "VPN 无法连接时，请先重置客户端，然后重新登录。"},
			{ID: "chunk-2", DocumentID: "doc-2", KnowledgeBaseID: "kb-1", Content: "如果重置后仍失败，请联系网络管理员。"},
		}},
		Retriever:       NewVectorRetriever(NewHashingEmbedder(64)),
		AnswerGenerator: TemplateAnswerGenerator{},
		MinConfidence:   0.15,
	})

	result, err := service.AskKnowledgeQuestion(context.Background(), identity.IdentityContext{OrganizationID: "org-1"}, AskKnowledgeQuestionInput{
		KnowledgeBaseID: "kb-1",
		Question:        "VPN 无法连接怎么办？",
		TopK:            2,
	})
	if err != nil {
		t.Fatalf("ask knowledge question: %v", err)
	}
	if result.Status != AnswerStatusAnswered {
		t.Fatalf("expected answered status, got %q", result.Status)
	}
	if len(result.Citations) == 0 {
		t.Fatalf("expected citations to be returned")
	}
	if !strings.Contains(result.Answer, "[1]") {
		t.Fatalf("expected answer to contain citation marker, got %q", result.Answer)
	}
	if result.Confidence <= 0 {
		t.Fatalf("expected positive confidence, got %f", result.Confidence)
	}
}

func TestAskKnowledgeQuestionDegradedWhenLowConfidence(t *testing.T) {
	service := NewService(ServiceDependencies{
		ChunkSource: stubChunkSource{chunks: []knowledge.DocumentChunk{
			{ID: "chunk-1", DocumentID: "doc-1", KnowledgeBaseID: "kb-1", Content: "人力系统薪资审批在每月 25 日截止。"},
		}},
		Retriever:       NewVectorRetriever(NewHashingEmbedder(64)),
		AnswerGenerator: TemplateAnswerGenerator{},
		MinConfidence:   0.25,
	})

	result, err := service.AskKnowledgeQuestion(context.Background(), identity.IdentityContext{OrganizationID: "org-1"}, AskKnowledgeQuestionInput{
		KnowledgeBaseID: "kb-1",
		Question:        "VPN 无法连接怎么办？",
		TopK:            2,
	})
	if err != nil {
		t.Fatalf("ask knowledge question: %v", err)
	}
	if result.Status != AnswerStatusDegraded {
		t.Fatalf("expected degraded status, got %q", result.Status)
	}
	if result.Answer == "" {
		t.Fatalf("expected degraded answer message")
	}
}

func TestAskKnowledgeQuestionPropagatesChunkSourceError(t *testing.T) {
	service := NewService(ServiceDependencies{
		ChunkSource:     stubChunkSource{err: knowledge.ErrKnowledgeForbidden},
		Retriever:       NewVectorRetriever(NewHashingEmbedder(64)),
		AnswerGenerator: TemplateAnswerGenerator{},
		MinConfidence:   0.15,
	})

	_, err := service.AskKnowledgeQuestion(context.Background(), identity.IdentityContext{OrganizationID: "org-1"}, AskKnowledgeQuestionInput{
		KnowledgeBaseID: "kb-1",
		Question:        "VPN 无法连接怎么办？",
		TopK:            2,
	})
	if !errors.Is(err, knowledge.ErrKnowledgeForbidden) {
		t.Fatalf("expected knowledge forbidden error, got %v", err)
	}
}

func TestHandleUnifiedIntakeReturnsAnswerWhenKnowledgeIsConfident(t *testing.T) {
	endUser := identity.IdentityContext{
		UserID:         "user-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}
	workspace := &stubTicketWorkspace{}
	service := NewService(ServiceDependencies{
		TicketWorkspace: workspace,
		KnowledgeAnswerer: stubKnowledgeAnswerer{result: AnswerResult{
			Status:     AnswerStatusAnswered,
			Answer:     "根据知识库资料，建议先重置 VPN 客户端。",
			Confidence: 0.82,
			Citations: []Citation{{
				ChunkID:    "chunk-1",
				DocumentID: "doc-1",
				Score:      0.82,
				Content:    "VPN 无法连接时，请先重置客户端。",
			}},
		}},
	})

	result, err := service.HandleUnifiedIntake(context.Background(), endUser, UnifiedIntakeInput{
		KnowledgeBaseID: "kb-1",
		Question:        "VPN 无法连接怎么办？",
		TopK:            3,
	})
	if err != nil {
		t.Fatalf("handle unified intake: %v", err)
	}
	if result.ResultType != UnifiedIntakeResultTypeAnswered {
		t.Fatalf("expected result type %q, got %q", UnifiedIntakeResultTypeAnswered, result.ResultType)
	}
	if result.TicketID != "" {
		t.Fatalf("expected no ticket id when directly answered, got %q", result.TicketID)
	}
	if workspace.createdTicketInput.Title != "" {
		t.Fatalf("expected no ticket to be created when answered")
	}
}

func TestHandleUnifiedIntakeCreatesTicketWhenKnowledgeIsDegraded(t *testing.T) {
	endUser := identity.IdentityContext{
		UserID:         "user-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}
	workspace := &stubTicketWorkspace{
		createdTicket: ticket.Ticket{ID: "ticket-1", OrganizationID: "org-1", RequesterID: "user-1", Title: "VPN 无法连接怎么办？", Status: ticket.TicketStatusOpen},
	}
	service := NewService(ServiceDependencies{
		TicketWorkspace: workspace,
		KnowledgeAnswerer: stubKnowledgeAnswerer{result: AnswerResult{
			Status:     AnswerStatusDegraded,
			Answer:     degradedAnswerResult().Answer,
			Confidence: 0,
			Citations:  []Citation{},
		}},
	})

	result, err := service.HandleUnifiedIntake(context.Background(), endUser, UnifiedIntakeInput{
		KnowledgeBaseID: "kb-1",
		Question:        "VPN 无法连接怎么办？",
		TopK:            3,
	})
	if err != nil {
		t.Fatalf("handle unified intake: %v", err)
	}
	if result.ResultType != UnifiedIntakeResultTypeTicketCreated {
		t.Fatalf("expected result type %q, got %q", UnifiedIntakeResultTypeTicketCreated, result.ResultType)
	}
	if result.TicketID != "ticket-1" {
		t.Fatalf("expected ticket id %q, got %q", "ticket-1", result.TicketID)
	}
	if workspace.createdTicketInput.Description != "VPN 无法连接怎么办？" {
		t.Fatalf("expected question to become ticket description, got %q", workspace.createdTicketInput.Description)
	}
	if workspace.createdTicketInput.Priority != ticket.TicketPriorityMedium {
		t.Fatalf("expected default intake priority %q, got %q", ticket.TicketPriorityMedium, workspace.createdTicketInput.Priority)
	}
}

func TestHandleUnifiedIntakeRejectsEmptyQuestion(t *testing.T) {
	endUser := identity.IdentityContext{OrganizationID: "org-1", Role: identity.RoleEndUser, Permissions: identity.RolePermissions(identity.RoleEndUser)}
	service := NewService(ServiceDependencies{
		TicketWorkspace:   &stubTicketWorkspace{},
		KnowledgeAnswerer: stubKnowledgeAnswerer{},
	})

	_, err := service.HandleUnifiedIntake(context.Background(), endUser, UnifiedIntakeInput{
		KnowledgeBaseID: "kb-1",
		Question:        "   ",
	})
	if !errors.Is(err, ErrInvalidAIInput) {
		t.Fatalf("expected invalid ai input error, got %v", err)
	}
}

type stubChunkSource struct {
	chunks []knowledge.DocumentChunk
	err    error
}

func (source stubChunkSource) ListIndexedChunks(identity.IdentityContext, string) ([]knowledge.DocumentChunk, error) {
	if source.err != nil {
		return nil, source.err
	}
	return source.chunks, nil
}
