package knowledge

import "time"

// DocumentStatus 表示文档在知识处理链路中的状态。
type DocumentStatus string

const (
	DocumentStatusUploaded   DocumentStatus = "uploaded"
	DocumentStatusProcessing DocumentStatus = "processing"
	DocumentStatusReady      DocumentStatus = "ready"
	DocumentStatusFailed     DocumentStatus = "failed"
)

// KnowledgeBase 是租户下的知识容器，后续文档、切块和检索都以它作为归属边界。
type KnowledgeBase struct {
	ID             string
	OrganizationID string
	Name           string
	Description    string
	CreatedBy      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Document 是原始文档的元数据记录。
type Document struct {
	ID              string
	KnowledgeBaseID string
	OrganizationID  string
	Filename        string
	ContentType     string
	SizeBytes       int64
	StorageKey      string
	Status          DocumentStatus
	UploadedBy      string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NewDocument(document Document) Document {
	now := time.Now()
	if document.Status == "" {
		// 当前阶段即使只实现上传，也提前保留 processing/ready/failed 扩展位，为后续 #6 异步处理稳定状态模型。
		document.Status = DocumentStatusUploaded
	}
	if document.CreatedAt.IsZero() {
		document.CreatedAt = now
	}
	if document.UpdatedAt.IsZero() {
		document.UpdatedAt = document.CreatedAt
	}

	return document
}
