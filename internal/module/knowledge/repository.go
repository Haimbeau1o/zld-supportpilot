package knowledge

// KnowledgeBaseRepository 定义知识库元数据访问能力。
type KnowledgeBaseRepository interface {
	Save(knowledgeBase KnowledgeBase) KnowledgeBase
	FindByID(knowledgeBaseID string) (KnowledgeBase, bool)
	ListByOrganization(organizationID string) []KnowledgeBase
}

// DocumentRepository 定义文档元数据访问能力。
type DocumentRepository interface {
	Save(document Document) Document
	FindByID(documentID string) (Document, bool)
	ListByKnowledgeBase(knowledgeBaseID string) []Document
}

// DocumentProcessingTaskRepository 定义处理任务访问能力。
type DocumentProcessingTaskRepository interface {
	Save(task DocumentProcessingTask) DocumentProcessingTask
	FindByID(taskID string) (DocumentProcessingTask, bool)
	FindLatestByDocument(documentID string) (DocumentProcessingTask, bool)
	ListByDocument(documentID string) []DocumentProcessingTask
}

// DocumentChunkRepository 定义切块索引结果访问能力。
type DocumentChunkRepository interface {
	ReplaceByDocument(documentID string, chunks []DocumentChunk)
	ListByDocument(documentID string) []DocumentChunk
	ListByKnowledgeBase(knowledgeBaseID string) []DocumentChunk
}
