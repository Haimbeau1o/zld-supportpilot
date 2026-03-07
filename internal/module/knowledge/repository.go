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
