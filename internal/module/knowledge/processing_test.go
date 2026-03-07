package knowledge

import (
	"context"
	"errors"
	"testing"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
)

func TestUploadDocumentCreatesProcessingTask(t *testing.T) {
	taskRepository := NewMemoryDocumentProcessingTaskRepository()
	chunkRepository := NewMemoryDocumentChunkRepository()
	dispatcher := &stubProcessingTaskDispatcher{}
	service := NewService(
		NewMemoryKnowledgeBaseRepository(),
		NewMemoryDocumentRepository(),
		NewMemoryObjectStorage(),
		WithProcessingPipeline(ProcessingDependencies{
			TaskRepository: taskRepository,
			Dispatcher:     dispatcher,
			Parser:         stubDocumentParser{},
			Chunker:        stubDocumentChunker{},
			Indexer:        stubChunkIndexer{repository: chunkRepository},
			MaxAttempts:    3,
		}),
	)

	agent := identity.IdentityContext{
		UserID:         "user-agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}

	knowledgeBase, err := service.CreateKnowledgeBase(agent, CreateKnowledgeBaseInput{Name: "IT 支持知识库"})
	if err != nil {
		t.Fatalf("create knowledge base: %v", err)
	}

	document, err := service.UploadDocument(agent, UploadDocumentInput{
		KnowledgeBaseID: knowledgeBase.ID,
		Filename:        "vpn-guide.txt",
		ContentType:     "text/plain",
		Content:         []byte("first paragraph\n\nsecond paragraph"),
	})
	if err != nil {
		t.Fatalf("upload document: %v", err)
	}

	if document.LastTaskID == "" {
		t.Fatalf("expected uploaded document to have processing task id")
	}
	if document.ProcessingAttempts != 1 {
		t.Fatalf("expected processing attempts 1, got %d", document.ProcessingAttempts)
	}
	if len(dispatcher.enqueuedTaskIDs) != 1 {
		t.Fatalf("expected exactly one enqueued task, got %d", len(dispatcher.enqueuedTaskIDs))
	}

	tasks := taskRepository.ListByDocument(document.ID)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 processing task, got %d", len(tasks))
	}
	if tasks[0].Status != ProcessingTaskStatusQueued {
		t.Fatalf("expected queued task, got %q", tasks[0].Status)
	}
	if tasks[0].Stage != ProcessingStageParse {
		t.Fatalf("expected parse stage, got %q", tasks[0].Stage)
	}
}

