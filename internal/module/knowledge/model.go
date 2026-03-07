package knowledge

import (
	"fmt"
	"time"
)

// DocumentStatus 表示文档在知识处理链路中的状态。
type DocumentStatus string

const (
	DocumentStatusUploaded   DocumentStatus = "uploaded"
	DocumentStatusProcessing DocumentStatus = "processing"
	DocumentStatusReady      DocumentStatus = "ready"
	DocumentStatusFailed     DocumentStatus = "failed"
)

// ProcessingTaskStatus 表示单次处理任务的执行状态。
type ProcessingTaskStatus string

const (
	ProcessingTaskStatusQueued    ProcessingTaskStatus = "queued"
	ProcessingTaskStatusRunning   ProcessingTaskStatus = "running"
	ProcessingTaskStatusSucceeded ProcessingTaskStatus = "succeeded"
	ProcessingTaskStatusFailed    ProcessingTaskStatus = "failed"
)

// ProcessingStage 表示处理任务当前推进到的阶段。
type ProcessingStage string

const (
	ProcessingStageParse ProcessingStage = "parse"
	ProcessingStageChunk ProcessingStage = "chunk"
	ProcessingStageIndex ProcessingStage = "index"
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
	ID                  string
	KnowledgeBaseID     string
	OrganizationID      string
	Filename            string
	ContentType         string
	SizeBytes           int64
	StorageKey          string
	Status              DocumentStatus
	UploadedBy          string
	LastTaskID          string
	ProcessingAttempts  int
	ChunkCount          int
	LastError           string
	ProcessingQueuedAt  time.Time
	ProcessingStartedAt time.Time
	ProcessedAt         time.Time
	FailedAt            time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// DocumentProcessingTask 记录单次文档处理尝试的执行轨迹。
type DocumentProcessingTask struct {
	ID              string
	DocumentID      string
	KnowledgeBaseID string
	OrganizationID  string
	Status          ProcessingTaskStatus
	Stage           ProcessingStage
	Attempt         int
	MaxAttempts     int
	ErrorMessage    string
	CreatedAt       time.Time
	StartedAt       time.Time
	FinishedAt      time.Time
	UpdatedAt       time.Time
}

// DocumentChunk 表示切块后的索引单元；当前阶段先稳定结构，后续 #7 再接向量检索。
type DocumentChunk struct {
	ID              string
	DocumentID      string
	KnowledgeBaseID string
	OrganizationID  string
	Sequence        int
	Content         string
	CreatedAt       time.Time
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

func NewDocumentProcessingTask(document Document, attempt int, maxAttempts int) DocumentProcessingTask {
	now := time.Now()
	if attempt < 1 {
		attempt = 1
	}
	if maxAttempts < attempt {
		maxAttempts = attempt
	}

	taskID := fmt.Sprintf("%s-process-%d", document.ID, attempt)
	if document.ID == "" {
		taskID = fmt.Sprintf("process-%d", attempt)
	}

	return DocumentProcessingTask{
		ID:              taskID,
		DocumentID:      document.ID,
		KnowledgeBaseID: document.KnowledgeBaseID,
		OrganizationID:  document.OrganizationID,
		Status:          ProcessingTaskStatusQueued,
		Stage:           ProcessingStageParse,
		Attempt:         attempt,
		MaxAttempts:     maxAttempts,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func NewDocumentChunk(document Document, sequence int, content string) DocumentChunk {
	now := time.Now()
	return DocumentChunk{
		ID:              fmt.Sprintf("%s-chunk-%d", document.ID, sequence),
		DocumentID:      document.ID,
		KnowledgeBaseID: document.KnowledgeBaseID,
		OrganizationID:  document.OrganizationID,
		Sequence:        sequence,
		Content:         content,
		CreatedAt:       now,
	}
}
