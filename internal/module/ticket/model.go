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

func InitialTicketStatus() TicketStatus {
	return TicketStatusOpen
}
