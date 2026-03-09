package knowledge

import (
	"fmt"
	"sort"
	"sync"
)

// KnowledgeCandidateRepository 定义知识候选元数据的访问能力。
type KnowledgeCandidateRepository interface {
	Save(candidate KnowledgeCandidate) KnowledgeCandidate
	FindByID(candidateID string) (KnowledgeCandidate, bool)
	ListByKnowledgeBase(knowledgeBaseID string) []KnowledgeCandidate
}

// MemoryKnowledgeCandidateRepository 使用内存维护知识候选，支撑单测与本地开发模式。
type MemoryKnowledgeCandidateRepository struct {
	mutex          sync.RWMutex
	candidatesByID map[string]KnowledgeCandidate
	nextSeq        uint64
}

func NewMemoryKnowledgeCandidateRepository() *MemoryKnowledgeCandidateRepository {
	return &MemoryKnowledgeCandidateRepository{
		candidatesByID: make(map[string]KnowledgeCandidate),
	}
}

func (repository *MemoryKnowledgeCandidateRepository) Save(candidate KnowledgeCandidate) KnowledgeCandidate {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	if candidate.ID == "" {
		repository.nextSeq++
		candidate.ID = fmt.Sprintf("kc-%d", repository.nextSeq)
	}

	repository.candidatesByID[candidate.ID] = candidate
	return candidate
}

func (repository *MemoryKnowledgeCandidateRepository) FindByID(candidateID string) (KnowledgeCandidate, bool) {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	candidate, ok := repository.candidatesByID[candidateID]
	return candidate, ok
}

func (repository *MemoryKnowledgeCandidateRepository) ListByKnowledgeBase(knowledgeBaseID string) []KnowledgeCandidate {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	candidates := make([]KnowledgeCandidate, 0)
	for _, candidate := range repository.candidatesByID {
		if candidate.KnowledgeBaseID == knowledgeBaseID {
			candidates = append(candidates, candidate)
		}
	}

	sort.Slice(candidates, func(left, right int) bool {
		if candidates[left].CreatedAt.Equal(candidates[right].CreatedAt) {
			return candidates[left].ID < candidates[right].ID
		}
		return candidates[left].CreatedAt.Before(candidates[right].CreatedAt)
	})

	return candidates
}
