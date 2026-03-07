package http

import (
	"encoding/json"
	"errors"
	"fmt"
	stdhttp "net/http"
	"strings"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ticket"
)

type ticketService interface {
	CreateTicket(actor identity.IdentityContext, input ticket.CreateTicketInput) (ticket.Ticket, error)
	GetTicket(actor identity.IdentityContext, ticketID string) (ticket.Ticket, error)
	ListTickets(actor identity.IdentityContext, input ticket.ListTicketsInput) ([]ticket.Ticket, error)
	AssignTicket(actor identity.IdentityContext, input ticket.AssignTicketInput) (ticket.Ticket, error)
	TransitionTicketStatus(actor identity.IdentityContext, input ticket.TransitionTicketStatusInput) (ticket.Ticket, error)
}

type TicketDependencies struct {
	TicketService ticketService
}

type TicketHandler struct {
	ticketService ticketService
}

type createTicketRequest struct {
	Title       string                `json:"title"`
	Description string                `json:"description"`
	Category    string                `json:"category"`
	Priority    ticket.TicketPriority `json:"priority"`
}

type assignTicketRequest struct {
	AssigneeID string `json:"assignee_id"`
}

type transitionTicketStatusRequest struct {
	Status ticket.TicketStatus `json:"status"`
}

type ticketResponse struct {
	ID             string                `json:"id"`
	OrganizationID string                `json:"organization_id"`
	RequesterID    string                `json:"requester_id"`
	AssigneeID     string                `json:"assignee_id"`
	Title          string                `json:"title"`
	Description    string                `json:"description"`
	Category       string                `json:"category"`
	Priority       ticket.TicketPriority `json:"priority"`
	Status         ticket.TicketStatus   `json:"status"`
}

type ticketListResponse struct {
	Items []ticketResponse `json:"items"`
}

func NewTicketHandler(ticketService ticketService) TicketHandler {
	return TicketHandler{ticketService: ticketService}
}

func (handler TicketHandler) Create(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	actor, ok := identityContextFromContext(request.Context())
	if !ok {
		writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "当前请求缺少身份上下文")
		return
	}

	var input createTicketRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		writeError(writer, stdhttp.StatusBadRequest, "invalid_request", "请求体不是合法的 JSON")
		return
	}

	createdTicket, err := handler.ticketService.CreateTicket(actor, ticket.CreateTicketInput{
		Title:       input.Title,
		Description: input.Description,
		Category:    input.Category,
		Priority:    input.Priority,
	})
	if err != nil {
		handleTicketError(writer, err, "创建工单失败")
		return
	}

	writeJSON(writer, stdhttp.StatusCreated, newTicketResponse(createdTicket))
}

func (handler TicketHandler) List(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	actor, ok := identityContextFromContext(request.Context())
	if !ok {
		writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "当前请求缺少身份上下文")
		return
	}

	tickets, err := handler.ticketService.ListTickets(actor, ticket.ListTicketsInput{})
	if err != nil {
		handleTicketError(writer, err, "查询工单失败")
		return
	}

	items := make([]ticketResponse, 0, len(tickets))
	for _, item := range tickets {
		items = append(items, newTicketResponse(item))
	}

	writeJSON(writer, stdhttp.StatusOK, ticketListResponse{Items: items})
}

func (handler TicketHandler) Detail(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	actor, ok := identityContextFromContext(request.Context())
	if !ok {
		writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "当前请求缺少身份上下文")
		return
	}

	ticketID := strings.TrimSpace(request.PathValue("id"))
	currentTicket, err := handler.ticketService.GetTicket(actor, ticketID)
	if err != nil {
		handleTicketError(writer, err, "查询工单详情失败")
		return
	}

	writeJSON(writer, stdhttp.StatusOK, newTicketResponse(currentTicket))
}

