package http

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/knowledge"
	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/config"
)

type knowledgeBaseResponsePayload struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
}

type knowledgeBaseListResponsePayload struct {
	Items []knowledgeBaseResponsePayload `json:"items"`
}

type documentResponsePayload struct {
	ID              string                   `json:"id"`
	KnowledgeBaseID string                   `json:"knowledge_base_id"`
	OrganizationID  string                   `json:"organization_id"`
	Filename        string                   `json:"filename"`
	ContentType     string                   `json:"content_type"`
	SizeBytes       int64                    `json:"size_bytes"`
	StorageKey      string                   `json:"storage_key"`
	Status          knowledge.DocumentStatus `json:"status"`
	UploadedBy      string                   `json:"uploaded_by"`
}

type documentListResponsePayload struct {
	Items []documentResponsePayload `json:"items"`
}

func TestKnowledgeCreateBase(t *testing.T) {
	handler, tokenManager := newTestKnowledgeMux(t)
	agent := identity.IdentityContext{
		UserID:         "user-agent-1",
		OrganizationID: "org-1",
		Role:           identity.RoleAgent,
		Permissions:    identity.RolePermissions(identity.RoleAgent),
	}

	request := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/v1/knowledge/bases",
		bytes.NewBufferString(`{"name":"IT 支持知识库","description":"用于沉淀 IT 文档"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+issueKnowledgeToken(t, tokenManager, agent))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusCreated, recorder.Code, recorder.Body.String())
	}

	var response knowledgeBaseResponsePayload
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode knowledge base response: %v", err)
	}

	if response.ID == "" {
		t.Fatalf("expected knowledge base id to be returned")
	}
}

func TestKnowledgeListBases(t *testing.T) {
	handler, tokenManager := newTestKnowledgeMux(t)
	agent := identity.IdentityContext{UserID: "user-agent-1", OrganizationID: "org-1", Role: identity.RoleAgent, Permissions: identity.RolePermissions(identity.RoleAgent)}
	otherOrgAgent := identity.IdentityContext{UserID: "user-agent-2", OrganizationID: "org-2", Role: identity.RoleAgent, Permissions: identity.RolePermissions(identity.RoleAgent)}

	createKnowledgeBaseViaHTTP(t, handler, tokenManager, agent, `{"name":"IT 支持知识库","description":"用于沉淀 IT 文档"}`)
	createKnowledgeBaseViaHTTP(t, handler, tokenManager, agent, `{"name":"HR 知识库","description":"用于沉淀 HR 文档"}`)
	createKnowledgeBaseViaHTTP(t, handler, tokenManager, otherOrgAgent, `{"name":"其他租户知识库","description":"其他组织"}`)

	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/knowledge/bases", nil)
	request.Header.Set("Authorization", "Bearer "+issueKnowledgeToken(t, tokenManager, agent))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response knowledgeBaseListResponsePayload
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode knowledge base list response: %v", err)
	}
	if len(response.Items) != 2 {
		t.Fatalf("expected 2 knowledge bases in tenant, got %d", len(response.Items))
	}
}

func TestKnowledgeUploadDocument(t *testing.T) {
	handler, tokenManager := newTestKnowledgeMux(t)
	agent := identity.IdentityContext{UserID: "user-agent-1", OrganizationID: "org-1", Role: identity.RoleAgent, Permissions: identity.RolePermissions(identity.RoleAgent)}
	knowledgeBase := createKnowledgeBaseViaHTTP(t, handler, tokenManager, agent, `{"name":"IT 支持知识库","description":"用于沉淀 IT 文档"}`)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("filename", "vpn-guide.pdf"); err != nil {
		t.Fatalf("write filename field: %v", err)
	}
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

	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/knowledge/bases/"+knowledgeBase.ID+"/documents", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+issueKnowledgeToken(t, tokenManager, agent))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusCreated, recorder.Code, recorder.Body.String())
	}

	var response documentResponsePayload
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}
	if response.ID == "" || response.StorageKey == "" {
		t.Fatalf("expected uploaded document metadata to be returned")
	}
	if response.Status != knowledge.DocumentStatusUploaded {
		t.Fatalf("expected document status %q, got %q", knowledge.DocumentStatusUploaded, response.Status)
	}
}

func TestKnowledgeListDocumentsAndGetDetail(t *testing.T) {
	handler, tokenManager := newTestKnowledgeMux(t)
	agent := identity.IdentityContext{UserID: "user-agent-1", OrganizationID: "org-1", Role: identity.RoleAgent, Permissions: identity.RolePermissions(identity.RoleAgent)}
	knowledgeBase := createKnowledgeBaseViaHTTP(t, handler, tokenManager, agent, `{"name":"IT 支持知识库","description":"用于沉淀 IT 文档"}`)
	document := uploadDocumentViaHTTP(t, handler, tokenManager, agent, knowledgeBase.ID, "vpn-guide.pdf", []byte("hello knowledge"))

	listRequest := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/knowledge/bases/"+knowledgeBase.ID+"/documents", nil)
	listRequest.Header.Set("Authorization", "Bearer "+issueKnowledgeToken(t, tokenManager, agent))
	listRecorder := httptest.NewRecorder()
	handler.ServeHTTP(listRecorder, listRequest)
	if listRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected list status %d, got %d, body=%s", stdhttp.StatusOK, listRecorder.Code, listRecorder.Body.String())
	}

	var listResponse documentListResponsePayload
	if err := json.NewDecoder(listRecorder.Body).Decode(&listResponse); err != nil {
		t.Fatalf("decode list documents response: %v", err)
	}
	if len(listResponse.Items) != 1 {
		t.Fatalf("expected 1 document, got %d", len(listResponse.Items))
	}

	detailRequest := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/knowledge/documents/"+document.ID, nil)
	detailRequest.Header.Set("Authorization", "Bearer "+issueKnowledgeToken(t, tokenManager, agent))
	detailRecorder := httptest.NewRecorder()
	handler.ServeHTTP(detailRecorder, detailRequest)
	if detailRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected detail status %d, got %d, body=%s", stdhttp.StatusOK, detailRecorder.Code, detailRecorder.Body.String())
	}
}

func TestKnowledgeRequiresAuthentication(t *testing.T) {
	handler, _ := newTestKnowledgeMux(t)
	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/knowledge/bases", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d, body=%s", stdhttp.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
}

func newTestKnowledgeMux(t *testing.T) (*stdhttp.ServeMux, identity.TokenManager) {
	t.Helper()

	knowledgeService := knowledge.NewService(
		knowledge.NewMemoryKnowledgeBaseRepository(),
		knowledge.NewMemoryDocumentRepository(),
		knowledge.NewMemoryObjectStorage(),
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
		Knowledge: &KnowledgeDependencies{
			KnowledgeService: knowledgeService,
		},
	}), tokenManager
}

func createKnowledgeBaseViaHTTP(t *testing.T, handler stdhttp.Handler, tokenManager identity.TokenManager, actor identity.IdentityContext, body string) knowledgeBaseResponsePayload {
	t.Helper()

	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/knowledge/bases", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+issueKnowledgeToken(t, tokenManager, actor))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected create knowledge base status %d, got %d, body=%s", stdhttp.StatusCreated, recorder.Code, recorder.Body.String())
	}

	var response knowledgeBaseResponsePayload
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode create knowledge base response: %v", err)
	}

	return response
}

func uploadDocumentViaHTTP(t *testing.T, handler stdhttp.Handler, tokenManager identity.TokenManager, actor identity.IdentityContext, knowledgeBaseID string, filename string, content []byte) documentResponsePayload {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fileWriter, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fileWriter.Write(content); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/knowledge/bases/"+knowledgeBaseID+"/documents", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+issueKnowledgeToken(t, tokenManager, actor))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)
	if recorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected upload status %d, got %d, body=%s", stdhttp.StatusCreated, recorder.Code, recorder.Body.String())
	}

	var response documentResponsePayload
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}

	return response
}

func issueKnowledgeToken(t *testing.T, tokenManager identity.TokenManager, actor identity.IdentityContext) string {
	t.Helper()

	accessToken, err := tokenManager.Issue(actor)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	return accessToken
}
