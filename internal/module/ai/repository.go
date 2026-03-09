package ai

// UnifiedIntakeRepository 定义统一受理主记录的访问能力。
type UnifiedIntakeRepository interface {
	Save(record UnifiedIntakeRecord) UnifiedIntakeRecord
	FindByID(intakeID string) (UnifiedIntakeRecord, bool)
}

// AnswerFeedbackRepository 定义答案反馈记录的访问能力。
type AnswerFeedbackRepository interface {
	Save(feedback AnswerFeedback) AnswerFeedback
	FindByIntakeAndActor(intakeID, actorID string) (AnswerFeedback, bool)
	ListByOrganization(organizationID string) []AnswerFeedback
}
