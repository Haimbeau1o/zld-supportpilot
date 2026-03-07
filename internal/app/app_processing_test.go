package app

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/knowledge"
)

type appDocumentResponse struct {
	ID         string                   `json:"id"`
	Status     knowledge.DocumentStatus `json:"status"`
	ChunkCount int                      `json:"chunk_count"`
}

func TestNewWiresKnowledgeAsyncProcessing(t *testing.T) {
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
	if _, err := fileWriter.Write([]byte("first paragraph\n\nsecond paragraph")); err != nil {
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

	var uploadResponse appDocumentResponse
	if err := json.NewDecoder(uploadRecorder.Body).Decode(&uploadResponse); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}
	if uploadResponse.Status != knowledge.DocumentStatusUploaded {
		t.Fatalf("expected uploaded document status %q, got %q", knowledge.DocumentStatusUploaded, uploadResponse.Status)
	}

	deadline := time.Now().Add(2 * time.Second)
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
		if detailResponse.Status == knowledge.DocumentStatusReady {
			if detailResponse.ChunkCount == 0 {
				t.Fatalf("expected chunk count to be updated when ready")
			}
			return
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("expected document to become ready before deadline")
}
