package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	stdhttp "net/http"
	"strings"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/knowledge"
)

type knowledgeService interface {
	CreateKnowledgeBase(actor identity.IdentityContext, input knowledge.CreateKnowledgeBaseInput) (knowledge.KnowledgeBase, error)
	ListKnowledgeBases(actor identity.IdentityContext) ([]knowledge.KnowledgeBase, error)
	UploadDocument(actor identity.IdentityContext, input knowledge.UploadDocumentInput) (knowledge.Document, error)
	ListDocuments(actor identity.IdentityContext, input knowledge.ListDocumentsInput) ([]knowledge.Document, error)
	GetDocument(actor identity.IdentityContext, documentID string) (knowledge.Document, error)
	ListDocumentProcessingTasks(actor identity.IdentityContext, documentID string) ([]knowledge.DocumentProcessingTask, error)
	RetryDocumentProcessing(actor identity.IdentityContext, documentID string) (knowledge.DocumentProcessingTask, error)
}

type KnowledgeDependencies struct {
	KnowledgeService knowledgeService
}

type KnowledgeHandler struct {
	knowledgeService knowledgeService
}

type createKnowledgeBaseRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type knowledgeBaseResponse struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
}

type knowledgeBaseListResponse struct {
	Items []knowledgeBaseResponse `json:"items"`
}

type documentResponse struct {
	ID                 string                   `json:"id"`
	KnowledgeBaseID    string                   `json:"knowledge_base_id"`
	OrganizationID     string                   `json:"organization_id"`
	Filename           string                   `json:"filename"`
	ContentType        string                   `json:"content_type"`
	SizeBytes          int64                    `json:"size_bytes"`
	StorageKey         string                   `json:"storage_key"`
	Status             knowledge.DocumentStatus `json:"status"`
	UploadedBy         string                   `json:"uploaded_by"`
	LastTaskID         string                   `json:"last_task_id"`
	ProcessingAttempts int                      `json:"processing_attempts"`
	ChunkCount         int                      `json:"chunk_count"`
	LastError          string                   `json:"last_error,omitempty"`
}

type documentListResponse struct {
	Items []documentResponse `json:"items"`
}

type processingTaskResponse struct {
	ID           string                         `json:"id"`
	DocumentID   string                         `json:"document_id"`
	Status       knowledge.ProcessingTaskStatus `json:"status"`
	Stage        knowledge.ProcessingStage      `json:"stage"`
	Attempt      int                            `json:"attempt"`
	MaxAttempts  int                            `json:"max_attempts"`
	ErrorMessage string                         `json:"error_message,omitempty"`
}

type processingTaskListResponse struct {
	Items []processingTaskResponse `json:"items"`
}

func NewKnowledgeHandler(knowledgeService knowledgeService) KnowledgeHandler {
	return KnowledgeHandler{knowledgeService: knowledgeService}
}

func (handler KnowledgeHandler) CreateKnowledgeBase(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	actor, ok := identityContextFromContext(request.Context())
	if !ok {
		writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "当前请求缺少身份上下文")
		return
	}

	var input createKnowledgeBaseRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		writeError(writer, stdhttp.StatusBadRequest, "invalid_request", "请求体不是合法的 JSON")
		return
	}

	knowledgeBase, err := handler.knowledgeService.CreateKnowledgeBase(actor, knowledge.CreateKnowledgeBaseInput{
		Name:        input.Name,
		Description: input.Description,
	})
	if err != nil {
		handleKnowledgeError(writer, err, "创建知识库失败")
		return
	}

	writeJSON(writer, stdhttp.StatusCreated, newKnowledgeBaseResponse(knowledgeBase))
}

func (handler KnowledgeHandler) ListKnowledgeBases(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	actor, ok := identityContextFromContext(request.Context())
	if !ok {
		writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "当前请求缺少身份上下文")
		return
	}

	knowledgeBases, err := handler.knowledgeService.ListKnowledgeBases(actor)
	if err != nil {
		handleKnowledgeError(writer, err, "查询知识库失败")
		return
	}

	items := make([]knowledgeBaseResponse, 0, len(knowledgeBases))
	for _, item := range knowledgeBases {
		items = append(items, newKnowledgeBaseResponse(item))
	}

	writeJSON(writer, stdhttp.StatusOK, knowledgeBaseListResponse{Items: items})
}

