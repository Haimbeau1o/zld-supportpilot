package ai

import (
	"context"
	"testing"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/knowledge"
)

func TestVectorRetrieverRanksRelevantChunksFirst(t *testing.T) {
	retriever := NewVectorRetriever(NewHashingEmbedder(64))
	chunks := []knowledge.DocumentChunk{
		{ID: "chunk-1", DocumentID: "doc-1", KnowledgeBaseID: "kb-1", Content: "VPN 无法连接时请先检查客户端是否在线"},
		{ID: "chunk-2", DocumentID: "doc-2", KnowledgeBaseID: "kb-1", Content: "薪资审批需要在人力系统中提交工单"},
		{ID: "chunk-3", DocumentID: "doc-3", KnowledgeBaseID: "kb-1", Content: "VPN 客户端重置后通常可以恢复连接"},
	}

	results, err := retriever.Retrieve(context.Background(), "VPN 无法连接怎么办", chunks, 2)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Chunk.ID != "chunk-1" && results[0].Chunk.ID != "chunk-3" {
		t.Fatalf("expected vpn chunk first, got %q", results[0].Chunk.ID)
	}
	if results[0].Score < results[1].Score {
		t.Fatalf("expected descending score order, got %f < %f", results[0].Score, results[1].Score)
	}
}

func TestVectorRetrieverAppliesTopK(t *testing.T) {
	retriever := NewVectorRetriever(NewHashingEmbedder(32))
	chunks := []knowledge.DocumentChunk{
		{ID: "chunk-1", DocumentID: "doc-1", KnowledgeBaseID: "kb-1", Content: "VPN 登录说明"},
		{ID: "chunk-2", DocumentID: "doc-2", KnowledgeBaseID: "kb-1", Content: "VPN 重置说明"},
		{ID: "chunk-3", DocumentID: "doc-3", KnowledgeBaseID: "kb-1", Content: "VPN 排障说明"},
	}

	results, err := retriever.Retrieve(context.Background(), "VPN", chunks, 1)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestVectorRetrieverHandlesEmptyChunks(t *testing.T) {
	retriever := NewVectorRetriever(NewHashingEmbedder(32))

	results, err := retriever.Retrieve(context.Background(), "VPN", nil, 3)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected empty results, got %d", len(results))
	}
}

func TestHashingEmbedderReturnsStableDimensions(t *testing.T) {
	embedder := NewHashingEmbedder(16)

	vector, err := embedder.EmbedText(context.Background(), "VPN 无法连接")
	if err != nil {
		t.Fatalf("embed text: %v", err)
	}
	if len(vector) != 16 {
		t.Fatalf("expected vector length 16, got %d", len(vector))
	}

	vector2, err := embedder.EmbedText(context.Background(), "VPN 无法连接")
	if err != nil {
		t.Fatalf("embed text second time: %v", err)
	}
	for index := range vector {
		if vector[index] != vector2[index] {
			t.Fatalf("expected deterministic embedding at index %d", index)
		}
	}
}
