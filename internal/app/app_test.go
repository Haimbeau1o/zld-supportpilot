package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime/multipart"
	stdhttp "net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/persistence"
)

type appLoginResponse struct {
	AccessToken string `json:"access_token"`
}

type appKnowledgeBaseResponse struct {
	ID string `json:"id"`
}

type appUnifiedIntakeResponse struct {
	IntakeID   string `json:"intake_id"`
	ResultType string `json:"result_type"`
	TicketID   string `json:"ticket_id"`
}

type appAnswerFeedbackResponse struct {
	FeedbackID   string `json:"feedback_id"`
	IntakeID     string `json:"intake_id"`
	Status       string `json:"status"`
	TicketAction string `json:"ticket_action"`
	TicketID     string `json:"ticket_id"`
}

type appAnswerFeedbackStatsResponse struct {
	Total      int `json:"total"`
	Resolved   int `json:"resolved"`
	Unresolved int `json:"unresolved"`
	Inaccurate int `json:"inaccurate"`
}

func TestNewWiresAuthRoutes(t *testing.T) {
	t.Setenv("AUTH_SIGNING_KEY", "test-signing-key")
	t.Setenv("AUTH_TOKEN_TTL_SECONDS", "3600")

	application, err := New()
	if err != nil {
		t.Fatalf("expected app bootstrap success, got error: %v", err)
	}
	defer application.Close()

	request := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/auth/register",
		bytes.NewBufferString(`{"name":"Alice","email":"alice@example.com","password":"secret123","tenant_name":"中联数据智能服务台"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	application.server.Handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusCreated, recorder.Code, recorder.Body.String())
	}
}

func TestNewBootstrapsPostgresModeWithDependencies(t *testing.T) {
	t.Setenv("APP_PERSISTENCE_MODE", "postgres")
	t.Setenv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/supportpilot?sslmode=disable")
	t.Setenv("DOCUMENT_STORAGE_MODE", "filesystem")
	t.Setenv("DOCUMENT_STORAGE_ROOT", filepath.Join(t.TempDir(), "documents"))
	t.Setenv("AUTH_SIGNING_KEY", "test-signing-key")
	t.Setenv("AUTH_TOKEN_TTL_SECONDS", "3600")

	originalOpenPostgres := openPostgres
	t.Cleanup(func() { openPostgres = originalOpenPostgres })
	openPostgres = func(ctx context.Context, dsn string, options ...persistence.PostgresOption) (*sql.DB, error) {
		db, _, err := sqlmock.New()
		if err != nil {
			return nil, err
		}
		return db, nil
	}

	application, err := New()
	if err != nil {
		t.Fatalf("expected postgres bootstrap success, got error: %v", err)
	}
	defer application.Close()
}

func TestNewReturnsErrorWhenPostgresModeMissingDSN(t *testing.T) {
	t.Setenv("APP_PERSISTENCE_MODE", "postgres")
	t.Setenv("POSTGRES_DSN", "")
	t.Setenv("AUTH_SIGNING_KEY", "test-signing-key")
	t.Setenv("AUTH_TOKEN_TTL_SECONDS", "3600")

	application, err := New()
	if err == nil {
		if application != nil {
			_ = application.Close()
		}
		t.Fatalf("expected postgres mode without dsn to return error")
	}
}

func TestNewWiresTicketRoutes(t *testing.T) {
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
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusCreated, createTicketRecorder.Code, createTicketRecorder.Body.String())
	}
}

func TestNewWiresKnowledgeRoutes(t *testing.T) {
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
	fileWriter, err := writer.CreateFormFile("file", "vpn-guide.pdf")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fileWriter.Write([]byte("hello knowledge")); err != nil {
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
}

func TestNewKnowledgeCandidateApprovalFeedsAIAnswer(t *testing.T) {
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

	createTicketRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/tickets",
		bytes.NewBufferString(`{"title":"VPN 无法连接","description":"错误码 691","category":"network","priority":"high"}`),
	)
	createTicketRequest.Header.Set("Content-Type", "application/json")
	createTicketRequest.Header.Set("Authorization", "Bearer "+accessToken)
	createTicketRecorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(createTicketRecorder, createTicketRequest)
	if createTicketRecorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected ticket create status %d, got %d, body=%s", stdhttp.StatusCreated, createTicketRecorder.Code, createTicketRecorder.Body.String())
	}

	var ticketResponse struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(createTicketRecorder.Body).Decode(&ticketResponse); err != nil {
		t.Fatalf("decode ticket response: %v", err)
	}

	for _, status := range []string{"in_progress", "resolved"} {
		transitionRequest := httptest.NewRequest(
			stdhttp.MethodPost,
			"/api/v1/tickets/"+ticketResponse.ID+"/status",
			bytes.NewBufferString(`{"status":"`+status+`"}`),
		)
		transitionRequest.Header.Set("Content-Type", "application/json")
		transitionRequest.Header.Set("Authorization", "Bearer "+accessToken)
		transitionRecorder := httptest.NewRecorder()
		application.server.Handler.ServeHTTP(transitionRecorder, transitionRequest)
		if transitionRecorder.Code != stdhttp.StatusOK {
			t.Fatalf("expected transition status %d, got %d, body=%s", stdhttp.StatusOK, transitionRecorder.Code, transitionRecorder.Body.String())
		}
	}

	createCandidateRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/knowledge/candidates",
		bytes.NewBufferString(fmt.Sprintf(`{"ticket_id":"%s","knowledge_base_id":"%s","title":"VPN 691 错误排查","summary":"通过重置账号拨号权限恢复","content":"处理步骤：检查账号状态，重置账号拨号权限，重新连接验证。"}`, ticketResponse.ID, knowledgeBaseResponse.ID)),
	)
	createCandidateRequest.Header.Set("Content-Type", "application/json")
	createCandidateRequest.Header.Set("Authorization", "Bearer "+accessToken)
	createCandidateRecorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(createCandidateRecorder, createCandidateRequest)
	if createCandidateRecorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected candidate create status %d, got %d, body=%s", stdhttp.StatusCreated, createCandidateRecorder.Code, createCandidateRecorder.Body.String())
	}

	var candidateResponse struct {
		ID                 string `json:"id"`
		ApprovedDocumentID string `json:"approved_document_id"`
		Status             string `json:"status"`
	}
	if err := json.NewDecoder(createCandidateRecorder.Body).Decode(&candidateResponse); err != nil {
		t.Fatalf("decode candidate response: %v", err)
	}

	reviewRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/knowledge/candidates/"+candidateResponse.ID+"/review",
		bytes.NewBufferString(`{"action":"approve","comment":"内容可入库"}`),
	)
	reviewRequest.Header.Set("Content-Type", "application/json")
	reviewRequest.Header.Set("Authorization", "Bearer "+accessToken)
	reviewRecorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(reviewRecorder, reviewRequest)
	if reviewRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected candidate review status %d, got %d, body=%s", stdhttp.StatusOK, reviewRecorder.Code, reviewRecorder.Body.String())
	}
	if err := json.NewDecoder(reviewRecorder.Body).Decode(&candidateResponse); err != nil {
		t.Fatalf("decode reviewed candidate response: %v", err)
	}
	if candidateResponse.ApprovedDocumentID == "" {
		t.Fatalf("expected approved document id to be returned")
	}

	waitForDocumentReady(t, application, accessToken, candidateResponse.ApprovedDocumentID)

	answerRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/ai/knowledge/bases/"+knowledgeBaseResponse.ID+"/answers",
		bytes.NewBufferString(`{"question":"VPN 691 错误怎么处理？","top_k":3}`),
	)
	answerRequest.Header.Set("Content-Type", "application/json")
	answerRequest.Header.Set("Authorization", "Bearer "+accessToken)
	answerRecorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(answerRecorder, answerRequest)
	if answerRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected answer status %d, got %d, body=%s", stdhttp.StatusOK, answerRecorder.Code, answerRecorder.Body.String())
	}

	var answerResponse struct {
		Status    string `json:"status"`
		Answer    string `json:"answer"`
		Citations []struct {
			Content string `json:"content"`
		} `json:"citations"`
	}
	if err := json.NewDecoder(answerRecorder.Body).Decode(&answerResponse); err != nil {
		t.Fatalf("decode answer response: %v", err)
	}
	if answerResponse.Status != "answered" {
		t.Fatalf("expected answer status answered, got %q", answerResponse.Status)
	}

	matched := strings.Contains(answerResponse.Answer, "重置账号拨号权限")
	if !matched {
		for _, citation := range answerResponse.Citations {
			if strings.Contains(citation.Content, "重置账号拨号权限") {
				matched = true
				break
			}
		}
	}
	if !matched {
		t.Fatalf("expected answer or citations to contain approved candidate content, got answer=%q citations=%+v", answerResponse.Answer, answerResponse.Citations)
	}
}

func TestNewWiresAIIntakeRoute(t *testing.T) {
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

	intakeRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/ai/intakes",
		bytes.NewBufferString(fmt.Sprintf(`{"knowledge_base_id":"%s","question":"VPN 无法连接怎么办？","top_k":2}`, knowledgeBaseResponse.ID)),
	)
	intakeRequest.Header.Set("Content-Type", "application/json")
	intakeRequest.Header.Set("Authorization", "Bearer "+accessToken)
	intakeRecorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(intakeRecorder, intakeRequest)

	if intakeRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected intake status %d, got %d, body=%s", stdhttp.StatusOK, intakeRecorder.Code, intakeRecorder.Body.String())
	}

	var response appUnifiedIntakeResponse
	if err := json.NewDecoder(intakeRecorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode intake response: %v", err)
	}
	if response.ResultType != "ticket_created" {
		t.Fatalf("expected result type ticket_created, got %q", response.ResultType)
	}
	if response.TicketID == "" {
		t.Fatalf("expected ticket id to be returned")
	}
	if response.IntakeID == "" {
		t.Fatalf("expected intake id to be returned")
	}
}

func TestNewWiresAIAnswerFeedbackRoute(t *testing.T) {
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

	intakeRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/ai/intakes",
		bytes.NewBufferString(fmt.Sprintf(`{"knowledge_base_id":"%s","question":"VPN 无法连接怎么办？","top_k":2}`, knowledgeBaseResponse.ID)),
	)
	intakeRequest.Header.Set("Content-Type", "application/json")
	intakeRequest.Header.Set("Authorization", "Bearer "+accessToken)
	intakeRecorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(intakeRecorder, intakeRequest)
	if intakeRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected intake status %d, got %d, body=%s", stdhttp.StatusOK, intakeRecorder.Code, intakeRecorder.Body.String())
	}

	var intakeResponse appUnifiedIntakeResponse
	if err := json.NewDecoder(intakeRecorder.Body).Decode(&intakeResponse); err != nil {
		t.Fatalf("decode intake response: %v", err)
	}

	feedbackRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/ai/intakes/"+intakeResponse.IntakeID+"/feedback",
		bytes.NewBufferString(`{"status":"unresolved","comment":"仍未解决"}`),
	)
	feedbackRequest.Header.Set("Content-Type", "application/json")
	feedbackRequest.Header.Set("Authorization", "Bearer "+accessToken)
	feedbackRecorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(feedbackRecorder, feedbackRequest)

	if feedbackRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected feedback status %d, got %d, body=%s", stdhttp.StatusOK, feedbackRecorder.Code, feedbackRecorder.Body.String())
	}

	var response appAnswerFeedbackResponse
	if err := json.NewDecoder(feedbackRecorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode feedback response: %v", err)
	}
	if response.TicketAction != "ticket_existing" {
		t.Fatalf("expected ticket action ticket_existing, got %q", response.TicketAction)
	}
	if response.TicketID != intakeResponse.TicketID {
		t.Fatalf("expected ticket id %q, got %q", intakeResponse.TicketID, response.TicketID)
	}
}

func TestNewWiresAIAnswerFeedbackStatsRoute(t *testing.T) {
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

	intakeRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/ai/intakes",
		bytes.NewBufferString(fmt.Sprintf(`{"knowledge_base_id":"%s","question":"VPN 无法连接怎么办？","top_k":2}`, knowledgeBaseResponse.ID)),
	)
	intakeRequest.Header.Set("Content-Type", "application/json")
	intakeRequest.Header.Set("Authorization", "Bearer "+accessToken)
	intakeRecorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(intakeRecorder, intakeRequest)
	if intakeRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected intake status %d, got %d, body=%s", stdhttp.StatusOK, intakeRecorder.Code, intakeRecorder.Body.String())
	}

	var intakeResponse appUnifiedIntakeResponse
	if err := json.NewDecoder(intakeRecorder.Body).Decode(&intakeResponse); err != nil {
		t.Fatalf("decode intake response: %v", err)
	}

	feedbackRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/ai/intakes/"+intakeResponse.IntakeID+"/feedback",
		bytes.NewBufferString(`{"status":"unresolved"}`),
	)
	feedbackRequest.Header.Set("Content-Type", "application/json")
	feedbackRequest.Header.Set("Authorization", "Bearer "+accessToken)
	feedbackRecorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(feedbackRecorder, feedbackRequest)
	if feedbackRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected feedback status %d, got %d, body=%s", stdhttp.StatusOK, feedbackRecorder.Code, feedbackRecorder.Body.String())
	}

	statsRequest := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/ai/feedback/stats", nil)
	statsRequest.Header.Set("Authorization", "Bearer "+accessToken)
	statsRecorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(statsRecorder, statsRequest)

	if statsRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected stats status %d, got %d, body=%s", stdhttp.StatusOK, statsRecorder.Code, statsRecorder.Body.String())
	}

	var response appAnswerFeedbackStatsResponse
	if err := json.NewDecoder(statsRecorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode stats response: %v", err)
	}
	if response.Total != 1 || response.Unresolved != 1 {
		t.Fatalf("unexpected stats response: %+v", response)
	}
}

func waitForDocumentReady(t *testing.T, application *App, accessToken string, documentID string) {
	t.Helper()

	for attempt := 0; attempt < 40; attempt++ {
		request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/knowledge/documents/"+documentID, nil)
		request.Header.Set("Authorization", "Bearer "+accessToken)
		recorder := httptest.NewRecorder()
		application.server.Handler.ServeHTTP(recorder, request)
		if recorder.Code != stdhttp.StatusOK {
			t.Fatalf("expected document detail status %d, got %d, body=%s", stdhttp.StatusOK, recorder.Code, recorder.Body.String())
		}

		var response struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("decode document detail response: %v", err)
		}
		if response.Status == "ready" {
			return
		}
		if response.Status == "failed" {
			t.Fatalf("expected document processing success, got failed")
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("document %s did not become ready in time", documentID)
}

func bootstrapLoggedInApplication(t *testing.T) (*App, string) {
	t.Helper()

	application, err := New()
	if err != nil {
		t.Fatalf("expected app bootstrap success, got error: %v", err)
	}
	t.Cleanup(func() { _ = application.Close() })

	registerRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/auth/register",
		bytes.NewBufferString(`{"name":"Alice","email":"alice@example.com","password":"secret123","tenant_name":"中联数据智能服务台"}`),
	)
	registerRequest.Header.Set("Content-Type", "application/json")
	registerRecorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(registerRecorder, registerRequest)
	if registerRecorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected register status %d, got %d, body=%s", stdhttp.StatusCreated, registerRecorder.Code, registerRecorder.Body.String())
	}

	loginRequest := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/auth/login",
		bytes.NewBufferString(`{"email":"alice@example.com","password":"secret123"}`),
	)
	loginRequest.Header.Set("Content-Type", "application/json")
	loginRecorder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected login status %d, got %d, body=%s", stdhttp.StatusOK, loginRecorder.Code, loginRecorder.Body.String())
	}

	var loginResponse appLoginResponse
	if err := json.NewDecoder(loginRecorder.Body).Decode(&loginResponse); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	return application, loginResponse.AccessToken
}
