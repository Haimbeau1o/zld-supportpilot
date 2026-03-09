package knowledge

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ticket"
)

var (
	ErrKnowledgeCandidateNotFound         = errors.New("knowledge candidate not found")
	ErrKnowledgeCandidateWorkflowDisabled = errors.New("knowledge candidate workflow disabled")
	ErrInvalidKnowledgeCandidateInput     = errors.New("invalid knowledge candidate input")
)

// CandidateWorkflowDependencies 表示知识候选审核流依赖。
type CandidateWorkflowDependencies struct {
	CandidateRepository KnowledgeCandidateRepository
	TicketSource        TicketSource
}

func WithCandidateWorkflow(dependencies CandidateWorkflowDependencies) ServiceOption {
	return func(service *Service) {
		service.candidateRepository = dependencies.CandidateRepository
		service.ticketSource = dependencies.TicketSource
	}
}

func (service *Service) CreateKnowledgeCandidateFromTicket(actor identity.IdentityContext, input CreateKnowledgeCandidateFromTicketInput) (KnowledgeCandidate, error) {
	if !canManageKnowledge(actor) {
		return KnowledgeCandidate{}, ErrKnowledgeForbidden
	}
	if !service.candidateWorkflowEnabled() {
		return KnowledgeCandidate{}, ErrKnowledgeCandidateWorkflowDisabled
	}

	knowledgeBase, err := service.getAccessibleKnowledgeBase(actor, input.KnowledgeBaseID)
	if err != nil {
		return KnowledgeCandidate{}, err
	}

	ticketID := strings.TrimSpace(input.TicketID)
	title := strings.TrimSpace(input.Title)
	summary := strings.TrimSpace(input.Summary)
	content := strings.TrimSpace(input.Content)
	if ticketID == "" || title == "" || summary == "" || content == "" {
		return KnowledgeCandidate{}, fmt.Errorf("%w: ticket id, title, summary and content are required", ErrInvalidKnowledgeCandidateInput)
	}

	sourceTicket, err := service.ticketSource.GetTicket(actor, ticketID)
	if err != nil {
		return KnowledgeCandidate{}, err
	}
	if sourceTicket.OrganizationID != knowledgeBase.OrganizationID {
		return KnowledgeCandidate{}, ErrKnowledgeForbidden
	}
	if sourceTicket.Status != ticket.TicketStatusResolved {
		return KnowledgeCandidate{}, fmt.Errorf("%w: only resolved tickets can generate knowledge candidates", ErrInvalidKnowledgeCandidateInput)
	}

	candidate := service.candidateRepository.Save(NewKnowledgeCandidate(KnowledgeCandidate{
		KnowledgeBaseID: knowledgeBase.ID,
		OrganizationID:  knowledgeBase.OrganizationID,
		SourceTicketID:  sourceTicket.ID,
		Title:           title,
		Summary:         summary,
		Content:         content,
		CreatedBy:       actor.UserID,
	}))

	return candidate, nil
}

func (service *Service) ListKnowledgeCandidates(actor identity.IdentityContext, input ListKnowledgeCandidatesInput) ([]KnowledgeCandidate, error) {
	if !service.candidateWorkflowEnabled() {
		return nil, ErrKnowledgeCandidateWorkflowDisabled
	}

	knowledgeBase, err := service.getAccessibleKnowledgeBase(actor, input.KnowledgeBaseID)
	if err != nil {
		return nil, err
	}

	return service.candidateRepository.ListByKnowledgeBase(knowledgeBase.ID), nil
}

