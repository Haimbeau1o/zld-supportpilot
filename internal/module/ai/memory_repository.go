package ai

import (
	"fmt"
	"sort"
	"sync"
)

// MemoryUnifiedIntakeRepository 使用内存保存统一受理主记录，便于本地开发与测试稳定复现行为。
type MemoryUnifiedIntakeRepository struct {
	mutex         sync.RWMutex
	intakesByID   map[string]UnifiedIntakeRecord
	nextIntakeSeq uint64
}

func NewMemoryUnifiedIntakeRepository() *MemoryUnifiedIntakeRepository {
	return &MemoryUnifiedIntakeRepository{
		intakesByID: make(map[string]UnifiedIntakeRecord),
	}
}

func (repository *MemoryUnifiedIntakeRepository) Save(record UnifiedIntakeRecord) UnifiedIntakeRecord {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	if record.ID == "" {
		repository.nextIntakeSeq++
		record.ID = fmt.Sprintf("intake-%d", repository.nextIntakeSeq)
	}

	repository.intakesByID[record.ID] = record
	return record
}

func (repository *MemoryUnifiedIntakeRepository) FindByID(intakeID string) (UnifiedIntakeRecord, bool) {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	record, ok := repository.intakesByID[intakeID]
	return record, ok
}

// MemoryAnswerFeedbackRepository 使用内存保存反馈记录，并按 intake + actor 维持最新一份反馈。
type MemoryAnswerFeedbackRepository struct {
	mutex                   sync.RWMutex
	feedbacksByID           map[string]AnswerFeedback
	feedbackIDByIntakeActor map[string]string
	nextFeedbackSeq         uint64
}

func NewMemoryAnswerFeedbackRepository() *MemoryAnswerFeedbackRepository {
	return &MemoryAnswerFeedbackRepository{
		feedbacksByID:           make(map[string]AnswerFeedback),
		feedbackIDByIntakeActor: make(map[string]string),
	}
}

func (repository *MemoryAnswerFeedbackRepository) Save(feedback AnswerFeedback) AnswerFeedback {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	key := repository.feedbackKey(feedback.IntakeID, feedback.ActorID)
	if feedback.ID == "" {
		if existingID, ok := repository.feedbackIDByIntakeActor[key]; ok {
			feedback.ID = existingID
			if existing, ok := repository.feedbacksByID[existingID]; ok && feedback.CreatedAt.IsZero() {
				feedback.CreatedAt = existing.CreatedAt
			}
		} else {
			repository.nextFeedbackSeq++
			feedback.ID = fmt.Sprintf("feedback-%d", repository.nextFeedbackSeq)
		}
	}

	repository.feedbacksByID[feedback.ID] = feedback
	repository.feedbackIDByIntakeActor[key] = feedback.ID
	return feedback
}

func (repository *MemoryAnswerFeedbackRepository) FindByIntakeAndActor(intakeID, actorID string) (AnswerFeedback, bool) {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	feedbackID, ok := repository.feedbackIDByIntakeActor[repository.feedbackKey(intakeID, actorID)]
	if !ok {
		return AnswerFeedback{}, false
	}
	feedback, ok := repository.feedbacksByID[feedbackID]
	return feedback, ok
}

func (repository *MemoryAnswerFeedbackRepository) ListByOrganization(organizationID string) []AnswerFeedback {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	items := make([]AnswerFeedback, 0)
	for _, feedback := range repository.feedbacksByID {
		if feedback.OrganizationID == organizationID {
			items = append(items, feedback)
		}
	}

	sort.Slice(items, func(left, right int) bool {
		if items[left].CreatedAt.Equal(items[right].CreatedAt) {
			return items[left].ID < items[right].ID
		}
		return items[left].CreatedAt.Before(items[right].CreatedAt)
	})

	return items
}

func (repository *MemoryAnswerFeedbackRepository) feedbackKey(intakeID, actorID string) string {
	return intakeID + "::" + actorID
}