func TestProcessTaskTransitionsDocumentToReady(t *testing.T) {
	taskRepository := NewMemoryDocumentProcessingTaskRepository()
	chunkRepository := NewMemoryDocumentChunkRepository()
	service := NewService(
		NewMemoryKnowledgeBaseRepository(),
		NewMemoryDocumentRepository(),
		NewMemoryObjectStorage(),
		WithProcessingPipeline(ProcessingDependencies{
			TaskRepository: taskRepository,
			Dispatcher:     &stubProcessingTaskDispatcher{},
			Parser:         stubDocumentParser{},
			Chunker:        stubDocumentChunker{},
			Indexer:        stubChunkIndexer{repository: chunkRepository},
			MaxAttempts:    3,
		}),
	)

	agent := identity.IdentityContext{
		UserID:         "user-agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}

	knowledgeBase, err := service.CreateKnowledgeBase(agent, CreateKnowledgeBaseInput{Name: "IT 支持知识库"})
	if err != nil {
		t.Fatalf("create knowledge base: %v", err)
	}

	document, err := service.UploadDocument(agent, UploadDocumentInput{
		KnowledgeBaseID: knowledgeBase.ID,
		Filename:        "vpn-guide.txt",
		ContentType:     "text/plain",
		Content:         []byte("first paragraph\n\nsecond paragraph"),
	})
	if err != nil {
		t.Fatalf("upload document: %v", err)
	}

	task, err := service.ProcessTask(context.Background(), document.LastTaskID)
	if err != nil {
		t.Fatalf("process task: %v", err)
	}
	if task.Status != ProcessingTaskStatusSucceeded {
		t.Fatalf("expected succeeded task, got %q", task.Status)
	}
	if task.Stage != ProcessingStageIndex {
		t.Fatalf("expected final stage index, got %q", task.Stage)
	}

	storedDocument, err := service.GetDocument(agent, document.ID)
	if err != nil {
		t.Fatalf("get document: %v", err)
	}
	if storedDocument.Status != DocumentStatusReady {
		t.Fatalf("expected ready document, got %q", storedDocument.Status)
	}
	if storedDocument.ChunkCount == 0 {
		t.Fatalf("expected chunk count to be updated")
	}
	if storedDocument.LastError != "" {
		t.Fatalf("expected last error to be cleared, got %q", storedDocument.LastError)
	}

	chunks := chunkRepository.ListByDocument(document.ID)
	if len(chunks) == 0 {
		t.Fatalf("expected indexed chunks to be stored")
	}
}

func TestProcessTaskFailureAllowsRetry(t *testing.T) {
	parser := &switchableDocumentParser{err: errors.New("parse failed")}
	taskRepository := NewMemoryDocumentProcessingTaskRepository()
	chunkRepository := NewMemoryDocumentChunkRepository()
	dispatcher := &stubProcessingTaskDispatcher{}
	service := NewService(
		NewMemoryKnowledgeBaseRepository(),
		NewMemoryDocumentRepository(),
		NewMemoryObjectStorage(),
		WithProcessingPipeline(ProcessingDependencies{
			TaskRepository: taskRepository,
			Dispatcher:     dispatcher,
			Parser:         parser,
			Chunker:        stubDocumentChunker{},
			Indexer:        stubChunkIndexer{repository: chunkRepository},
			MaxAttempts:    3,
		}),
	)

	agent := identity.IdentityContext{
		UserID:         "user-agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}

	knowledgeBase, err := service.CreateKnowledgeBase(agent, CreateKnowledgeBaseInput{Name: "IT 支持知识库"})
	if err != nil {
		t.Fatalf("create knowledge base: %v", err)
	}

	document, err := service.UploadDocument(agent, UploadDocumentInput{
		KnowledgeBaseID: knowledgeBase.ID,
		Filename:        "vpn-guide.txt",
		ContentType:     "text/plain",
		Content:         []byte("first paragraph"),
	})
	if err != nil {
		t.Fatalf("upload document: %v", err)
	}

	if _, err := service.ProcessTask(context.Background(), document.LastTaskID); err == nil {
		t.Fatalf("expected process task to fail")
	}

	failedDocument, err := service.GetDocument(agent, document.ID)
	if err != nil {
		t.Fatalf("get failed document: %v", err)
	}
	if failedDocument.Status != DocumentStatusFailed {
		t.Fatalf("expected failed document, got %q", failedDocument.Status)
	}
	if failedDocument.LastError == "" {
		t.Fatalf("expected failed document to record last error")
	}

	parser.err = nil
	retryTask, err := service.RetryDocumentProcessing(agent, document.ID)
	if err != nil {
		t.Fatalf("retry document processing: %v", err)
	}
	if retryTask.Attempt != 2 {
		t.Fatalf("expected retry attempt 2, got %d", retryTask.Attempt)
	}
	if len(dispatcher.enqueuedTaskIDs) != 2 {
		t.Fatalf("expected two enqueue events, got %d", len(dispatcher.enqueuedTaskIDs))
	}

	if _, err := service.ProcessTask(context.Background(), retryTask.ID); err != nil {
		t.Fatalf("process retry task: %v", err)
	}

	readyDocument, err := service.GetDocument(agent, document.ID)
	if err != nil {
		t.Fatalf("get ready document: %v", err)
	}
	if readyDocument.Status != DocumentStatusReady {
		t.Fatalf("expected ready document after retry, got %q", readyDocument.Status)
	}
	if readyDocument.ProcessingAttempts != 2 {
		t.Fatalf("expected processing attempts 2, got %d", readyDocument.ProcessingAttempts)
	}

	tasks := taskRepository.ListByDocument(document.ID)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks after retry, got %d", len(tasks))
	}
}

type stubProcessingTaskDispatcher struct {
	enqueuedTaskIDs []string
	err             error
}

func (dispatcher *stubProcessingTaskDispatcher) Enqueue(taskID string) error {
	if dispatcher.err != nil {
		return dispatcher.err
	}
	dispatcher.enqueuedTaskIDs = append(dispatcher.enqueuedTaskIDs, taskID)
	return nil
}

type stubDocumentParser struct{}

func (stubDocumentParser) Parse(_ context.Context, _ Document, object StoredObject) (ParsedDocument, error) {
	return ParsedDocument{PlainText: string(object.Content)}, nil
}

type switchableDocumentParser struct {
	err error
}

func (parser *switchableDocumentParser) Parse(_ context.Context, _ Document, object StoredObject) (ParsedDocument, error) {
	if parser.err != nil {
		return ParsedDocument{}, parser.err
	}
	return ParsedDocument{PlainText: string(object.Content)}, nil
}

type stubDocumentChunker struct{}

func (stubDocumentChunker) Chunk(_ context.Context, document Document, parsed ParsedDocument) ([]DocumentChunk, error) {
	return []DocumentChunk{NewDocumentChunk(document, 1, parsed.PlainText)}, nil
}

type stubChunkIndexer struct {
	repository DocumentChunkRepository
}

func (indexer stubChunkIndexer) ReplaceDocumentChunks(_ context.Context, document Document, chunks []DocumentChunk) error {
	indexer.repository.ReplaceByDocument(document.ID, chunks)
	return nil
}
