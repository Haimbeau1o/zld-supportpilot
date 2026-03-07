package knowledge

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
)

var (
	ErrKnowledgeForbidden     = errors.New("knowledge forbidden")
	ErrKnowledgeBaseNotFound  = errors.New("knowledge base not found")
	ErrDocumentNotFound       = errors.New("document not found")
	ErrInvalidKnowledgeInput  = errors.New("invalid knowledge input")
	ErrInvalidDocumentInput   = errors.New("invalid document input")
	ErrKnowledgeStorageFailed = errors.New("knowledge storage failed")
)

type CreateKnowledgeBaseInput struct {
	Name        string
	Description string
}

type UploadDocumentInput struct {
	KnowledgeBaseID string
	Filename        string
	ContentType     string
	Content         []byte
}

type ListDocumentsInput struct {
	KnowledgeBaseID string
}

type Service struct {
	knowledgeBaseRepository KnowledgeBaseRepository
	documentRepository      DocumentRepository
	objectStorage           ObjectStorage
}

func NewService(knowledgeBaseRepository KnowledgeBaseRepository, documentRepository DocumentRepository, objectStorage ObjectStorage) *Service {
	return &Service{
		knowledgeBaseRepository: knowledgeBaseRepository,
		documentRepository:      documentRepository,
		objectStorage:           objectStorage,
	}
}

func (service *Service) CreateKnowledgeBase(actor identity.IdentityContext, input CreateKnowledgeBaseInput) (KnowledgeBase, error) {
	if !canManageKnowledge(actor) {
		return KnowledgeBase{}, ErrKnowledgeForbidden
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return KnowledgeBase{}, fmt.Errorf("%w: knowledge base name is required", ErrInvalidKnowledgeInput)
	}

	now := time.Now()
	knowledgeBase := service.knowledgeBaseRepository.Save(KnowledgeBase{
		OrganizationID: actor.OrganizationID,
		Name:           name,
		Description:    strings.TrimSpace(input.Description),
		CreatedBy:      actor.UserID,
		CreatedAt:      now,
		UpdatedAt:      now,
	})

	return knowledgeBase, nil
}

func (service *Service) ListKnowledgeBases(actor identity.IdentityContext) ([]KnowledgeBase, error) {
	if !canReadKnowledge(actor) {
		return nil, ErrKnowledgeForbidden
	}

	return service.knowledgeBaseRepository.ListByOrganization(actor.OrganizationID), nil
}

func (service *Service) UploadDocument(actor identity.IdentityContext, input UploadDocumentInput) (Document, error) {
	if !canManageKnowledge(actor) {
		return Document{}, ErrKnowledgeForbidden
	}

	knowledgeBase, err := service.getAccessibleKnowledgeBase(actor, input.KnowledgeBaseID)
	if err != nil {
		return Document{}, err
	}

	filename := strings.TrimSpace(input.Filename)
	if filename == "" {
		return Document{}, fmt.Errorf("%w: filename is required", ErrInvalidDocumentInput)
	}
	if len(input.Content) == 0 {
		return Document{}, fmt.Errorf("%w: document content is empty", ErrInvalidDocumentInput)
	}

	contentType := strings.TrimSpace(input.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	document := service.documentRepository.Save(NewDocument(Document{
		KnowledgeBaseID: knowledgeBase.ID,
		OrganizationID:  knowledgeBase.OrganizationID,
		Filename:        filename,
		ContentType:     contentType,
		SizeBytes:       int64(len(input.Content)),
		UploadedBy:      actor.UserID,
	}))

	storageKey := buildStorageKey(document)
	if err := service.objectStorage.Save(SaveObjectInput{
		Key:         storageKey,
		ContentType: contentType,
		Content:     input.Content,
	}); err != nil {
		document.Status = DocumentStatusFailed
		document.UpdatedAt = time.Now()
		service.documentRepository.Save(document)
		return Document{}, fmt.Errorf("%w: %v", ErrKnowledgeStorageFailed, err)
	}

	document.StorageKey = storageKey
	document.UpdatedAt = time.Now()
	document = service.documentRepository.Save(document)
	return document, nil
}

func (service *Service) ListDocuments(actor identity.IdentityContext, input ListDocumentsInput) ([]Document, error) {
	knowledgeBase, err := service.getAccessibleKnowledgeBase(actor, input.KnowledgeBaseID)
	if err != nil {
		return nil, err
	}

	return service.documentRepository.ListByKnowledgeBase(knowledgeBase.ID), nil
}

func (service *Service) GetDocument(actor identity.IdentityContext, documentID string) (Document, error) {
	if !canReadKnowledge(actor) {
		return Document{}, ErrKnowledgeForbidden
	}

	document, ok := service.documentRepository.FindByID(documentID)
	if !ok {
		return Document{}, ErrDocumentNotFound
	}
	if actor.OrganizationID != document.OrganizationID {
		return Document{}, ErrKnowledgeForbidden
	}

	return document, nil
}

func (service *Service) getAccessibleKnowledgeBase(actor identity.IdentityContext, knowledgeBaseID string) (KnowledgeBase, error) {
	if !canReadKnowledge(actor) {
		return KnowledgeBase{}, ErrKnowledgeForbidden
	}

	knowledgeBase, ok := service.knowledgeBaseRepository.FindByID(strings.TrimSpace(knowledgeBaseID))
	if !ok {
		return KnowledgeBase{}, ErrKnowledgeBaseNotFound
	}
	if actor.OrganizationID != knowledgeBase.OrganizationID {
		return KnowledgeBase{}, ErrKnowledgeForbidden
	}

	return knowledgeBase, nil
}

func canManageKnowledge(actor identity.IdentityContext) bool {
	return actor.Permissions.Contains(identity.PermissionKnowledgeWrite)
}

func canReadKnowledge(actor identity.IdentityContext) bool {
	return actor.Permissions.Contains(identity.PermissionKnowledgeRead) || actor.Permissions.Contains(identity.PermissionKnowledgeWrite)
}

func buildStorageKey(document Document) string {
	safeFilename := sanitizeFilename(document.Filename)
	// 上传文件统一挂在知识库路径之下，这样后续 #6 的异步处理和 #7 的检索链路能直接按知识库边界追踪原始文件来源。
	return fmt.Sprintf("org/%s/knowledge/%s/documents/%s/%s", document.OrganizationID, document.KnowledgeBaseID, document.ID, safeFilename)
}

func sanitizeFilename(filename string) string {
	base := filepath.Base(strings.TrimSpace(filename))
	replacer := strings.NewReplacer(" ", "_", "/", "-", "\\", "-", ":", "-")
	return replacer.Replace(base)
}
