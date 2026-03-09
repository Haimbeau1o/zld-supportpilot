package app

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
)

type appMainLoopMeResponse struct {
	OrganizationID string `json:"organization_id"`
}

type appMainLoopIntakeResponse struct {
	IntakeID     string `json:"intake_id"`
	ResultType   string `json:"result_type"`
	AnswerStatus string `json:"answer_status"`
	Answer       string `json:"answer"`
	TicketID     string `json:"ticket_id"`
	Citations    []struct {
		Content string `json:"content"`
	} `json:"citations"`
}

type appMainLoopCandidateResponse struct {
	ID                 string `json:"id"`
	Status             string `json:"status"`
	ApprovedDocumentID string `json:"approved_document_id"`
}

func TestNewMainLoopFusionSupportsEndUserFeedbackToKnowledgeCycle(t *testing.T) {
	t.Setenv("AUTH_SIGNING_KEY", "test-signing-key")
	t.Setenv("AUTH_TOKEN_TTL_SECONDS", "3600")

	application, adminAccessToken := bootstrapLoggedInApplication(t)
	defer application.Close()

	organizationID := getOrganizationIDViaHTTP(t, application, adminAccessToken)
	agentToken := issueAppToken(t, identity.IdentityContext{
		UserID:         "user-agent-1",
		OrganizationID: organizationID,
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	})
	endUserToken := issueAppToken(t, identity.IdentityContext{
		UserID:         "user-end-1",
		OrganizationID: organizationID,
		Role:           identity.RoleEndUser,
		Permissions:    identity.RolePermissions(identity.RoleEndUser),
	})

	knowledgeBase := createKnowledgeBaseInApp(t, application, agentToken, `{"name":"IT 支持知识库","description":"用于沉淀 IT 文档"}`)
	uploadedDocumentID := uploadTextDocumentInApp(t, application, agentToken, knowledgeBase.ID, "vpn-general-guide.txt", "VPN 691 错误通常需要由 IT 服务台人工处理，请先提交工单，稍后由客服继续排查。")
	waitForDocumentReady(t, application, agentToken, uploadedDocumentID)

	firstIntakeRecorder := httptest.NewRecorder()
	firstIntakeRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/ai/intakes",
		bytes.NewBufferString(`{"knowledge_base_id":"`+knowledgeBase.ID+`","question":"VPN 691 错误怎么处理？","top_k":3}`),
	)
	firstIntakeRequest.Header.Set("Content-Type", "application/json")
	firstIntakeRequest.Header.Set("Authorization", "Bearer "+endUserToken)
	application.server.Handler.ServeHTTP(firstIntakeRecorder, firstIntakeRequest)
	if firstIntakeRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected end user intake status %d, got %d, body=%s", stdhttp.StatusOK, firstIntakeRecorder.Code, firstIntakeRecorder.Body.String())
	}

	var firstIntakeResponse appMainLoopIntakeResponse
	if err := json.NewDecoder(firstIntakeRecorder.Body).Decode(&firstIntakeResponse); err != nil {
		t.Fatalf("decode first intake response: %v", err)
	}
	if firstIntakeResponse.ResultType != "answered" {
		t.Fatalf("expected first intake result type answered, got %q", firstIntakeResponse.ResultType)
	}
	if firstIntakeResponse.IntakeID == "" {
		t.Fatalf("expected intake id to be returned")
	}
	if strings.TrimSpace(firstIntakeResponse.Answer) == "" {
		t.Fatalf("expected first intake answer to be returned")
	}

	feedbackRecorder := httptest.NewRecorder()
	feedbackRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/ai/intakes/"+firstIntakeResponse.IntakeID+"/feedback",
		bytes.NewBufferString(`{"status":"unresolved","comment":"按建议操作后仍失败，需要人工继续处理。"}`),
	)
	feedbackRequest.Header.Set("Content-Type", "application/json")
	feedbackRequest.Header.Set("Authorization", "Bearer "+endUserToken)
	application.server.Handler.ServeHTTP(feedbackRecorder, feedbackRequest)
	if feedbackRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected feedback status %d, got %d, body=%s", stdhttp.StatusOK, feedbackRecorder.Code, feedbackRecorder.Body.String())
	}

	var feedbackResponse appAnswerFeedbackResponse
	if err := json.NewDecoder(feedbackRecorder.Body).Decode(&feedbackResponse); err != nil {
		t.Fatalf("decode feedback response: %v", err)
	}
	if feedbackResponse.TicketAction != "ticket_created" {
		t.Fatalf("expected feedback ticket action ticket_created, got %q", feedbackResponse.TicketAction)
	}
	if feedbackResponse.TicketID == "" {
		t.Fatalf("expected feedback to create ticket")
	}

	assistRecorder := httptest.NewRecorder()
	assistRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/ai/tickets/"+feedbackResponse.TicketID+"/assist",
		bytes.NewBufferString(`{"knowledge_base_id":"`+knowledgeBase.ID+`"}`),
	)
	assistRequest.Header.Set("Content-Type", "application/json")
	assistRequest.Header.Set("Authorization", "Bearer "+agentToken)
	application.server.Handler.ServeHTTP(assistRecorder, assistRequest)
	if assistRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected ticket assist status %d, got %d, body=%s", stdhttp.StatusOK, assistRecorder.Code, assistRecorder.Body.String())
	}

	var assistResponse appAITicketAssistResponse
	if err := json.NewDecoder(assistRecorder.Body).Decode(&assistResponse); err != nil {
		t.Fatalf("decode ticket assist response: %v", err)
	}
	if assistResponse.RecordedCommentID == "" {
		t.Fatalf("expected ticket assist to record internal note")
	}

	transitionTicketStatusInApp(t, application, agentToken, feedbackResponse.TicketID, "in_progress")
	transitionTicketStatusInApp(t, application, agentToken, feedbackResponse.TicketID, "resolved")

	candidate := createKnowledgeCandidateInApp(t, application, agentToken, `{"ticket_id":"`+feedbackResponse.TicketID+`","knowledge_base_id":"`+knowledgeBase.ID+`","title":"VPN 691 错误排查","summary":"通过重置账号拨号权限恢复","content":"处理步骤：检查账号状态，重置账号拨号权限，重新连接验证。"}`)

	reviewRecorder := httptest.NewRecorder()
	reviewRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/knowledge/candidates/"+candidate.ID+"/review",
		bytes.NewBufferString(`{"action":"approve","comment":"内容可入库"}`),
	)
	reviewRequest.Header.Set("Content-Type", "application/json")
	reviewRequest.Header.Set("Authorization", "Bearer "+agentToken)
	application.server.Handler.ServeHTTP(reviewRecorder, reviewRequest)
	if reviewRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected candidate review status %d, got %d, body=%s", stdhttp.StatusOK, reviewRecorder.Code, reviewRecorder.Body.String())
	}

	if err := json.NewDecoder(reviewRecorder.Body).Decode(&candidate); err != nil {
		t.Fatalf("decode candidate review response: %v", err)
	}
	if candidate.ApprovedDocumentID == "" {
		t.Fatalf("expected approved document id to be returned")
	}
	waitForDocumentReady(t, application, agentToken, candidate.ApprovedDocumentID)

	secondIntakeRecorder := httptest.NewRecorder()
	secondIntakeRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/ai/intakes",
		bytes.NewBufferString(`{"knowledge_base_id":"`+knowledgeBase.ID+`","question":"VPN 691 错误怎么处理？","top_k":3}`),
	)
	secondIntakeRequest.Header.Set("Content-Type", "application/json")
	secondIntakeRequest.Header.Set("Authorization", "Bearer "+endUserToken)
	application.server.Handler.ServeHTTP(secondIntakeRecorder, secondIntakeRequest)
	if secondIntakeRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected second intake status %d, got %d, body=%s", stdhttp.StatusOK, secondIntakeRecorder.Code, secondIntakeRecorder.Body.String())
	}

	var secondIntakeResponse appMainLoopIntakeResponse
	if err := json.NewDecoder(secondIntakeRecorder.Body).Decode(&secondIntakeResponse); err != nil {
		t.Fatalf("decode second intake response: %v", err)
	}
	if secondIntakeResponse.ResultType != "answered" {
		t.Fatalf("expected second intake result type answered, got %q", secondIntakeResponse.ResultType)
	}

	matched := strings.Contains(secondIntakeResponse.Answer, "重置账号拨号权限")
	if !matched {
		for _, citation := range secondIntakeResponse.Citations {
			if strings.Contains(citation.Content, "重置账号拨号权限") {
				matched = true
				break
			}
		}
	}
	if !matched {
		t.Fatalf("expected second intake answer or citations to contain approved candidate content, got answer=%q citations=%+v", secondIntakeResponse.Answer, secondIntakeResponse.Citations)
	}
}