func (handler KnowledgeHandler) UploadDocument(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	actor, ok := identityContextFromContext(request.Context())
	if !ok {
		writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "当前请求缺少身份上下文")
		return
	}

	file, fileHeader, err := request.FormFile("file")
	if err != nil {
		writeError(writer, stdhttp.StatusBadRequest, "invalid_request", "缺少上传文件")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		writeError(writer, stdhttp.StatusBadRequest, "invalid_request", "读取上传文件失败")
		return
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" && len(content) > 0 {
		contentType = stdhttp.DetectContentType(content)
	}

	document, err := handler.knowledgeService.UploadDocument(actor, knowledge.UploadDocumentInput{
		KnowledgeBaseID: strings.TrimSpace(request.PathValue("id")),
		Filename:        fileHeader.Filename,
		ContentType:     contentType,
		Content:         content,
	})
	if err != nil {
		handleKnowledgeError(writer, err, "上传文档失败")
		return
	}

	writeJSON(writer, stdhttp.StatusCreated, newDocumentResponse(document))
}

func (handler KnowledgeHandler) ListDocuments(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	actor, ok := identityContextFromContext(request.Context())
	if !ok {
		writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "当前请求缺少身份上下文")
		return
	}

	documents, err := handler.knowledgeService.ListDocuments(actor, knowledge.ListDocumentsInput{
		KnowledgeBaseID: strings.TrimSpace(request.PathValue("id")),
	})
	if err != nil {
		handleKnowledgeError(writer, err, "查询文档失败")
		return
	}

	items := make([]documentResponse, 0, len(documents))
	for _, item := range documents {
		items = append(items, newDocumentResponse(item))
	}

	writeJSON(writer, stdhttp.StatusOK, documentListResponse{Items: items})
}

func (handler KnowledgeHandler) GetDocument(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	actor, ok := identityContextFromContext(request.Context())
	if !ok {
		writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "当前请求缺少身份上下文")
		return
	}

	document, err := handler.knowledgeService.GetDocument(actor, strings.TrimSpace(request.PathValue("id")))
	if err != nil {
		handleKnowledgeError(writer, err, "查询文档详情失败")
		return
	}

	writeJSON(writer, stdhttp.StatusOK, newDocumentResponse(document))
}

func (handler KnowledgeHandler) ListDocumentProcessingTasks(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	actor, ok := identityContextFromContext(request.Context())
	if !ok {
		writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "当前请求缺少身份上下文")
		return
	}

	tasks, err := handler.knowledgeService.ListDocumentProcessingTasks(actor, strings.TrimSpace(request.PathValue("id")))
	if err != nil {
		handleKnowledgeError(writer, err, "查询文档处理任务失败")
		return
	}

	items := make([]processingTaskResponse, 0, len(tasks))
	for _, item := range tasks {
		items = append(items, newProcessingTaskResponse(item))
	}

	writeJSON(writer, stdhttp.StatusOK, processingTaskListResponse{Items: items})
}

func (handler KnowledgeHandler) RetryDocumentProcessing(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	actor, ok := identityContextFromContext(request.Context())
	if !ok {
		writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "当前请求缺少身份上下文")
		return
	}

	task, err := handler.knowledgeService.RetryDocumentProcessing(actor, strings.TrimSpace(request.PathValue("id")))
	if err != nil {
		handleKnowledgeError(writer, err, "重试文档处理失败")
		return
	}

	writeJSON(writer, stdhttp.StatusAccepted, newProcessingTaskResponse(task))
}

func newKnowledgeBaseResponse(knowledgeBase knowledge.KnowledgeBase) knowledgeBaseResponse {
	return knowledgeBaseResponse{
		ID:             knowledgeBase.ID,
		OrganizationID: knowledgeBase.OrganizationID,
		Name:           knowledgeBase.Name,
		Description:    knowledgeBase.Description,
	}
}

func newDocumentResponse(document knowledge.Document) documentResponse {
	return documentResponse{
		ID:                 document.ID,
		KnowledgeBaseID:    document.KnowledgeBaseID,
		OrganizationID:     document.OrganizationID,
		Filename:           document.Filename,
		ContentType:        document.ContentType,
		SizeBytes:          document.SizeBytes,
		StorageKey:         document.StorageKey,
		Status:             document.Status,
		UploadedBy:         document.UploadedBy,
		LastTaskID:         document.LastTaskID,
		ProcessingAttempts: document.ProcessingAttempts,
		ChunkCount:         document.ChunkCount,
		LastError:          document.LastError,
	}
}

func newProcessingTaskResponse(task knowledge.DocumentProcessingTask) processingTaskResponse {
	return processingTaskResponse{
		ID:           task.ID,
		DocumentID:   task.DocumentID,
		Status:       task.Status,
		Stage:        task.Stage,
		Attempt:      task.Attempt,
		MaxAttempts:  task.MaxAttempts,
		ErrorMessage: task.ErrorMessage,
	}
}

