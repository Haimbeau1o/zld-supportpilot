package knowledge

import (
	"testing"
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
)

func TestListIndexedChunksAllowsEndUserSelfServiceAccess(t *testing.T) {
	chunkRepository := NewMemoryDocumentChunkRepository()
	service := NewService(
		NewMemoryKnowledgeBaseRepository(),
		NewMemoryDocumentRepository(),
		NewMemoryObjectStorage(),
	)
	service.chunkRepository = chunkRepository

	agent := identity.IdentityContext{
		UserID:         "user-agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}
	endUser := identity.IdentityContext{
		UserID:         "user-end-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}

	knowledgeBase, err := service.CreateKnowledgeBase(agent, CreateKnowledgeBaseInput{Name: "IT 支持知识库"})
	if err != nil {
		t.Fatalf("create knowledge base: %v", err)
	}

	chunkRepository.ReplaceByDocument("doc-1", []DocumentChunk{{
		ID:              "chunk-1",
		DocumentID:      "doc-1",
		KnowledgeBaseID: knowledgeBase.ID,
		OrganizationID:  knowledgeBase.OrganizationID,
		Sequence:        1,
		Content:         "VPN 691 错误通常需要 IT 服务台继续处理。",
		CreatedAt:       time.Now(),
	}})

	chunks, err := service.ListIndexedChunks(endUser, knowledgeBase.ID)
	if err != nil {
		t.Fatalf("expected end user to read indexed chunks for self-service ai, got %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 indexed chunk, got %d", len(chunks))
	}
}
