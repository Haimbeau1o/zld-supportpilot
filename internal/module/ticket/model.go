package ticket

import "time"

// TicketStatus 表示工单在主流程中的生命周期状态。
type TicketStatus string

const (
	TicketStatusOpen       TicketStatus = "open"
	TicketStatusInProgress TicketStatus = "in_progress"
	TicketStatusResolved   TicketStatus = "resolved"
	TicketStatusClosed     TicketStatus = "closed"
)

// TicketPriority 表示工单优先级。当前阶段先固定最小集合，避免优先级命名在主链路未稳定前漂移。
type TicketPriority string

const (
	TicketPriorityLow    TicketPriority = "low"
	TicketPriorityMedium TicketPriority = "medium"
	TicketPriorityHigh   TicketPriority = "high"
	TicketPriorityUrgent TicketPriority = "urgent"
)

// TicketCommentType 表示工单协作内容的可见性类型。
type TicketCommentType string

const (
	TicketCommentTypeComment      TicketCommentType = "comment"
	TicketCommentTypeInternalNote TicketCommentType = "internal_note"
)

// TicketAuditEventType 表示工单关键动作的审计事件类型。
type TicketAuditEventType string

const (
	TicketAuditEventTypeCreated           TicketAuditEventType = "created"
	TicketAuditEventTypeAssignmentChanged TicketAuditEventType = "assignment_changed"
	TicketAuditEventTypeStatusChanged     TicketAuditEventType = "status_changed"
)

// TicketTimelineItemType 表示时间线条目的来源类型。
type TicketTimelineItemType string

const (
	TicketTimelineItemTypeComment    TicketTimelineItemType = "comment"
	TicketTimelineItemTypeAuditEvent TicketTimelineItemType = "audit_event"
)

// Ticket 是当前阶段的工单主模型，只保留后续主流程真正依赖的核心字段。
type Ticket struct {
	ID             string
	OrganizationID string
	RequesterID    string
	AssigneeID     string
	Title          string
	Description    string
	Category       string
	Priority       TicketPriority
	Status         TicketStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// TicketComment 表示工单下的协作评论或内部备注。
type TicketComment struct {
	ID             string
	TicketID       string
	OrganizationID string
	AuthorID       string
	Type           TicketCommentType
	Content        string
	CreatedAt      time.Time
}

// TicketAuditEvent 表示工单关键动作的可审计记录。
type TicketAuditEvent struct {
	ID             string
	TicketID       string
	OrganizationID string
	ActorID        string
	Type           TicketAuditEventType
	Content        string
	FromValue      string
	ToValue        string
	CreatedAt      time.Time
}

// TicketTimelineItem 是统一输出给前端或 API 的时间线条目。
type TicketTimelineItem struct {
	ID             string
	TicketID       string
	OrganizationID string
	ActorID        string
	ItemType       TicketTimelineItemType
	CommentType    TicketCommentType
	AuditEventType TicketAuditEventType
	Content        string
	FromValue      string
	ToValue        string
	CreatedAt      time.Time
}

func InitialTicketStatus() TicketStatus {
	return TicketStatusOpen
}
