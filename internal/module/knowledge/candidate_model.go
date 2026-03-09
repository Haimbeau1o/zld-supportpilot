package knowledge

import (
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ticket"
)

// KnowledgeCandidateStatus 表示知识候选在审核流中的当前状态。
type KnowledgeCandidateStatus string

const (
	KnowledgeCandidateStatusPendingReview KnowledgeCandidateStatus = "pending_review"
	KnowledgeCandidateStatusApproved      KnowledgeCandidateStatus = "approved"
	KnowledgeCandidateStatusRejected      KnowledgeCandidateStatus = "rejected"
)

// ReviewKnowledgeCandidateAction 表示审核动作。
type ReviewKnowledgeCandidateAction string

const (
	ReviewKnowledgeCandidateActionApprove ReviewKnowledgeCandidateAction = "approve"
	ReviewKnowledgeCandidateActionReject  ReviewKnowledgeCandidateAction = "reject"
)

// KnowledgeCandidate 表示从已解决工单沉淀出的知识候选。
type KnowledgeCandidate struct {
	ID                 string
	KnowledgeBaseID    string
	OrganizationID     string
	SourceTicketID     string
	Title              string
	Summary            string
	Content            string
	Status             KnowledgeCandidateStatus
	CreatedBy          string
	ReviewedBy         string
	ReviewComment      string
	ApprovedDocumentID string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	ReviewedAt         time.Time
}

// CreateKnowledgeCandidateFromTicketInput 表示从工单创建知识候选的输入。
type CreateKnowledgeCandidateFromTicketInput struct {
	TicketID        string
	KnowledgeBaseID string
	Title           string
	Summary         string
	Content         string
}

// ListKnowledgeCandidatesInput 表示候选列表查询条件。
type ListKnowledgeCandidatesInput struct {
	KnowledgeBaseID string
}

// ReviewKnowledgeCandidateInput 表示候选审核输入。
type ReviewKnowledgeCandidateInput struct {
	CandidateID string
	Action      ReviewKnowledgeCandidateAction
	Comment     string
}

// TicketSource 抽象知识模块对工单读取能力的依赖，避免知识模块反向侵入工单实现细节。
type TicketSource interface {
	GetTicket(actor identity.IdentityContext, ticketID string) (ticket.Ticket, error)
}

func NewKnowledgeCandidate(candidate KnowledgeCandidate) KnowledgeCandidate {
	now := time.Now()
	if candidate.Status == "" {
		candidate.Status = KnowledgeCandidateStatusPendingReview
	}
	if candidate.CreatedAt.IsZero() {
		candidate.CreatedAt = now
	}
	if candidate.UpdatedAt.IsZero() {
		candidate.UpdatedAt = candidate.CreatedAt
	}
	return candidate
}