func (service *Service) ReviewKnowledgeCandidate(actor identity.IdentityContext, input ReviewKnowledgeCandidateInput) (KnowledgeCandidate, error) {
	if !canManageKnowledge(actor) {
		return KnowledgeCandidate{}, ErrKnowledgeForbidden
	}
	if !service.candidateWorkflowEnabled() {
		return KnowledgeCandidate{}, ErrKnowledgeCandidateWorkflowDisabled
	}

	candidate, err := service.getAccessibleKnowledgeCandidate(actor, input.CandidateID)
	if err != nil {
		return KnowledgeCandidate{}, err
	}
	if candidate.Status != KnowledgeCandidateStatusPendingReview {
		return KnowledgeCandidate{}, fmt.Errorf("%w: candidate already reviewed", ErrInvalidKnowledgeCandidateInput)
	}

	action := ReviewKnowledgeCandidateAction(strings.TrimSpace(string(input.Action)))
	if action != ReviewKnowledgeCandidateActionApprove && action != ReviewKnowledgeCandidateActionReject {
		return KnowledgeCandidate{}, fmt.Errorf("%w: unsupported review action", ErrInvalidKnowledgeCandidateInput)
	}

	now := time.Now()
	candidate.ReviewedBy = actor.UserID
	candidate.ReviewComment = strings.TrimSpace(input.Comment)
	candidate.ReviewedAt = now
	candidate.UpdatedAt = now

	if action == ReviewKnowledgeCandidateActionApprove {
		// 审核通过后复用既有文档上传与异步处理链路，确保知识候选和普通知识文档共享同一生命周期与索引行为。
		document, err := service.UploadDocument(actor, UploadDocumentInput{
			KnowledgeBaseID: candidate.KnowledgeBaseID,
			Filename:        buildKnowledgeCandidateFilename(candidate),
			ContentType:     "text/markdown; charset=utf-8",
			Content:         []byte(buildKnowledgeCandidateMarkdown(candidate)),
		})
		if err != nil {
			return KnowledgeCandidate{}, err
		}
		candidate.Status = KnowledgeCandidateStatusApproved
		candidate.ApprovedDocumentID = document.ID
	} else {
		candidate.Status = KnowledgeCandidateStatusRejected
		candidate.ApprovedDocumentID = ""
	}

	candidate = service.candidateRepository.Save(candidate)
	return candidate, nil
}

func (service *Service) getAccessibleKnowledgeCandidate(actor identity.IdentityContext, candidateID string) (KnowledgeCandidate, error) {
	if !canReadKnowledge(actor) {
		return KnowledgeCandidate{}, ErrKnowledgeForbidden
	}
	if !service.candidateWorkflowEnabled() {
		return KnowledgeCandidate{}, ErrKnowledgeCandidateWorkflowDisabled
	}

	candidate, ok := service.candidateRepository.FindByID(strings.TrimSpace(candidateID))
	if !ok {
		return KnowledgeCandidate{}, ErrKnowledgeCandidateNotFound
	}
	if candidate.OrganizationID != actor.OrganizationID {
		return KnowledgeCandidate{}, ErrKnowledgeForbidden
	}
	return candidate, nil
}

func (service *Service) candidateWorkflowEnabled() bool {
	return service.candidateRepository != nil && service.ticketSource != nil
}

func buildKnowledgeCandidateMarkdown(candidate KnowledgeCandidate) string {
	builder := strings.Builder{}
	builder.WriteString("# ")
	builder.WriteString(candidate.Title)
	builder.WriteString("\n\n")
	builder.WriteString("## 摘要\n")
	builder.WriteString(candidate.Summary)
	builder.WriteString("\n\n")
	builder.WriteString("## 适用工单\n")
	builder.WriteString("- 来源工单：")
	builder.WriteString(candidate.SourceTicketID)
	builder.WriteString("\n")
	builder.WriteString("- 候选编号：")
	builder.WriteString(candidate.ID)
	builder.WriteString("\n\n")
	builder.WriteString("## 处理结论\n")
	builder.WriteString(candidate.Content)
	builder.WriteString("\n")
	return builder.String()
}

func buildKnowledgeCandidateFilename(candidate KnowledgeCandidate) string {
	safeTitle := sanitizeKnowledgeCandidateFilename(candidate.Title)
	if safeTitle == "" {
		safeTitle = candidate.ID
	}
	return fmt.Sprintf("%s-%s.md", candidate.ID, safeTitle)
}

func sanitizeKnowledgeCandidateFilename(title string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "",
		"<", "",
		">", "",
		"|", "-",
	)
	title = replacer.Replace(strings.TrimSpace(title))
	title = strings.Join(strings.Fields(title), "-")
	return strings.Trim(title, "-.")
}
