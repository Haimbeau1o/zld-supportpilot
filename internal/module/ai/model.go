package ai

import (
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/knowledge"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ticket"
)

// AnswerStatus 表示本次问答是正常回答还是降级返回。
type AnswerStatus string

const (
	AnswerStatusAnswered AnswerStatus = "answered"
	AnswerStatusDegraded AnswerStatus = "degraded"
)

// UnifiedIntakeResultType 表示统一受理的最终结果类型。
type UnifiedIntakeResultType string

const (
	UnifiedIntakeResultTypeAnswered      UnifiedIntakeResultType = "answered"
	UnifiedIntakeResultTypeTicketCreated UnifiedIntakeResultType = "ticket_created"
)

// AnswerFeedbackStatus 表示用户对一次 AI 结果的反馈分类。
type AnswerFeedbackStatus string

const (
	AnswerFeedbackStatusResolved   AnswerFeedbackStatus = "resolved"
	AnswerFeedbackStatusUnresolved AnswerFeedbackStatus = "unresolved"
	AnswerFeedbackStatusInaccurate AnswerFeedbackStatus = "inaccurate"
)

// AnswerFeedbackTicketAction 表示反馈提交后系统对工单的处理动作。
type AnswerFeedbackTicketAction string

const (
	AnswerFeedbackTicketActionNone           AnswerFeedbackTicketAction = "none"
	AnswerFeedbackTicketActionTicketCreated  AnswerFeedbackTicketAction = "ticket_created"
	AnswerFeedbackTicketActionTicketExisting AnswerFeedbackTicketAction = "ticket_existing"
)

// Citation 表示回答引用的知识片段。
type Citation struct {
	ChunkID    string
	DocumentID string
	Score      float64
	Content    string
}

// AnswerResult 是回答接口的统一返回结构。
type AnswerResult struct {
	Status     AnswerStatus
	Answer     string
	Confidence float64
	Citations  []Citation
}

// UnifiedIntakeInput 描述统一 AI 受理入口所需输入。
type UnifiedIntakeInput struct {
	KnowledgeBaseID string
	Question        string
	TopK            int
}

// UnifiedIntakeRecord 是统一受理的主记录，用于承接反馈与后续运营分析。
type UnifiedIntakeRecord struct {
	ID                string
	OrganizationID    string
	RequesterID       string
	KnowledgeBaseID   string
	Question          string
	ResultType        UnifiedIntakeResultType
	AnswerStatus      AnswerStatus
	Answer            string
	Confidence        float64
	TicketID          string
	EscalatedTicketID string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// UnifiedIntakeResult 描述统一受理返回：要么直接回答，要么已升级工单。
type UnifiedIntakeResult struct {
	IntakeID     string
	ResultType   UnifiedIntakeResultType
	AnswerStatus AnswerStatus
	Answer       string
	Confidence   float64
	Citations    []Citation
	TicketID     string
}

// AnswerFeedbackInput 描述一次答案反馈提交。
type AnswerFeedbackInput struct {
	IntakeID string
	Status   AnswerFeedbackStatus
	Comment  string
}

// AnswerFeedback 是用户对某次 AI 结果的持久化反馈记录。
type AnswerFeedback struct {
	ID             string
	IntakeID       string
	OrganizationID string
	ActorID        string
	Status         AnswerFeedbackStatus
	Comment        string
	TicketAction   AnswerFeedbackTicketAction
	TicketID       string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// AnswerFeedbackResult 描述一次反馈提交后的业务结果。
type AnswerFeedbackResult struct {
	FeedbackID   string
	IntakeID     string
	Status       AnswerFeedbackStatus
	TicketAction AnswerFeedbackTicketAction
	TicketID     string
}

// AnswerFeedbackStats 描述后台看到的最小反馈统计视图。
type AnswerFeedbackStats struct {
	Total      int
	Resolved   int
	Unresolved int
	Inaccurate int
}

// RetrievedChunk 表示检索器返回的候选 chunk 与分数。
type RetrievedChunk struct {
	Chunk knowledge.DocumentChunk
	Score float64
}

// AskKnowledgeQuestionInput 描述一次知识库问答请求。
type AskKnowledgeQuestionInput struct {
	KnowledgeBaseID string
	Question        string
	TopK            int
}

// GenerateTicketAssistInput 描述一次 AI 工单辅助请求。
type GenerateTicketAssistInput struct {
	TicketID        string
	KnowledgeBaseID string
}

// TicketAssistResult 描述 AI 工单辅助的统一输出。
type TicketAssistResult struct {
	CategorySuggestion string
	Summary            string
	ReplyDraft         string
	AnswerStatus       AnswerStatus
	Confidence         float64
	Citations          []Citation
	RecordedCommentID  string
}

// TicketAnalysisInput 描述 ticket analyzer 所需的上下文。
type TicketAnalysisInput struct {
	Ticket          ticket.Ticket
	Timeline        []ticket.TicketTimelineItem
	KnowledgeAnswer AnswerResult
}

// TicketAnalysisResult 描述 ticket analyzer 生成的建议。
type TicketAnalysisResult struct {
	CategorySuggestion string
	Summary            string
	ReplyDraft         string
}