func getOrganizationIDViaHTTP(t *testing.T, application *App, accessToken string) string {
	t.Helper()

	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/auth/me", nil)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	recorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected me status %d, got %d, body=%s", stdhttp.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response appMainLoopMeResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode me response: %v", err)
	}
	if response.OrganizationID == "" {
		t.Fatalf("expected organization id to be returned")
	}
	return response.OrganizationID
}

func issueAppToken(t *testing.T, actor identity.IdentityContext) string {
	t.Helper()

	tokenManager := identity.NewTokenManager("test-signing-key", time.Hour)
	accessToken, err := tokenManager.Issue(actor)
	if err != nil {
		t.Fatalf("issue app token: %v", err)
	}
	return accessToken
}

func createKnowledgeBaseInApp(t *testing.T, application *App, accessToken string, body string) appKnowledgeBaseResponse {
	t.Helper()

	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/knowledge/bases", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+accessToken)
	recorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected create knowledge base status %d, got %d, body=%s", stdhttp.StatusCreated, recorder.Code, recorder.Body.String())
	}

	var response appKnowledgeBaseResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode create knowledge base response: %v", err)
	}
	return response
}

func uploadTextDocumentInApp(t *testing.T, application *App, accessToken string, knowledgeBaseID string, filename string, content string) string {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fileWriter, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fileWriter.Write([]byte(content)); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/knowledge/bases/"+knowledgeBaseID+"/documents", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+accessToken)
	recorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected upload document status %d, got %d, body=%s", stdhttp.StatusCreated, recorder.Code, recorder.Body.String())
	}

	var response struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode upload document response: %v", err)
	}
	if response.ID == "" {
		t.Fatalf("expected uploaded document id to be returned")
	}
	return response.ID
}

func transitionTicketStatusInApp(t *testing.T, application *App, accessToken string, ticketID string, status string) {
	t.Helper()

	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/tickets/"+ticketID+"/status", bytes.NewBufferString(`{"status":"`+status+`"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+accessToken)
	recorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected transition ticket status %d, got %d, body=%s", stdhttp.StatusOK, recorder.Code, recorder.Body.String())
	}
}

func createKnowledgeCandidateInApp(t *testing.T, application *App, accessToken string, body string) appMainLoopCandidateResponse {
	t.Helper()

	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/knowledge/candidates", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+accessToken)
	recorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected create knowledge candidate status %d, got %d, body=%s", stdhttp.StatusCreated, recorder.Code, recorder.Body.String())
	}

	var response appMainLoopCandidateResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode create knowledge candidate response: %v", err)
	}
	return response
}
