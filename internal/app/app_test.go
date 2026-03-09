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
	"testing"

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
	ResultType string `json:"result_type"`
	TicketID   string `json:"ticket_id"`
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
