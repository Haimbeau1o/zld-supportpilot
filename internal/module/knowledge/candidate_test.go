package knowledge

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ticket"
)

func TestCreateKnowledgeCandidateFromResolvedTicket(t *testing.T) {
	agent := identity.IdentityContext{
		UserID:         "agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}

	service := NewService(
		NewMemoryKnowledgeBaseRepository(),
		NewMemoryDocumentRepository(),
		NewMemoryObjectStorage(),
		WithCandidateWorkflow(CandidateWorkflowDependencies{
			CandidateRepository: NewMemoryKnowledgeCandidateRepository(),
			TicketSource: stubKnowledgeCandidateTicketSource{ticket: ticket.Ticket{
				ID:             "ticket-1",
				OrganizationID: "org-1",
				RequesterID:    "user-1",
				Title:          "VPN 无法连接",
				Description:    "错误码 691",
				Status:         ticket.TicketStatusResolved,
			}},
		}),
	)

	knowledgeBase, err := service.CreateKnowledgeBase(agent, CreateKnowledgeBaseInput{Name: "IT 支持知识库"})
	if err != nil {
		t.Fatalf("create knowledge base: %v", err)
	}

	candidate, err := service.CreateKnowledgeCandidateFromTicket(agent, CreateKnowledgeCandidateFromTicketInput{
		TicketID:        "ticket-1",
		KnowledgeBaseID: knowledgeBase.ID,
		Title:           "VPN 691 错误排查",
		Summary:         "通过重置账号拨号权限恢复",
		Content:         "处理步骤：检查账号状态，重置拨号权限，重新连接验证。",
	})
	if err != nil {
		t.Fatalf("create knowledge candidate: %v", err)
	}
	if candidate.Status != KnowledgeCandidateStatusPendingReview {
		t.Fatalf("expected candidate status %q, got %q", KnowledgeCandidateStatusPendingReview, candidate.Status)
	}
	if candidate.SourceTicketID != "ticket-1" {
		t.Fatalf("expected source ticket id %q, got %q", "ticket-1", candidate.SourceTicketID)
	}
	if candidate.ID == "" {
		t.Fatalf("expected candidate id to be generated")
	}
}

func TestCreateKnowledgeCandidateRejectsNonResolvedTicket(t *testing.T) {
	agent := identity.IdentityContext{
		UserID:         "agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}

	service := NewService(
		NewMemoryKnowledgeBaseRepository(),
		NewMemoryDocumentRepository(),
		NewMemoryObjectStorage(),
		WithCandidateWorkflow(CandidateWorkflowDependencies{
			CandidateRepository: NewMemoryKnowledgeCandidateRepository(),
			TicketSource: stubKnowledgeCandidateTicketSource{ticket: ticket.Ticket{
				ID:             "ticket-1",
				OrganizationID: "org-1",
				RequesterID:    "user-1",
				Title:          "VPN 无法连接",
				Description:    "错误码 691",
				Status:         ticket.TicketStatusInProgress,
			}},
		}),
	)

	knowledgeBase, err := service.CreateKnowledgeBase(agent, CreateKnowledgeBaseInput{Name: "IT 支持知识库"})
	if err != nil {
		t.Fatalf("create knowledge base: %v", err)
	}

	_, err = service.CreateKnowledgeCandidateFromTicket(agent, CreateKnowledgeCandidateFromTicketInput{
		TicketID:        "ticket-1",
		KnowledgeBaseID: knowledgeBase.ID,
		Title:           "VPN 691 错误排查",
		Summary:         "通过重置账号拨号权限恢复",
		Content:         "处理步骤：检查账号状态，重置拨号权限，重新连接验证。",
	})
	if !errors.Is(err, ErrInvalidKnowledgeCandidateInput) {
		t.Fatalf("expected invalid knowledge candidate input, got %v", err)
	}
}

func TestReviewKnowledgeCandidateApproveCreatesDocumentAndProcessingTask(t *testing.T) {
	agent := identity.IdentityContext{
		UserID:         "agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}

	taskRepository := NewMemoryDocumentProcessingTaskRepository()
	chunkRepository := NewMemoryDocumentChunkRepository()
	dispatcher := &stubProcessingTaskDispatcher{}
	service := NewService(
		NewMemoryKnowledgeBaseRepository(),
		NewMemoryDocumentRepository(),
		NewMemoryObjectStorage(),
		WithProcessingPipeline(ProcessingDependencies{
			TaskRepository:  taskRepository,
			ChunkRepository: chunkRepository,
			Dispatcher:      dispatcher,
			Parser:          stubDocumentParser{},
			Chunker:         stubDocumentChunker{},
			Indexer:         stubChunkIndexer{repository: chunkRepository},
			MaxAttempts:     3,
		}),
		WithCandidateWorkflow(CandidateWorkflowDependencies{
			CandidateRepository: NewMemoryKnowledgeCandidateRepository(),
			TicketSource: stubKnowledgeCandidateTicketSource{ticket: ticket.Ticket{
				ID:             "ticket-1",
				OrganizationID: "org-1",
				RequesterID:    "user-1",
				Title:          "VPN 无法连接",
				Description:    "错误码 691",
				Status:         ticket.TicketStatusResolved,
			}},
		}),
	)

	knowledgeBase, err := service.CreateKnowledgeBase(agent, CreateKnowledgeBaseInput{Name: "IT 支持知识库"})
	if err != nil {
		t.Fatalf("create knowledge base: %v", err)
	}

	candidate, err := service.CreateKnowledgeCandidateFromTicket(agent, CreateKnowledgeCandidateFromTicketInput{
		TicketID:        "ticket-1",
		KnowledgeBaseID: knowledgeBase.ID,
		Title:           "VPN 691 错误排查",
		Summary:         "通过重置账号拨号权限恢复",
		Content:         "处理步骤：检查账号状态，重置拨号权限，重新连接验证。",
	})
	if err != nil {
		t.Fatalf("create knowledge candidate: %v", err)
	}

	reviewedCandidate, err := service.ReviewKnowledgeCandidate(agent, ReviewKnowledgeCandidateInput{
		CandidateID: candidate.ID,
		Action:      ReviewKnowledgeCandidateActionApprove,
		Comment:     "内容可入库",
	})
	if err != nil {
		t.Fatalf("review knowledge candidate: %v", err)
	}
	if reviewedCandidate.Status != KnowledgeCandidateStatusApproved {
		t.Fatalf("expected candidate status %q, got %q", KnowledgeCandidateStatusApproved, reviewedCandidate.Status)
	}
	if reviewedCandidate.ApprovedDocumentID == "" {
		t.Fatalf("expected approved document id to be returned")
	}
	if len(dispatcher.enqueuedTaskIDs) != 1 {
		t.Fatalf("expected one processing task to be enqueued, got %d", len(dispatcher.enqueuedTaskIDs))
	}

	document, err := service.GetDocument(agent, reviewedCandidate.ApprovedDocumentID)
	if err != nil {
		t.Fatalf("get approved document: %v", err)
	}
	if document.LastTaskID == "" {
		t.Fatalf("expected approved document to have processing task")
	}
	if _, err := service.ProcessTask(context.Background(), document.LastTaskID); err != nil {
		t.Fatalf("process approved document task: %v", err)
	}

	chunks, err := service.ListIndexedChunks(agent, knowledgeBase.ID)
	if err != nil {
		t.Fatalf("list indexed chunks: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatalf("expected approved knowledge candidate to produce indexed chunks")
	}
	if !strings.Contains(chunks[0].Content, "重置账号拨号权限") {
		t.Fatalf("expected chunk content to contain approved candidate content, got %q", chunks[0].Content)
	}
}

func TestReviewKnowledgeCandidateRejectDoesNotCreateDocument(t *testing.T) {
	agent := identity.IdentityContext{
		UserID:         "agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}

	service := NewService(
		NewMemoryKnowledgeBaseRepository(),
		NewMemoryDocumentRepository(),
		NewMemoryObjectStorage(),
		WithCandidateWorkflow(CandidateWorkflowDependencies{
			CandidateRepository: NewMemoryKnowledgeCandidateRepository(),
			TicketSource: stubKnowledgeCandidateTicketSource{ticket: ticket.Ticket{
				ID:             "ticket-1",
				OrganizationID: "org-1",
				RequesterID:    "user-1",
				Title:          "VPN 无法连接",
				Description:    "错误码 691",
				Status:         ticket.TicketStatusResolved,
			}},
		}),
	)

	knowledgeBase, err := service.CreateKnowledgeBase(agent, CreateKnowledgeBaseInput{Name: "IT 支持知识库"})
	if err != nil {
		t.Fatalf("create knowledge base: %v", err)
	}

	candidate, err := service.CreateKnowledgeCandidateFromTicket(agent, CreateKnowledgeCandidateFromTicketInput{
		TicketID:        "ticket-1",
		KnowledgeBaseID: knowledgeBase.ID,
		Title:           "VPN 691 错误排查",
		Summary:         "通过重置账号拨号权限恢复",
		Content:         "处理步骤：检查账号状态，重置拨号权限，重新连接验证。",
	})
	if err != nil {
		t.Fatalf("create knowledge candidate: %v", err)
	}

	reviewedCandidate, err := service.ReviewKnowledgeCandidate(agent, ReviewKnowledgeCandidateInput{
		CandidateID: candidate.ID,
		Action:      ReviewKnowledgeCandidateActionReject,
		Comment:     "内容还不够完整",
	})
	if err != nil {
		t.Fatalf("review knowledge candidate: %v", err)
	}
	if reviewedCandidate.Status != KnowledgeCandidateStatusRejected {
		t.Fatalf("expected candidate status %q, got %q", KnowledgeCandidateStatusRejected, reviewedCandidate.Status)
	}
	if reviewedCandidate.ApprovedDocumentID != "" {
		t.Fatalf("expected no approved document id, got %q", reviewedCandidate.ApprovedDocumentID)
	}

	documents, err := service.ListDocuments(agent, ListDocumentsInput{KnowledgeBaseID: knowledgeBase.ID})
	if err != nil {
		t.Fatalf("list documents: %v", err)
	}
	if len(documents) != 0 {
		t.Fatalf("expected no documents to be created on reject, got %d", len(documents))
	}
}

type stubKnowledgeCandidateTicketSource struct {
	ticket ticket.Ticket
	err   error
}

func (source stubKnowledgeCandidateTicketSource) GetTicket(identity.IdentityContext, string) (ticket.Ticket, error) {
	if source.err != nil {
		return ticket.Ticket{}, source.err
	}
	return source.ticket, nil
}