func handleKnowledgeError(writer stdhttp.ResponseWriter, err error, fallbackMessage string) {
	switch {
	case errors.Is(err, knowledge.ErrKnowledgeForbidden):
		writeError(writer, stdhttp.StatusForbidden, "forbidden", "当前身份没有权限执行该知识库操作")
	case errors.Is(err, knowledge.ErrKnowledgeBaseNotFound):
		writeError(writer, stdhttp.StatusNotFound, "knowledge_base_not_found", "知识库不存在")
	case errors.Is(err, knowledge.ErrDocumentNotFound):
		writeError(writer, stdhttp.StatusNotFound, "document_not_found", "文档不存在")
	case errors.Is(err, knowledge.ErrDocumentProcessingTaskNotFound):
		writeError(writer, stdhttp.StatusNotFound, "document_processing_task_not_found", "文档处理任务不存在")
	case errors.Is(err, knowledge.ErrInvalidKnowledgeInput), errors.Is(err, knowledge.ErrInvalidDocumentInput):
		writeError(writer, stdhttp.StatusBadRequest, "invalid_knowledge_request", err.Error())
	case errors.Is(err, knowledge.ErrDocumentRetryNotAllowed):
		writeError(writer, stdhttp.StatusConflict, "document_retry_not_allowed", err.Error())
	case errors.Is(err, knowledge.ErrKnowledgeProcessingDisabled):
		writeError(writer, stdhttp.StatusServiceUnavailable, "knowledge_processing_unavailable", "当前环境未启用文档异步处理")
	default:
		writeError(writer, stdhttp.StatusInternalServerError, "internal_error", fallbackMessage)
	}
}

func registerKnowledgeRoutes(mux *stdhttp.ServeMux, runtime routeRuntime, authDependencies AuthDependencies, knowledgeDependencies KnowledgeDependencies) error {
	if authDependencies.TokenManager == nil {
		return fmt.Errorf("token manager is required for knowledge routes")
	}
	if knowledgeDependencies.KnowledgeService == nil {
		return fmt.Errorf("knowledge service is required")
	}

	authMiddleware := NewAuthMiddleware(authDependencies.TokenManager)
	knowledgeHandler := NewKnowledgeHandler(knowledgeDependencies.KnowledgeService)

	// 知识库与文档接口全部挂在鉴权之后，避免后续接入异步处理或检索链路时出现“资源有归属但接口没身份边界”的问题。
	mux.Handle("POST /api/v1/knowledge/bases", runtime.protected("POST /api/v1/knowledge/bases", authMiddleware, stdhttp.HandlerFunc(knowledgeHandler.CreateKnowledgeBase)))
	mux.Handle("GET /api/v1/knowledge/bases", runtime.protected("GET /api/v1/knowledge/bases", authMiddleware, stdhttp.HandlerFunc(knowledgeHandler.ListKnowledgeBases)))
	// 文档上传与重试会消耗解析、切块和索引资源，因此被定义为当前阶段的高成本限流边界。
	mux.Handle("POST /api/v1/knowledge/bases/{id}/documents", runtime.protectedRateLimited("POST /api/v1/knowledge/bases/{id}/documents", authMiddleware, identityOrClientRateLimitKey, stdhttp.HandlerFunc(knowledgeHandler.UploadDocument)))
	mux.Handle("GET /api/v1/knowledge/bases/{id}/documents", runtime.protected("GET /api/v1/knowledge/bases/{id}/documents", authMiddleware, stdhttp.HandlerFunc(knowledgeHandler.ListDocuments)))
	mux.Handle("GET /api/v1/knowledge/documents/{id}", runtime.protected("GET /api/v1/knowledge/documents/{id}", authMiddleware, stdhttp.HandlerFunc(knowledgeHandler.GetDocument)))
	mux.Handle("GET /api/v1/knowledge/documents/{id}/tasks", runtime.protected("GET /api/v1/knowledge/documents/{id}/tasks", authMiddleware, stdhttp.HandlerFunc(knowledgeHandler.ListDocumentProcessingTasks)))
	mux.Handle("POST /api/v1/knowledge/documents/{id}/retry", runtime.protectedRateLimited("POST /api/v1/knowledge/documents/{id}/retry", authMiddleware, identityOrClientRateLimitKey, stdhttp.HandlerFunc(knowledgeHandler.RetryDocumentProcessing)))
	return nil
}
