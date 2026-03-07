package knowledge

import (
	"fmt"
	"sort"
	"sync"
)

// MemoryKnowledgeBaseRepository 使用内存维护知识库元数据，当前阶段用于先稳定服务边界和测试行为。
type MemoryKnowledgeBaseRepository struct {
	mutex           sync.RWMutex
	basesByID       map[string]KnowledgeBase
	nextKnowledgeID uint64
}

func NewMemoryKnowledgeBaseRepository() *MemoryKnowledgeBaseRepository {
	return &MemoryKnowledgeBaseRepository{
		basesByID: make(map[string]KnowledgeBase),
	}
}

func (repository *MemoryKnowledgeBaseRepository) Save(knowledgeBase KnowledgeBase) KnowledgeBase {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	if knowledgeBase.ID == "" {
		repository.nextKnowledgeID++
		knowledgeBase.ID = fmt.Sprintf("kb-%d", repository.nextKnowledgeID)
	}

	repository.basesByID[knowledgeBase.ID] = knowledgeBase
	return knowledgeBase
}

func (repository *MemoryKnowledgeBaseRepository) FindByID(knowledgeBaseID string) (KnowledgeBase, bool) {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	knowledgeBase, ok := repository.basesByID[knowledgeBaseID]
	return knowledgeBase, ok
}

func (repository *MemoryKnowledgeBaseRepository) ListByOrganization(organizationID string) []KnowledgeBase {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	knowledgeBases := make([]KnowledgeBase, 0)
	for _, knowledgeBase := range repository.basesByID {
		if knowledgeBase.OrganizationID == organizationID {
			knowledgeBases = append(knowledgeBases, knowledgeBase)
		}
	}

	sort.Slice(knowledgeBases, func(left, right int) bool {
		if knowledgeBases[left].CreatedAt.Equal(knowledgeBases[right].CreatedAt) {
			return knowledgeBases[left].ID < knowledgeBases[right].ID
		}
		return knowledgeBases[left].CreatedAt.Before(knowledgeBases[right].CreatedAt)
	})

	return knowledgeBases
}

// MemoryDocumentRepository 使用内存维护文档元数据。
type MemoryDocumentRepository struct {
	mutex         sync.RWMutex
	documentsByID map[string]Document
	nextDocSeq    uint64
}

func NewMemoryDocumentRepository() *MemoryDocumentRepository {
	return &MemoryDocumentRepository{
		documentsByID: make(map[string]Document),
	}
}

func (repository *MemoryDocumentRepository) Save(document Document) Document {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	if document.ID == "" {
		repository.nextDocSeq++
		document.ID = fmt.Sprintf("doc-%d", repository.nextDocSeq)
	}

	repository.documentsByID[document.ID] = document
	return document
}

func (repository *MemoryDocumentRepository) FindByID(documentID string) (Document, bool) {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	document, ok := repository.documentsByID[documentID]
	return document, ok
}

func (repository *MemoryDocumentRepository) ListByKnowledgeBase(knowledgeBaseID string) []Document {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	documents := make([]Document, 0)
	for _, document := range repository.documentsByID {
		if document.KnowledgeBaseID == knowledgeBaseID {
			documents = append(documents, document)
		}
	}

	sort.Slice(documents, func(left, right int) bool {
		if documents[left].CreatedAt.Equal(documents[right].CreatedAt) {
			return documents[left].ID < documents[right].ID
		}
		return documents[left].CreatedAt.Before(documents[right].CreatedAt)
	})

	return documents
}

// MemoryDocumentProcessingTaskRepository 使用内存记录每次处理尝试。
type MemoryDocumentProcessingTaskRepository struct {
	mutex     sync.RWMutex
	tasksByID map[string]DocumentProcessingTask
}

func NewMemoryDocumentProcessingTaskRepository() *MemoryDocumentProcessingTaskRepository {
	return &MemoryDocumentProcessingTaskRepository{
		tasksByID: make(map[string]DocumentProcessingTask),
	}
}

func (repository *MemoryDocumentProcessingTaskRepository) Save(task DocumentProcessingTask) DocumentProcessingTask {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	repository.tasksByID[task.ID] = task
	return task
}

func (repository *MemoryDocumentProcessingTaskRepository) FindByID(taskID string) (DocumentProcessingTask, bool) {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	task, ok := repository.tasksByID[taskID]
	return task, ok
}

func (repository *MemoryDocumentProcessingTaskRepository) FindLatestByDocument(documentID string) (DocumentProcessingTask, bool) {
	tasks := repository.ListByDocument(documentID)
	if len(tasks) == 0 {
		return DocumentProcessingTask{}, false
	}
	return tasks[len(tasks)-1], true
}

func (repository *MemoryDocumentProcessingTaskRepository) ListByDocument(documentID string) []DocumentProcessingTask {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	tasks := make([]DocumentProcessingTask, 0)
	for _, task := range repository.tasksByID {
		if task.DocumentID == documentID {
			tasks = append(tasks, task)
		}
	}

	sort.Slice(tasks, func(left, right int) bool {
		if tasks[left].CreatedAt.Equal(tasks[right].CreatedAt) {
			return tasks[left].ID < tasks[right].ID
		}
		return tasks[left].CreatedAt.Before(tasks[right].CreatedAt)
	})

	return tasks
}

// MemoryDocumentChunkRepository 使用内存维护切块结果，为后续检索能力保留稳定承接点。
type MemoryDocumentChunkRepository struct {
	mutex            sync.RWMutex
	chunksByDocument map[string][]DocumentChunk
}

func NewMemoryDocumentChunkRepository() *MemoryDocumentChunkRepository {
	return &MemoryDocumentChunkRepository{
		chunksByDocument: make(map[string][]DocumentChunk),
	}
}

func (repository *MemoryDocumentChunkRepository) ReplaceByDocument(documentID string, chunks []DocumentChunk) {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	copied := make([]DocumentChunk, 0, len(chunks))
	for _, chunk := range chunks {
		copied = append(copied, chunk)
	}
	repository.chunksByDocument[documentID] = copied
}

func (repository *MemoryDocumentChunkRepository) ListByDocument(documentID string) []DocumentChunk {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	chunks := repository.chunksByDocument[documentID]
	copied := make([]DocumentChunk, 0, len(chunks))
	for _, chunk := range chunks {
		copied = append(copied, chunk)
	}
	return copied
}

func (repository *MemoryDocumentChunkRepository) ListByKnowledgeBase(knowledgeBaseID string) []DocumentChunk {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	chunks := make([]DocumentChunk, 0)
	for _, documentChunks := range repository.chunksByDocument {
		for _, chunk := range documentChunks {
			if chunk.KnowledgeBaseID == knowledgeBaseID {
				chunks = append(chunks, chunk)
			}
		}
	}

	sort.Slice(chunks, func(left, right int) bool {
		if chunks[left].CreatedAt.Equal(chunks[right].CreatedAt) {
			return chunks[left].ID < chunks[right].ID
		}
		return chunks[left].CreatedAt.Before(chunks[right].CreatedAt)
	})

	return chunks
}
