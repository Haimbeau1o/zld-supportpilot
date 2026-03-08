package ticket

// TicketRepository 定义工单主模型的访问能力。
type TicketRepository interface {
	Save(ticket Ticket) Ticket
	FindByID(ticketID string) (Ticket, bool)
	ListByOrganization(organizationID string) []Ticket
}

// TicketCommentRepository 定义工单评论与内部备注的访问能力。
type TicketCommentRepository interface {
	Save(comment TicketComment) TicketComment
	ListByTicket(ticketID string) []TicketComment
}

// TicketAuditEventRepository 定义工单审计事件的访问能力。
type TicketAuditEventRepository interface {
	Save(event TicketAuditEvent) TicketAuditEvent
	ListByTicket(ticketID string) []TicketAuditEvent
}
