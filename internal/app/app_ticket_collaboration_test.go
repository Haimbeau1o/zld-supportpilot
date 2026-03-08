package app

import (
	"bytes"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ticket"
)

type appTicketResponse struct {
	ID string `json:"id"`
}

type appTicketCommentResponse struct {
	ID       string                   `json:"id"`
	TicketID string                   `json:"ticket_id"`
	Type     ticket.TicketCommentType `json:"type"`
	Content  string                   `json:"content"`
}

type appTicketTimelineItemResponse struct {
	ItemType       ticket.TicketTimelineItemType `json:"item_type"`
	CommentType    ticket.TicketCommentType      `json:"comment_type"`
	AuditEventType ticket.TicketAuditEventType   `json:"audit_event_type"`
	Content        string                        `json:"content"`
}

type appTicketTimelineResponse struct {
	Items []appTicketTimelineItemResponse `json:"items"`
}

func TestNewWiresTicketCollaborationRoutes(t *testing.T) {
	t.Setenv("AUTH_SIGNING_KEY", "test-signing-key")
	t.Setenv("AUTH_TOKEN_TTL_SECONDS", "3600")

	application, accessToken := bootstrapLoggedInApplication(t)
	defer application.Close()

	createTicketRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/tickets",
		bytes.NewBufferString(`{"title":"VPN 无法连接","description":"今天上午开始无法连接公司 VPN","category":"network","priority":"high"}`),
	)
	createTicketRequest.Header.Set("Content-Type", "application/json")
	createTicketRequest.Header.Set("Authorization", "Bearer "+accessToken)
	createTicketRecorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(createTicketRecorder, createTicketRequest)

	if createTicketRecorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected create ticket status %d, got %d, body=%s", stdhttp.StatusCreated, createTicketRecorder.Code, createTicketRecorder.Body.String())
	}

	var createTicketResponse appTicketResponse
	if err := json.NewDecoder(createTicketRecorder.Body).Decode(&createTicketResponse); err != nil {
		t.Fatalf("decode create ticket response: %v", err)
	}

	commentRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/tickets/"+createTicketResponse.ID+"/comments",
		bytes.NewBufferString(`{"type":"comment","content":"补充了故障出现时间与截图。"}`),
	)
	commentRequest.Header.Set("Content-Type", "application/json")
	commentRequest.Header.Set("Authorization", "Bearer "+accessToken)
	commentRecorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(commentRecorder, commentRequest)

	if commentRecorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected add comment status %d, got %d, body=%s", stdhttp.StatusCreated, commentRecorder.Code, commentRecorder.Body.String())
	}

	var commentResponse appTicketCommentResponse
	if err := json.NewDecoder(commentRecorder.Body).Decode(&commentResponse); err != nil {
		t.Fatalf("decode add comment response: %v", err)
	}

	if commentResponse.TicketID != createTicketResponse.ID {
		t.Fatalf("expected comment ticket id %q, got %q", createTicketResponse.ID, commentResponse.TicketID)
	}
	if commentResponse.Type != ticket.TicketCommentTypeComment {
		t.Fatalf("expected comment type %q, got %q", ticket.TicketCommentTypeComment, commentResponse.Type)
	}

	timelineRequest := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/tickets/"+createTicketResponse.ID+"/timeline", nil)
	timelineRequest.Header.Set("Authorization", "Bearer "+accessToken)
	timelineRecorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(timelineRecorder, timelineRequest)

	if timelineRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected timeline status %d, got %d, body=%s", stdhttp.StatusOK, timelineRecorder.Code, timelineRecorder.Body.String())
	}

	var timelineResponse appTicketTimelineResponse
	if err := json.NewDecoder(timelineRecorder.Body).Decode(&timelineResponse); err != nil {
		t.Fatalf("decode timeline response: %v", err)
	}

	if len(timelineResponse.Items) < 2 {
		t.Fatalf("expected timeline to include created audit and comment, got %d", len(timelineResponse.Items))
	}

	var hasCreatedAudit bool
	var hasComment bool
	for _, item := range timelineResponse.Items {
		if item.ItemType == ticket.TicketTimelineItemTypeAuditEvent && item.AuditEventType == ticket.TicketAuditEventTypeCreated {
			hasCreatedAudit = true
		}
		if item.ItemType == ticket.TicketTimelineItemTypeComment && item.CommentType == ticket.TicketCommentTypeComment {
			hasComment = true
		}
	}

	if !hasCreatedAudit {
		t.Fatalf("expected created audit event in app timeline")
	}
	if !hasComment {
		t.Fatalf("expected comment item in app timeline")
	}
}
