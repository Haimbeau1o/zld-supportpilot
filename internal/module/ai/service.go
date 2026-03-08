package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/knowledge"
)

var ErrInvalidAIInput = errors.New("invalid ai input")

// ChunkSource 定义 AI 服务读取 ready chunk 的边界，避免 AI 直接介入知识处理流水线内部细节。
type ChunkSource interface {
	ListIndexedChunks(actor identity.IdentityContext, knowledgeBaseID string) ([]knowledge.DocumentChunk, error)
}

// AnswerGenerator 负责把检索结果组织成最终回答文本。
type AnswerGenerator interface {
	GenerateAnswer(ctx context.Context, question string, retrievedChunks []RetrievedChunk) (string, error)
}

type ServiceDependencies struct {
	ChunkSource     ChunkSource
	Retriever       Retriever
	AnswerGenerator AnswerGenerator
	MinConfidence   float64
}

type Service struct {
	chunkSource     ChunkSource
	retriever       Retriever
	answerGenerator AnswerGenerator
	minConfidence   float64
}

func NewService(dependencies ServiceDependencies) *Service {
	minConfidence := dependencies.MinConfidence
	if minConfidence <= 0 {
		minConfidence = 0.15
	}
	return &Service{
		chunkSource:     dependencies.ChunkSource,
		retriever:       dependencies.Retriever,
		answerGenerator: dependencies.AnswerGenerator,
		minConfidence:   minConfidence,
	}
}

func (service *Service) AskKnowledgeQuestion(ctx context.Context, actor identity.IdentityContext, input AskKnowledgeQuestionInput) (AnswerResult, error) {
	if service.chunkSource == nil || service.retriever == nil || service.answerGenerator == nil {
		return AnswerResult{}, fmt.Errorf("%w: ai service dependencies are incomplete", ErrInvalidAIInput)
	}

	knowledgeBaseID := strings.TrimSpace(input.KnowledgeBaseID)
	question := strings.TrimSpace(input.Question)
	if knowledgeBaseID == "" {
		return AnswerResult{}, fmt.Errorf("%w: knowledge base id is required", ErrInvalidAIInput)
	}
	if question == "" {
		return AnswerResult{}, fmt.Errorf("%w: question is required", ErrInvalidAIInput)
	}

	topK := input.TopK
	if topK == 0 {
		topK = 3
	}
	if topK < 0 {
		return AnswerResult{}, fmt.Errorf("%w: top_k must be positive", ErrInvalidAIInput)
	}

	chunks, err := service.chunkSource.ListIndexedChunks(actor, knowledgeBaseID)
	if err != nil {
		return AnswerResult{}, err
	}

	retrievedChunks, err := service.retriever.Retrieve(ctx, question, chunks, topK)
	if err != nil {
		return AnswerResult{}, err
	}
	if len(retrievedChunks) == 0 {
		return degradedAnswerResult(), nil
	}

	confidence := retrievedChunks[0].Score
	if confidence < service.minConfidence {
		return degradedAnswerResult(), nil
	}

	answer, err := service.answerGenerator.GenerateAnswer(ctx, question, retrievedChunks)
	if err != nil {
		return AnswerResult{}, err
	}

	citations := make([]Citation, 0, len(retrievedChunks))
	for _, item := range retrievedChunks {
		citations = append(citations, Citation{
			ChunkID:    item.Chunk.ID,
			DocumentID: item.Chunk.DocumentID,
			Score:      item.Score,
			Content:    item.Chunk.Content,
		})
	}

	return AnswerResult{
		Status:     AnswerStatusAnswered,
		Answer:     answer,
		Confidence: confidence,
		Citations:  citations,
	}, nil
}

// TemplateAnswerGenerator 先用可解释的模板式回答把接口契约稳定下来，后续可平滑替换成真实 LLM 生成器。
type TemplateAnswerGenerator struct{}

func (TemplateAnswerGenerator) GenerateAnswer(_ context.Context, question string, retrievedChunks []RetrievedChunk) (string, error) {
	if len(retrievedChunks) == 0 {
		return degradedAnswerResult().Answer, nil
	}

	builder := strings.Builder{}
	builder.WriteString("根据知识库资料，建议优先执行以下操作：")
	for index, chunk := range retrievedChunks {
		builder.WriteString(fmt.Sprintf(" [%d] %s", index+1, strings.TrimSpace(chunk.Chunk.Content)))
		if index >= 1 {
			break
		}
	}
	builder.WriteString(" 如仍无法解决，建议转人工进一步排查。")
	return builder.String(), nil
}

func degradedAnswerResult() AnswerResult {
	// 当检索置信度不足时，宁可显式降级，也不返回“看似合理”的猜测性答案。
	return AnswerResult{
		Status:     AnswerStatusDegraded,
		Answer:     "当前知识库暂无足够依据可靠回答该问题，建议转人工处理或补充相关文档后重试。",
		Confidence: 0,
		Citations:  []Citation{},
	}
}
