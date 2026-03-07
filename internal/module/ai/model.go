package ai

import "github.com/Haimbeau1o/zld-supportpilot/internal/module/knowledge"

// AnswerStatus 表示本次问答是正常回答还是降级返回。
type AnswerStatus string

const (
	AnswerStatusAnswered AnswerStatus = "answered"
	AnswerStatusDegraded AnswerStatus = "degraded"
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
