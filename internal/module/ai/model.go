package ai

import (
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

// UnifiedIntakeResult 描述统一受理返回：要么直接回答，要么已升级工单。
type UnifiedIntakeResult struct {
	ResultType   UnifiedIntakeResultType
	AnswerStatus AnswerStatus
	Answer       string
	Confidence   float64
	Citations    []Citation
	TicketID     string
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
