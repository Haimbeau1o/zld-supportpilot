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
