package knowledge

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
)

var (
	ErrKnowledgeForbidden             = errors.New("knowledge forbidden")
	ErrKnowledgeBaseNotFound          = errors.New("knowledge base not found")
	ErrDocumentNotFound               = errors.New("document not found")
	ErrInvalidKnowledgeInput          = errors.New("invalid knowledge input")
	ErrInvalidDocumentInput           = errors.New("invalid document input")
	ErrKnowledgeStorageFailed         = errors.New("knowledge storage failed")
	ErrKnowledgeProcessingDisabled    = errors.New("knowledge processing disabled")
	ErrDocumentProcessingTaskNotFound = errors.New("document processing task not found")
	ErrDocumentRetryNotAllowed        = errors.New("document retry not allowed")
	ErrKnowledgeProcessingFailed      = errors.New("knowledge processing failed")
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

type ProcessingDependencies struct {
	TaskRepository  DocumentProcessingTaskRepository
	ChunkRepository DocumentChunkRepository
	Dispatcher      ProcessingTaskDispatcher
	Parser          DocumentParser
	Chunker         DocumentChunker
	Indexer         ChunkIndexer
	MaxAttempts     int
}

type ServiceOption func(*Service)

type Service struct {
	knowledgeBaseRepository KnowledgeBaseRepository
	documentRepository      DocumentRepository
	objectStorage           ObjectStorage
	candidateRepository     KnowledgeCandidateRepository
	ticketSource            TicketSource
	taskRepository          DocumentProcessingTaskRepository
	chunkRepository         DocumentChunkRepository
	dispatcher              ProcessingTaskDispatcher
	parser                  DocumentParser
	chunker                 DocumentChunker
	indexer                 ChunkIndexer
	maxProcessingAttempts   int
}

func WithProcessingPipeline(dependencies ProcessingDependencies) ServiceOption {
	return func(service *Service) {
		service.taskRepository = dependencies.TaskRepository
		service.chunkRepository = dependencies.ChunkRepository
		service.dispatcher = dependencies.Dispatcher
		service.parser = dependencies.Parser
		service.chunker = dependencies.Chunker
		service.indexer = dependencies.Indexer
		if dependencies.MaxAttempts > 0 {
			service.maxProcessingAttempts = dependencies.MaxAttempts
		}
	}
}

func NewService(knowledgeBaseRepository KnowledgeBaseRepository, documentRepository DocumentRepository, objectStorage ObjectStorage, options ...ServiceOption) *Service {
	service := &Service{
		knowledgeBaseRepository: knowledgeBaseRepository,
		documentRepository:      documentRepository,
		objectStorage:           objectStorage,
		maxProcessingAttempts:   3,
	}
	for _, option := range options {
		option(service)
	}
	if service.maxProcessingAttempts < 1 {
		service.maxProcessingAttempts = 1
	}
	return service
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

	if service.processingEnabled() {
		// 上传成功后只负责“创建任务并入队”，后续耗时解析放到后台 worker，避免 API 被长任务阻塞。
		document, _ = service.scheduleDocumentProcessing(document, 1)
	}

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

func (service *Service) ListIndexedChunks(actor identity.IdentityContext, knowledgeBaseID string) ([]DocumentChunk, error) {
	if service.chunkRepository == nil {
		return nil, ErrKnowledgeProcessingDisabled
	}
	if !canReadPublishedKnowledge(actor) {
		return nil, ErrKnowledgeForbidden
	}

	knowledgeBase, ok := service.knowledgeBaseRepository.FindByID(strings.TrimSpace(knowledgeBaseID))
	if !ok {
		return nil, ErrKnowledgeBaseNotFound
	}
	if actor.OrganizationID != knowledgeBase.OrganizationID {
		return nil, ErrKnowledgeForbidden
	}

	// 已发布 chunk 是面向问答与自助服务的“发布态知识”，这里允许终端用户通过 AI 能力间接消费，
	// 但仍不放开原始文档和知识库管理接口，避免把后台维护面直接暴露出去。
	return service.chunkRepository.ListByKnowledgeBase(knowledgeBase.ID), nil
}

func (service *Service) ListDocumentProcessingTasks(actor identity.IdentityContext, documentID string) ([]DocumentProcessingTask, error) {
	if !service.processingEnabled() {
		return nil, ErrKnowledgeProcessingDisabled
	}

	document, err := service.getAccessibleDocument(actor, documentID)
	if err != nil {
		return nil, err
	}

	return service.taskRepository.ListByDocument(document.ID), nil
}

func (service *Service) RetryDocumentProcessing(actor identity.IdentityContext, documentID string) (DocumentProcessingTask, error) {
	if !service.processingEnabled() {
		return DocumentProcessingTask{}, ErrKnowledgeProcessingDisabled
	}
	if !canManageKnowledge(actor) {
		return DocumentProcessingTask{}, ErrKnowledgeForbidden
	}

	document, err := service.getAccessibleDocument(actor, documentID)
	if err != nil {
		return DocumentProcessingTask{}, err
	}
	if document.Status != DocumentStatusFailed {
		return DocumentProcessingTask{}, fmt.Errorf("%w: document must be in failed status", ErrDocumentRetryNotAllowed)
	}

	attempt := 1
	if latestTask, ok := service.taskRepository.FindLatestByDocument(document.ID); ok {
		attempt = latestTask.Attempt + 1
		if latestTask.Attempt >= service.maxProcessingAttempts {
			return DocumentProcessingTask{}, fmt.Errorf("%w: max attempts reached", ErrDocumentRetryNotAllowed)
		}
	}

	document.Status = DocumentStatusUploaded
	document.LastError = ""
	document.FailedAt = time.Time{}
	document.UpdatedAt = time.Now()
	document = service.documentRepository.Save(document)

	_, task := service.scheduleDocumentProcessing(document, attempt)
	return task, nil
}

func (service *Service) ProcessTask(ctx context.Context, taskID string) (DocumentProcessingTask, error) {
	if !service.processingEnabled() {
		return DocumentProcessingTask{}, ErrKnowledgeProcessingDisabled
	}

	task, ok := service.taskRepository.FindByID(strings.TrimSpace(taskID))
	if !ok {
		return DocumentProcessingTask{}, ErrDocumentProcessingTaskNotFound
	}

	document, ok := service.documentRepository.FindByID(task.DocumentID)
	if !ok {
		return service.failProcessingTask(task, Document{}, ErrDocumentNotFound)
	}

	now := time.Now()
	task.Status = ProcessingTaskStatusRunning
	task.Stage = ProcessingStageParse
	task.StartedAt = now
	task.UpdatedAt = now
	task = service.taskRepository.Save(task)

	document.Status = DocumentStatusProcessing
	document.ProcessingStartedAt = now
	document.LastError = ""
	document.UpdatedAt = now
	document = service.documentRepository.Save(document)

	storedObject, err := service.objectStorage.Load(document.StorageKey)
	if err != nil {
		return service.failProcessingTask(task, document, err)
	}

	parsedDocument, err := service.parser.Parse(ctx, document, storedObject)
	if err != nil {
		return service.failProcessingTask(task, document, err)
	}

	task.Stage = ProcessingStageChunk
	task.UpdatedAt = time.Now()
	task = service.taskRepository.Save(task)

	chunks, err := service.chunker.Chunk(ctx, document, parsedDocument)
	if err != nil {
		return service.failProcessingTask(task, document, err)
	}

	task.Stage = ProcessingStageIndex
	task.UpdatedAt = time.Now()
	task = service.taskRepository.Save(task)

	if err := service.indexer.ReplaceDocumentChunks(ctx, document, chunks); err != nil {
		return service.failProcessingTask(task, document, err)
	}

	finishedAt := time.Now()
	task.Status = ProcessingTaskStatusSucceeded
	task.Stage = ProcessingStageIndex
	task.ErrorMessage = ""
	task.FinishedAt = finishedAt
	task.UpdatedAt = finishedAt
	task = service.taskRepository.Save(task)

	document.Status = DocumentStatusReady
	document.ChunkCount = len(chunks)
	document.LastError = ""
	document.ProcessedAt = finishedAt
	document.FailedAt = time.Time{}
	document.UpdatedAt = finishedAt
	document = service.documentRepository.Save(document)

	return task, nil
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

func (service *Service) getAccessibleDocument(actor identity.IdentityContext, documentID string) (Document, error) {
	if !canReadKnowledge(actor) {
		return Document{}, ErrKnowledgeForbidden
	}

	document, ok := service.documentRepository.FindByID(strings.TrimSpace(documentID))
	if !ok {
		return Document{}, ErrDocumentNotFound
	}
	if actor.OrganizationID != document.OrganizationID {
		return Document{}, ErrKnowledgeForbidden
	}

	return document, nil
}

func (service *Service) scheduleDocumentProcessing(document Document, attempt int) (Document, DocumentProcessingTask) {
	task := NewDocumentProcessingTask(document, attempt, service.maxProcessingAttempts)
	task = service.taskRepository.Save(task)

	document.LastTaskID = task.ID
	document.ProcessingAttempts = task.Attempt
	document.ProcessingQueuedAt = task.CreatedAt
	document.UpdatedAt = task.CreatedAt
	document = service.documentRepository.Save(document)

	if err := service.dispatcher.Enqueue(task.ID); err != nil {
		// 入队失败不回滚文档和原始文件，这样“上传成功”和“处理失败”可以被明确区分并支持后续重试。
		failedTask, _ := service.failProcessingTask(task, document, err)
		return service.mustGetDocument(document.ID), failedTask
	}

	return document, task
}

func (service *Service) failProcessingTask(task DocumentProcessingTask, document Document, err error) (DocumentProcessingTask, error) {
	finishedAt := time.Now()
	task.Status = ProcessingTaskStatusFailed
	task.ErrorMessage = err.Error()
	task.FinishedAt = finishedAt
	task.UpdatedAt = finishedAt
	task = service.taskRepository.Save(task)

	if document.ID != "" {
		document.Status = DocumentStatusFailed
		document.LastError = task.ErrorMessage
		document.FailedAt = finishedAt
		document.UpdatedAt = finishedAt
		service.documentRepository.Save(document)
	}

	return task, fmt.Errorf("%w: %v", ErrKnowledgeProcessingFailed, err)
}

func (service *Service) mustGetDocument(documentID string) Document {
	document, _ := service.documentRepository.FindByID(documentID)
	return document
}

func (service *Service) processingEnabled() bool {
	return service.taskRepository != nil && service.dispatcher != nil && service.parser != nil && service.chunker != nil && service.indexer != nil
}

func canManageKnowledge(actor identity.IdentityContext) bool {
	return actor.OrganizationID != "" && actor.Permissions.Contains(identity.PermissionKnowledgeWrite)
}

func canReadKnowledge(actor identity.IdentityContext) bool {
	return actor.OrganizationID != "" && (actor.Permissions.Contains(identity.PermissionKnowledgeRead) || actor.Permissions.Contains(identity.PermissionKnowledgeWrite))
}

func canReadPublishedKnowledge(actor identity.IdentityContext) bool {
	if actor.OrganizationID == "" {
		return false
	}
	if canReadKnowledge(actor) {
		return true
	}
	return actor.Permissions.Contains(identity.PermissionTicketSelfCreate) || actor.Permissions.Contains(identity.PermissionTicketSelfRead)
}

func buildStorageKey(document Document) string {
	safeFilename := strings.TrimSpace(document.Filename)
	if safeFilename == "" {
		safeFilename = "document.bin"
	}

	baseName := filepath.Base(safeFilename)
	return fmt.Sprintf("knowledge/%s/%s/%s", document.OrganizationID, document.KnowledgeBaseID, document.ID+"-"+baseName)
}
