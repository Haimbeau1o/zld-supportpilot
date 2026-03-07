package knowledge

import (
	"bytes"
	"errors"
	"testing"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
)

func TestCreateKnowledgeBase(t *testing.T) {
	service := NewService(NewMemoryKnowledgeBaseRepository(), NewMemoryDocumentRepository(), NewMemoryObjectStorage())
	actor := identity.IdentityContext{
		UserID:         "user-agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}

	knowledgeBase, err := service.CreateKnowledgeBase(actor, CreateKnowledgeBaseInput{
		Name:        "IT 支持知识库",
		Description: "用于沉淀常见 IT 问题处理文档",
	})
	if err != nil {
		t.Fatalf("expected create knowledge base success, got error: %v", err)
	}

	if knowledgeBase.ID == "" {
		t.Fatalf("expected knowledge base id to be generated")
	}
	if knowledgeBase.OrganizationID != actor.OrganizationID {
		t.Fatalf("expected organization id %q, got %q", actor.OrganizationID, knowledgeBase.OrganizationID)
	}
}

func TestCreateKnowledgeBaseForbiddenWithoutWritePermission(t *testing.T) {
	service := NewService(NewMemoryKnowledgeBaseRepository(), NewMemoryDocumentRepository(), NewMemoryObjectStorage())
	actor := identity.IdentityContext{
		UserID:         "user-end-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}

	_, err := service.CreateKnowledgeBase(actor, CreateKnowledgeBaseInput{Name: "终端用户知识库"})
	if !errors.Is(err, ErrKnowledgeForbidden) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestUploadDocument(t *testing.T) {
	service := NewService(NewMemoryKnowledgeBaseRepository(), NewMemoryDocumentRepository(), NewMemoryObjectStorage())
	actor := identity.IdentityContext{
		UserID:         "user-agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}

	knowledgeBase, err := service.CreateKnowledgeBase(actor, CreateKnowledgeBaseInput{Name: "IT 支持知识库"})
	if err != nil {
		t.Fatalf("create knowledge base: %v", err)
	}

	document, err := service.UploadDocument(actor, UploadDocumentInput{
		KnowledgeBaseID: knowledgeBase.ID,
		Filename:        "vpn-guide.pdf",
		ContentType:     "application/pdf",
		Content:         bytes.NewBufferString("hello knowledge").Bytes(),
	})
	if err != nil {
		t.Fatalf("expected upload document success, got error: %v", err)
	}

	if document.ID == "" {
		t.Fatalf("expected document id to be generated")
	}
	if document.StorageKey == "" {
		t.Fatalf("expected storage key to be generated")
	}
	if document.Status != DocumentStatusUploaded {
		t.Fatalf("expected document status %q, got %q", DocumentStatusUploaded, document.Status)
	}
}

func TestUploadDocumentRejectsEmptyFile(t *testing.T) {
	service := NewService(NewMemoryKnowledgeBaseRepository(), NewMemoryDocumentRepository(), NewMemoryObjectStorage())
	actor := identity.IdentityContext{
		UserID:         "user-agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}

	knowledgeBase, err := service.CreateKnowledgeBase(actor, CreateKnowledgeBaseInput{Name: "IT 支持知识库"})
	if err != nil {
		t.Fatalf("create knowledge base: %v", err)
	}

	_, err = service.UploadDocument(actor, UploadDocumentInput{
		KnowledgeBaseID: knowledgeBase.ID,
		Filename:        "empty.txt",
		ContentType:     "text/plain",
		Content:         []byte{},
	})
	if !errors.Is(err, ErrInvalidDocumentInput) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestListKnowledgeBasesAndDocuments(t *testing.T) {
	service := NewService(NewMemoryKnowledgeBaseRepository(), NewMemoryDocumentRepository(), NewMemoryObjectStorage())
	agent := identity.IdentityContext{
		UserID:         "user-agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}
	otherOrgAgent := identity.IdentityContext{
		UserID:         "user-agent-2",
		OrganizationID: "org-2",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}

	knowledgeBase1, err := service.CreateKnowledgeBase(agent, CreateKnowledgeBaseInput{Name: "IT 支持知识库"})
	if err != nil {
		t.Fatalf("create first knowledge base: %v", err)
	}
	if _, err := service.CreateKnowledgeBase(agent, CreateKnowledgeBaseInput{Name: "HR 知识库"}); err != nil {
		t.Fatalf("create second knowledge base: %v", err)
	}
	otherOrgKB, err := service.CreateKnowledgeBase(otherOrgAgent, CreateKnowledgeBaseInput{Name: "其他租户知识库"})
	if err != nil {
		t.Fatalf("create third knowledge base: %v", err)
	}

	if _, err := service.UploadDocument(agent, UploadDocumentInput{
		KnowledgeBaseID: knowledgeBase1.ID,
		Filename:        "vpn-guide.pdf",
		ContentType:     "application/pdf",
		Content:         []byte("hello knowledge"),
	}); err != nil {
		t.Fatalf("upload first document: %v", err)
	}
	if _, err := service.UploadDocument(otherOrgAgent, UploadDocumentInput{
		KnowledgeBaseID: otherOrgKB.ID,
		Filename:        "other.pdf",
		ContentType:     "application/pdf",
		Content:         []byte("other"),
	}); err != nil {
		t.Fatalf("upload second document: %v", err)
	}

	knowledgeBases, err := service.ListKnowledgeBases(agent)
	if err != nil {
		t.Fatalf("expected list knowledge bases success, got error: %v", err)
	}
	if len(knowledgeBases) != 2 {
		t.Fatalf("expected agent to see 2 knowledge bases in tenant, got %d", len(knowledgeBases))
	}

	documents, err := service.ListDocuments(agent, ListDocumentsInput{KnowledgeBaseID: knowledgeBase1.ID})
	if err != nil {
		t.Fatalf("expected list documents success, got error: %v", err)
	}
	if len(documents) != 1 {
		t.Fatalf("expected agent to see 1 document in tenant knowledge base, got %d", len(documents))
	}

	document, err := service.GetDocument(agent, documents[0].ID)
	if err != nil {
		t.Fatalf("expected get document success, got error: %v", err)
	}
	if document.KnowledgeBaseID != knowledgeBase1.ID {
		t.Fatalf("expected knowledge base id %q, got %q", knowledgeBase1.ID, document.KnowledgeBaseID)
	}
}