func (handler TicketHandler) Assign(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	actor, ok := identityContextFromContext(request.Context())
	if !ok {
		writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "当前请求缺少身份上下文")
		return
	}

	var input assignTicketRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		writeError(writer, stdhttp.StatusBadRequest, "invalid_request", "请求体不是合法的 JSON")
		return
	}

	updatedTicket, err := handler.ticketService.AssignTicket(actor, ticket.AssignTicketInput{
		TicketID:   strings.TrimSpace(request.PathValue("id")),
		AssigneeID: input.AssigneeID,
	})
	if err != nil {
		handleTicketError(writer, err, "指派工单失败")
		return
	}

	writeJSON(writer, stdhttp.StatusOK, newTicketResponse(updatedTicket))
}

func (handler TicketHandler) TransitionStatus(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	actor, ok := identityContextFromContext(request.Context())
	if !ok {
		writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "当前请求缺少身份上下文")
		return
	}

	var input transitionTicketStatusRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		writeError(writer, stdhttp.StatusBadRequest, "invalid_request", "请求体不是合法的 JSON")
		return
	}

	updatedTicket, err := handler.ticketService.TransitionTicketStatus(actor, ticket.TransitionTicketStatusInput{
		TicketID: strings.TrimSpace(request.PathValue("id")),
		ToStatus: input.Status,
	})
	if err != nil {
		handleTicketError(writer, err, "工单状态流转失败")
		return
	}

	writeJSON(writer, stdhttp.StatusOK, newTicketResponse(updatedTicket))
}

func newTicketResponse(currentTicket ticket.Ticket) ticketResponse {
	return ticketResponse{
		ID:             currentTicket.ID,
		OrganizationID: currentTicket.OrganizationID,
		RequesterID:    currentTicket.RequesterID,
		AssigneeID:     currentTicket.AssigneeID,
		Title:          currentTicket.Title,
		Description:    currentTicket.Description,
		Category:       currentTicket.Category,
		Priority:       currentTicket.Priority,
		Status:         currentTicket.Status,
	}
}

func handleTicketError(writer stdhttp.ResponseWriter, err error, fallbackMessage string) {
	switch {
	case errors.Is(err, ticket.ErrTicketForbidden):
		writeError(writer, stdhttp.StatusForbidden, "forbidden", "当前身份没有权限执行该工单操作")
	case errors.Is(err, ticket.ErrTicketNotFound):
		writeError(writer, stdhttp.StatusNotFound, "ticket_not_found", "工单不存在")
	case errors.Is(err, ticket.ErrInvalidTicketInput), errors.Is(err, ticket.ErrInvalidStatusTransition):
		writeError(writer, stdhttp.StatusBadRequest, "invalid_ticket_request", err.Error())
	default:
		writeError(writer, stdhttp.StatusInternalServerError, "internal_error", fallbackMessage)
	}
}

func registerTicketRoutes(mux *stdhttp.ServeMux, authDependencies AuthDependencies, ticketDependencies TicketDependencies) error {
	if authDependencies.TokenManager == nil {
		return fmt.Errorf("token manager is required for ticket routes")
	}
	if ticketDependencies.TicketService == nil {
		return fmt.Errorf("ticket service is required")
	}

	authMiddleware := NewAuthMiddleware(authDependencies.TokenManager)
	ticketHandler := NewTicketHandler(ticketDependencies.TicketService)

	// 工单接口全部挂在鉴权中间件之后，避免未来再出现“接口存在但忘了套身份边界”的隐性漏洞。
	mux.Handle("POST /api/v1/tickets", authMiddleware.RequireIdentity(stdhttp.HandlerFunc(ticketHandler.Create)))
	mux.Handle("GET /api/v1/tickets", authMiddleware.RequireIdentity(stdhttp.HandlerFunc(ticketHandler.List)))
	mux.Handle("GET /api/v1/tickets/{id}", authMiddleware.RequireIdentity(stdhttp.HandlerFunc(ticketHandler.Detail)))
	mux.Handle("POST /api/v1/tickets/{id}/assign", authMiddleware.RequireIdentity(stdhttp.HandlerFunc(ticketHandler.Assign)))
	mux.Handle("POST /api/v1/tickets/{id}/status", authMiddleware.RequireIdentity(stdhttp.HandlerFunc(ticketHandler.TransitionStatus)))
	return nil
}
