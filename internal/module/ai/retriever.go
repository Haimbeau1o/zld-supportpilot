package ai

import (
	"context"
	"errors"
	"hash/fnv"
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/knowledge"
)

var ErrInvalidRetrieverInput = errors.New("invalid retriever input")

// VectorEmbedder 负责把文本映射为检索向量；当前阶段先用可解释的哈希向量稳定接口，后续再换真实 embedding 模型。
type VectorEmbedder interface {
	EmbedText(ctx context.Context, text string) ([]float64, error)
}

// Retriever 负责对 chunk 做相似度检索与排序。
type Retriever interface {
	Retrieve(ctx context.Context, query string, chunks []knowledge.DocumentChunk, topK int) ([]RetrievedChunk, error)
}

// HashingEmbedder 用固定维度哈希向量模拟 embedding，避免当前阶段引入外部依赖。
type HashingEmbedder struct {
	dimensions int
}

func NewHashingEmbedder(dimensions int) *HashingEmbedder {
	if dimensions < 8 {
		dimensions = 8
	}
	return &HashingEmbedder{dimensions: dimensions}
}

func (embedder *HashingEmbedder) EmbedText(_ context.Context, text string) ([]float64, error) {
	vector := make([]float64, embedder.dimensions)
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return vector, nil
	}

	for _, token := range tokens {
		index, sign := hashToken(token, embedder.dimensions)
		vector[index] += sign
	}

	normalize(vector)
	return vector, nil
}

// VectorRetriever 用 embedding + 余弦相似度完成 chunk 排序。
type VectorRetriever struct {
	embedder VectorEmbedder
}

func NewVectorRetriever(embedder VectorEmbedder) *VectorRetriever {
	return &VectorRetriever{embedder: embedder}
}

func (retriever *VectorRetriever) Retrieve(ctx context.Context, query string, chunks []knowledge.DocumentChunk, topK int) ([]RetrievedChunk, error) {
	if retriever.embedder == nil {
		return nil, errors.New("embedder is required")
	}
	if topK < 1 {
		return nil, ErrInvalidRetrieverInput
	}
	if len(chunks) == 0 {
		return []RetrievedChunk{}, nil
	}

	queryVector, err := retriever.embedder.EmbedText(ctx, query)
	if err != nil {
		return nil, err
	}

	results := make([]RetrievedChunk, 0, len(chunks))
	for _, chunk := range chunks {
		chunkVector, err := retriever.embedder.EmbedText(ctx, chunk.Content)
		if err != nil {
			return nil, err
		}
		results = append(results, RetrievedChunk{
			Chunk: chunk,
			Score: cosineSimilarity(queryVector, chunkVector),
		})
	}

	sort.Slice(results, func(left, right int) bool {
		if results[left].Score == results[right].Score {
			return results[left].Chunk.ID < results[right].Chunk.ID
		}
		return results[left].Score > results[right].Score
	})

	if topK > len(results) {
		topK = len(results)
	}
	return results[:topK], nil
}

func tokenize(text string) []string {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return nil
	}

	tokens := make([]string, 0)
	asciiBuffer := make([]rune, 0)
	flushASCII := func() {
		if len(asciiBuffer) == 0 {
			return
		}
		tokens = append(tokens, string(asciiBuffer))
		asciiBuffer = asciiBuffer[:0]
	}

	for _, r := range lower {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if unicode.Is(unicode.Han, r) {
				flushASCII()
				tokens = append(tokens, string(r))
				continue
			}
			asciiBuffer = append(asciiBuffer, r)
		default:
			flushASCII()
		}
	}
	flushASCII()
	return tokens
}

func hashToken(token string, dimensions int) (int, float64) {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(token))
	sum := hasher.Sum64()
	index := int(sum % uint64(dimensions))
	sign := 1.0
	if sum&1 == 1 {
		sign = -1.0
	}
	return index, sign
}

func cosineSimilarity(left []float64, right []float64) float64 {
	if len(left) == 0 || len(right) == 0 || len(left) != len(right) {
		return 0
	}

	dotProduct := 0.0
	leftNorm := 0.0
	rightNorm := 0.0
	for index := range left {
		dotProduct += left[index] * right[index]
		leftNorm += left[index] * left[index]
		rightNorm += right[index] * right[index]
	}
	if leftNorm == 0 || rightNorm == 0 {
		return 0
	}
	return dotProduct / (math.Sqrt(leftNorm) * math.Sqrt(rightNorm))
}

func normalize(vector []float64) {
	norm := 0.0
	for _, value := range vector {
		norm += value * value
	}
	if norm == 0 {
		return
	}
	norm = math.Sqrt(norm)
	for index := range vector {
		vector[index] = vector[index] / norm
	}
}
