package http

import (
	"bytes"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ticket"
	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/config"
)

type ticketResponsePayload struct {
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

type ticketListResponsePayload struct {
	Items []ticketResponsePayload `json:"items"`
}

type ticketCommentResponsePayload struct {
	ID        string                   `json:"id"`
	TicketID  string                   `json:"ticket_id"`
	AuthorID  string                   `json:"author_id"`
	Type      ticket.TicketCommentType `json:"type"`
	Content   string                   `json:"content"`
	CreatedAt string                   `json:"created_at"`
}

type ticketTimelineItemResponsePayload struct {
	ID             string                        `json:"id"`
	TicketID       string                        `json:"ticket_id"`
	ActorID        string                        `json:"actor_id"`
	ItemType       ticket.TicketTimelineItemType `json:"item_type"`
	CommentType    ticket.TicketCommentType      `json:"comment_type"`
	AuditEventType ticket.TicketAuditEventType   `json:"audit_event_type"`
	Content        string                        `json:"content"`
	FromValue      string                        `json:"from_value"`
	ToValue        string                        `json:"to_value"`
	CreatedAt      string                        `json:"created_at"`
}

type ticketTimelineResponsePayload struct {
	Items []ticketTimelineItemResponsePayload `json:"items"`
}

func TestTicketCreate(t *testing.T) {
	handler, tokenManager := newTestTicketMux(t)
	endUser := identity.IdentityContext{
		UserID:         "user-end-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}

	request := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/tickets",
		bytes.NewBufferString(`{"title":"VPN 无法连接","description":"今天上午开始无法连接公司 VPN","category":"network","priority":"high"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+issueToken(t, tokenManager, endUser))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusCreated, recorder.Code, recorder.Body.String())
	}

	var response ticketResponsePayload
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode ticket create response: %v", err)
	}

	if response.ID == "" {
		t.Fatalf("expected ticket id to be returned")
	}
	if response.RequesterID != endUser.UserID {
		t.Fatalf("expected requester id %q, got %q", endUser.UserID, response.RequesterID)
	}
	if response.Status != ticket.TicketStatusOpen {
		t.Fatalf("expected ticket status %q, got %q", ticket.TicketStatusOpen, response.Status)
	}
}

func TestTicketListOwn(t *testing.T) {
	handler, tokenManager := newTestTicketMux(t)
	endUser1 := identity.IdentityContext{
		UserID:         "user-end-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}
	endUser2 := identity.IdentityContext{
		UserID:         "user-end-2",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}

	createTicketViaHTTP(t, handler, tokenManager, endUser1, `{"title":"VPN 无法连接","description":"今天上午开始无法连接公司 VPN","category":"network","priority":"high"}`)
	createTicketViaHTTP(t, handler, tokenManager, endUser2, `{"title":"邮箱异常","description":"收不到邮件","category":"mail","priority":"medium"}`)

	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/tickets", nil)
	request.Header.Set("Authorization", "Bearer "+issueToken(t, tokenManager, endUser1))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response ticketListResponsePayload
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode ticket list response: %v", err)
	}

	if len(response.Items) != 1 {
		t.Fatalf("expected 1 ticket for end user, got %d", len(response.Items))
	}
	if response.Items[0].RequesterID != endUser1.UserID {
		t.Fatalf("expected end user to see only own ticket")
	}
}

func TestTicketDetailForAgent(t *testing.T) {
	handler, tokenManager := newTestTicketMux(t)
	endUser := identity.IdentityContext{
		UserID:         "user-end-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}
	agent := identity.IdentityContext{
		UserID:         "user-agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}

	createdTicket := createTicketViaHTTP(t, handler, tokenManager, endUser, `{"title":"共享盘无权限","description":"无法访问部门共享盘","category":"storage","priority":"high"}`)

	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/tickets/"+createdTicket.ID, nil)
	request.Header.Set("Authorization", "Bearer "+issueToken(t, tokenManager, agent))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusOK, recorder.Code, recorder.Body.String())
	}
}

func TestTicketAssign(t *testing.T) {
	handler, tokenManager := newTestTicketMux(t)
	endUser := identity.IdentityContext{
		UserID:         "user-end-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}
	agent := identity.IdentityContext{
		UserID:         "user-agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}

	createdTicket := createTicketViaHTTP(t, handler, tokenManager, endUser, `{"title":"账号锁定","description":"登录多次失败后被锁定","category":"account","priority":"urgent"}`)

	request := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/tickets/"+createdTicket.ID+"/assign",
		bytes.NewBufferString(`{"assignee_id":"user-agent-2"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+issueToken(t, tokenManager, agent))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response ticketResponsePayload
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode assign response: %v", err)
	}

	if response.AssigneeID != "user-agent-2" {
		t.Fatalf("expected assignee id user-agent-2, got %q", response.AssigneeID)
	}
}

func TestTicketStatusTransition(t *testing.T) {
	handler, tokenManager := newTestTicketMux(t)
	endUser := identity.IdentityContext{
		UserID:         "user-end-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}
	agent := identity.IdentityContext{
		UserID:         "user-agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}

	createdTicket := createTicketViaHTTP(t, handler, tokenManager, endUser, `{"title":"共享盘无权限","description":"无法访问部门共享盘","category":"storage","priority":"high"}`)

	request := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/tickets/"+createdTicket.ID+"/status",
		bytes.NewBufferString(`{"status":"in_progress"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+issueToken(t, tokenManager, agent))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response ticketResponsePayload
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode status transition response: %v", err)
	}

	if response.Status != ticket.TicketStatusInProgress {
		t.Fatalf("expected status %q, got %q", ticket.TicketStatusInProgress, response.Status)
	}
}

func TestTicketAddComment(t *testing.T) {
	handler, tokenManager := newTestTicketMux(t)
	endUser := identity.IdentityContext{
		UserID:         "user-end-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}

	createdTicket := createTicketViaHTTP(t, handler, tokenManager, endUser, `{"title":"VPN 无法连接","description":"今天上午开始无法连接公司 VPN","category":"network","priority":"high"}`)

	request := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/tickets/"+createdTicket.ID+"/comments",
		bytes.NewBufferString(`{"type":"comment","content":"已补充错误截图，请继续排查。"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+issueToken(t, tokenManager, endUser))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusCreated, recorder.Code, recorder.Body.String())
	}

	var response ticketCommentResponsePayload
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode add comment response: %v", err)
	}

	if response.TicketID != createdTicket.ID {
		t.Fatalf("expected ticket id %q, got %q", createdTicket.ID, response.TicketID)
	}
	if response.AuthorID != endUser.UserID {
		t.Fatalf("expected author id %q, got %q", endUser.UserID, response.AuthorID)
	}
	if response.Type != ticket.TicketCommentTypeComment {
		t.Fatalf("expected comment type %q, got %q", ticket.TicketCommentTypeComment, response.Type)
	}
}

func TestTicketEndUserCannotAddInternalNote(t *testing.T) {
	handler, tokenManager := newTestTicketMux(t)
	endUser := identity.IdentityContext{
		UserID:         "user-end-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}

	createdTicket := createTicketViaHTTP(t, handler, tokenManager, endUser, `{"title":"邮箱异常","description":"邮箱登录失败","category":"mail","priority":"medium"}`)

	request := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/tickets/"+createdTicket.ID+"/comments",
		bytes.NewBufferString(`{"type":"internal_note","content":"排查 AD 同步任务。"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+issueToken(t, tokenManager, endUser))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusForbidden {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusForbidden, recorder.Code, recorder.Body.String())
	}
}

func TestTicketTimelineForAgentReturnsAuditAndComment(t *testing.T) {
	handler, tokenManager := newTestTicketMux(t)
	endUser := identity.IdentityContext{
		UserID:         "user-end-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}
	agent := identity.IdentityContext{
		UserID:         "user-agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}

	createdTicket := createTicketViaHTTP(t, handler, tokenManager, endUser, `{"title":"共享盘无权限","description":"无法访问部门共享盘","category":"storage","priority":"high"}`)
	addCommentViaHTTP(t, handler, tokenManager, agent, createdTicket.ID, `{"type":"comment","content":"已联系域控管理员协助排查。"}`, stdhttp.StatusCreated)

	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/tickets/"+createdTicket.ID+"/timeline", nil)
	request.Header.Set("Authorization", "Bearer "+issueToken(t, tokenManager, agent))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response ticketTimelineResponsePayload
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode timeline response: %v", err)
	}

	if len(response.Items) < 2 {
		t.Fatalf("expected timeline to include audit and comment items, got %d", len(response.Items))
	}

	var hasCreatedAudit bool
	var hasComment bool
	for _, item := range response.Items {
		if item.ItemType == ticket.TicketTimelineItemTypeAuditEvent && item.AuditEventType == ticket.TicketAuditEventTypeCreated {
			hasCreatedAudit = true
		}
		if item.ItemType == ticket.TicketTimelineItemTypeComment && item.CommentType == ticket.TicketCommentTypeComment {
			hasComment = true
		}
	}

	if !hasCreatedAudit {
		t.Fatalf("expected created audit event in timeline")
	}
	if !hasComment {
		t.Fatalf("expected public comment item in timeline")
	}
}

func TestTicketTimelineHidesInternalNotesFromEndUser(t *testing.T) {
	handler, tokenManager := newTestTicketMux(t)
	endUser := identity.IdentityContext{
		UserID:         "user-end-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}
	agent := identity.IdentityContext{
		UserID:         "user-agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}

	createdTicket := createTicketViaHTTP(t, handler, tokenManager, endUser, `{"title":"账号锁定","description":"登录多次失败后被锁定","category":"account","priority":"urgent"}`)
	addCommentViaHTTP(t, handler, tokenManager, agent, createdTicket.ID, `{"type":"comment","content":"已重置账号状态，请稍后重试。"}`, stdhttp.StatusCreated)
	addCommentViaHTTP(t, handler, tokenManager, agent, createdTicket.ID, `{"type":"internal_note","content":"怀疑触发了异常登录告警。"}`, stdhttp.StatusCreated)

	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/tickets/"+createdTicket.ID+"/timeline", nil)
	request.Header.Set("Authorization", "Bearer "+issueToken(t, tokenManager, endUser))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response ticketTimelineResponsePayload
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode timeline response: %v", err)
	}

	for _, item := range response.Items {
		if item.CommentType == ticket.TicketCommentTypeInternalNote {
			t.Fatalf("expected end user timeline to hide internal notes")
		}
	}
}

func TestTicketCollaborationRoutesRequireAuthentication(t *testing.T) {
	handler, tokenManager := newTestTicketMux(t)
	endUser := identity.IdentityContext{
		UserID:         "user-end-1",
		OrganizationID: "org-1",
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	}

	createdTicket := createTicketViaHTTP(t, handler, tokenManager, endUser, `{"title":"打印机异常","description":"打印机无法打印","category":"device","priority":"low"}`)

	commentRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/tickets/"+createdTicket.ID+"/comments",
		bytes.NewBufferString(`{"type":"comment","content":"继续补充日志。"}`),
	)
	commentRequest.Header.Set("Content-Type", "application/json")
	commentRecorder := httptest.NewRecorder()
	handler.ServeHTTP(commentRecorder, commentRequest)
	if commentRecorder.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("expected comment route status %d, got %d, body=%s", stdhttp.StatusUnauthorized, commentRecorder.Code, commentRecorder.Body.String())
	}

	timelineRequest := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/tickets/"+createdTicket.ID+"/timeline", nil)
	timelineRecorder := httptest.NewRecorder()
	handler.ServeHTTP(timelineRecorder, timelineRequest)
	if timelineRecorder.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("expected timeline route status %d, got %d, body=%s", stdhttp.StatusUnauthorized, timelineRecorder.Code, timelineRecorder.Body.String())
	}
}

func TestTicketRequiresAuthentication(t *testing.T) {
	handler, _ := newTestTicketMux(t)
	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/tickets", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
}

func newTestTicketMux(t *testing.T) (*stdhttp.ServeMux, identity.TokenManager) {
	t.Helper()

	ticketService := ticket.NewService(
		ticket.NewMemoryTicketRepository(),
		ticket.WithCollaborationDependencies(ticket.CollaborationDependencies{
			CommentRepository: ticket.NewMemoryTicketCommentRepository(),
			AuditRepository:   ticket.NewMemoryTicketAuditEventRepository(),
		}),
	)
	tokenManager := identity.NewTokenManager("test-signing-key", time.Hour)

	return NewMuxWithRouteDependencies(config.Config{
		AppName:  "zld-supportpilot-test",
		AppEnv:   "test",
		HTTPAddr: ":0",
	}, RouteDependencies{
		Auth: &AuthDependencies{
			TokenManager: tokenManager,
		},
		Ticket: &TicketDependencies{
			TicketService: ticketService,
		},
	}), tokenManager
}

func createTicketViaHTTP(t *testing.T, handler stdhttp.Handler, tokenManager identity.TokenManager, actor identity.IdentityContext, body string) ticketResponsePayload {
	t.Helper()

	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/tickets", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+issueToken(t, tokenManager, actor))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)
	if recorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected create ticket status %d, got %d, body=%s", stdhttp.StatusCreated, recorder.Code, recorder.Body.String())
	}

	var response ticketResponsePayload
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode create ticket response: %v", err)
	}

	return response
}

func addCommentViaHTTP(t *testing.T, handler stdhttp.Handler, tokenManager identity.TokenManager, actor identity.IdentityContext, ticketID string, body string, expectedStatus int) ticketCommentResponsePayload {
	t.Helper()

	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/tickets/"+ticketID+"/comments", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+issueToken(t, tokenManager, actor))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)
	if recorder.Code != expectedStatus {
		t.Fatalf("expected add comment status %d, got %d, body=%s", expectedStatus, recorder.Code, recorder.Body.String())
	}

	var response ticketCommentResponsePayload
	if expectedStatus == stdhttp.StatusCreated {
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("decode add comment response: %v", err)
		}
	}

	return response
}

func issueToken(t *testing.T, tokenManager identity.TokenManager, actor identity.IdentityContext) string {
	t.Helper()

	accessToken, err := tokenManager.Issue(actor)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	return accessToken
}
