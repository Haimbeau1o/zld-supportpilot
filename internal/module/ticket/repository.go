package ticket

// TicketRepository 定义工单数据访问能力。
type TicketRepository interface {
	Save(ticket Ticket) Ticket
	FindByID(ticketID string) (Ticket, bool)
	ListByOrganization(organizationID string) []Ticket
}
