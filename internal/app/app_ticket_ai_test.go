package app

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ticket"
)

type appAITicketAssistResponse struct {
	CategorySuggestion string `json:"category_suggestion"`
	Summary            string `json:"summary"`
	ReplyDraft         string `json:"reply_draft"`
	RecordedCommentID  string `json:"recorded_comment_id"`
}

type appAITicketTimelineResponse struct {
	Items []struct {
		ItemType    ticket.TicketTimelineItemType `json:"item_type"`
		CommentType ticket.TicketCommentType      `json:"comment_type"`
		Content     string                        `json:"content"`
	} `json:"items"`
}

func TestNewWiresAITicketAssistRoute(t *testing.T) {
	t.Setenv("AUTH_SIGNING_KEY", "test-signing-key")
	t.Setenv("AUTH_TOKEN_TTL_SECONDS", "3600")

	application, accessToken := bootstrapLoggedInApplication(t)
	defer application.Close()

	createKnowledgeBaseRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/knowledge/bases",
		bytes.NewBufferString(`{"name":"IT 支持知识库","description":"用于沉淀 IT 文档"}`),
	)
	createKnowledgeBaseRequest.Header.Set("Content-Type", "application/json")
	createKnowledgeBaseRequest.Header.Set("Authorization", "Bearer "+accessToken)
	createKnowledgeBaseRecorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(createKnowledgeBaseRecorder, createKnowledgeBaseRequest)
	if createKnowledgeBaseRecorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected knowledge base status %d, got %d, body=%s", stdhttp.StatusCreated, createKnowledgeBaseRecorder.Code, createKnowledgeBaseRecorder.Body.String())
	}

	var knowledgeBaseResponse appKnowledgeBaseResponse
	if err := json.NewDecoder(createKnowledgeBaseRecorder.Body).Decode(&knowledgeBaseResponse); err != nil {
		t.Fatalf("decode knowledge base response: %v", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fileWriter, err := writer.CreateFormFile("file", "vpn-guide.txt")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fileWriter.Write([]byte("VPN 无法连接时，请先重置客户端，再重新登录。如果仍失败，请联系网络管理员。")); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	uploadRequest := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/knowledge/bases/"+knowledgeBaseResponse.ID+"/documents", body)
	uploadRequest.Header.Set("Content-Type", writer.FormDataContentType())
	uploadRequest.Header.Set("Authorization", "Bearer "+accessToken)
	uploadRecorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(uploadRecorder, uploadRequest)
	if uploadRecorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected upload status %d, got %d, body=%s", stdhttp.StatusCreated, uploadRecorder.Code, uploadRecorder.Body.String())
	}

	var uploadResponse struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(uploadRecorder.Body).Decode(&uploadResponse); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	var documentReady bool
	for time.Now().Before(deadline) {
		detailRequest := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/knowledge/documents/"+uploadResponse.ID, nil)
		detailRequest.Header.Set("Authorization", "Bearer "+accessToken)
		detailRecorder := httptest.NewRecorder()
		application.server.Handler.ServeHTTP(detailRecorder, detailRequest)
		if detailRecorder.Code != stdhttp.StatusOK {
			t.Fatalf("expected detail status %d, got %d, body=%s", stdhttp.StatusOK, detailRecorder.Code, detailRecorder.Body.String())
		}

		var detailResponse appDocumentResponse
		if err := json.NewDecoder(detailRecorder.Body).Decode(&detailResponse); err != nil {
			t.Fatalf("decode detail response: %v", err)
		}
		if detailResponse.Status == "ready" {
			documentReady = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !documentReady {
		t.Fatalf("expected uploaded document to become ready before triggering ai assist")
	}

	createTicketRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/tickets",
		bytes.NewBufferString(`{"title":"VPN 无法连接","description":"今天上午开始无法连接公司 VPN，重试后仍失败。","category":"network","priority":"high"}`),
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

	var assistRecorder *httptest.ResponseRecorder
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		assistRequest := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/ai/tickets/"+createTicketResponse.ID+"/assist", bytes.NewBufferString(`{"knowledge_base_id":"`+knowledgeBaseResponse.ID+`"}`))
		assistRequest.Header.Set("Content-Type", "application/json")
		assistRequest.Header.Set("Authorization", "Bearer "+accessToken)
		assistRecorder = httptest.NewRecorder()
		application.server.Handler.ServeHTTP(assistRecorder, assistRequest)
		if assistRecorder.Code == stdhttp.StatusOK {
			break
		}
		if assistRecorder.Code != stdhttp.StatusServiceUnavailable {
			t.Fatalf("expected assist status %d or %d during warmup, got %d, body=%s", stdhttp.StatusOK, stdhttp.StatusServiceUnavailable, assistRecorder.Code, assistRecorder.Body.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
	if assistRecorder == nil || assistRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected assist route to become ready, last code=%d body=%s", assistRecorder.Code, assistRecorder.Body.String())
	}

	var assistResponse appAITicketAssistResponse
	if err := json.NewDecoder(assistRecorder.Body).Decode(&assistResponse); err != nil {
		t.Fatalf("decode assist response: %v", err)
	}
	if assistResponse.CategorySuggestion == "" || assistResponse.Summary == "" || assistResponse.ReplyDraft == "" || assistResponse.RecordedCommentID == "" {
		t.Fatalf("expected structured assist response, got %+v", assistResponse)
	}

	timelineRequest := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/tickets/"+createTicketResponse.ID+"/timeline", nil)
	timelineRequest.Header.Set("Authorization", "Bearer "+accessToken)
	timelineRecorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(timelineRecorder, timelineRequest)
	if timelineRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected timeline status %d, got %d, body=%s", stdhttp.StatusOK, timelineRecorder.Code, timelineRecorder.Body.String())
	}

	var timelineResponse appAITicketTimelineResponse
	if err := json.NewDecoder(timelineRecorder.Body).Decode(&timelineResponse); err != nil {
		t.Fatalf("decode timeline response: %v", err)
	}

	var hasAIRecord bool
	for _, item := range timelineResponse.Items {
		if item.CommentType == ticket.TicketCommentTypeInternalNote && bytes.Contains([]byte(item.Content), []byte("【AI 执行记录】")) {
			hasAIRecord = true
		}
	}
	if !hasAIRecord {
		t.Fatalf("expected ai execution record in timeline")
	}
}
