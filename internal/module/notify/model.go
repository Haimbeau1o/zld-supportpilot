package notify

import "time"

// EventType 表示通知事件类型，保持最小稳定集合，便于后续扩展更多渠道与业务动作。
type EventType string

const (
	EventTypeTicketAssigned      EventType = "ticket_assigned"
	EventTypeTicketStatusChanged EventType = "ticket_status_changed"
	EventTypeAIEscalated         EventType = "ai_escalated"
)

// Event 表示业务侧发布给通知模块的统一事件。
type Event struct {
	Type             EventType
	OrganizationID   string
	TicketID         string
	ActorID          string
	RecipientUserIDs []string
	Subject          string
	Content          string
	Metadata         map[string]string
	CreatedAt        time.Time
}

// User 是通知模块关心的最小用户视图，只保留投递真正需要的字段。
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Request 是交给具体渠道的统一投递请求。
type Request struct {
	Event      Event
	Recipients []User
}

// Mail 表示邮件渠道需要发送的标准化消息。
type Mail struct {
	To       []string          `json:"to"`
	Subject  string            `json:"subject"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata,omitempty"`
}
