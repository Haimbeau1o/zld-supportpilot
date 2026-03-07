# Knowledge Upload Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 `ZLD SupportPilot` 建立最小可用的知识库与文档上传链路，在认证与租户边界之上支持知识库创建、文档上传与元数据追踪。

**Architecture:** 采用 interface-first 的 knowledge 模块设计，在 `internal/module/knowledge` 中定义知识库、文档元数据、上传状态、服务层、仓储接口与存储接口；在 `internal/platform/http` 暴露受保护的知识库与文档上传 API，并复用当前 `identity` 模块提供的租户与权限上下文。当前阶段优先稳定上传入口和元数据边界，不在本计划中引入切块、向量索引、真实 MinIO 适配与检索链路。

**Tech Stack:** Go stdlib HTTP, multipart upload, in-memory repository and object storage, table-driven tests, current auth middleware

### Task 1: 定义知识库与文档领域模型

**Files:**
- Create: `internal/module/knowledge/model.go`
- Test: `internal/module/knowledge/model_test.go`

**Step 1: Write the failing test**

- 编写 `model_test.go`，验证文档初始状态和必要元数据字段行为是否符合预期。
- 至少覆盖：新文档默认状态为 `uploaded`，文档状态集合包含 `processing`、`ready`、`failed`。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/module/knowledge -run TestDocumentDefaults`
Expected: FAIL because knowledge models do not exist yet.

**Step 3: Write minimal implementation**

- 在 `model.go` 中定义 `KnowledgeBase`、`Document`、`DocumentStatus`。
- 在状态集合和核心元数据字段处补中文注释，说明为什么当前阶段就保留可扩展状态字段。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/module/knowledge -run TestDocumentDefaults`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/module/knowledge/model.go internal/module/knowledge/model_test.go
git commit -m "feat: add knowledge models and document statuses"
```

### Task 2: 建立 knowledge 仓储、存储接口、服务层与内存适配

**Files:**
- Create: `internal/module/knowledge/repository.go`
- Create: `internal/module/knowledge/storage.go`
- Create: `internal/module/knowledge/memory_repository.go`
- Create: `internal/module/knowledge/memory_storage.go`
- Create: `internal/module/knowledge/service.go`
- Test: `internal/module/knowledge/service_test.go`

**Step 1: Write the failing test**

- 编写 `service_test.go`，覆盖：创建知识库成功、无写权限创建失败、上传文档成功、空文件上传失败、客服可查看租户内知识库与文档元数据。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/module/knowledge -run 'Test(CreateKnowledgeBase|UploadDocument|ListKnowledgeBases|ListDocuments|GetDocument)'`
Expected: FAIL because service, repositories and storage do not exist yet.

**Step 3: Write minimal implementation**

- 在 `repository.go` 中定义知识库与文档仓储接口。
- 在 `storage.go` 中定义对象存储接口。
- 在 `memory_repository.go` 和 `memory_storage.go` 中实现线程安全的内存适配。
- 在 `service.go` 中实现 `CreateKnowledgeBase`、`ListKnowledgeBases`、`UploadDocument`、`ListDocuments`、`GetDocument`。
- 在知识库权限判断、跨租户保护、上传状态与存储键设计处补中文注释。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/module/knowledge -run 'Test(CreateKnowledgeBase|UploadDocument|ListKnowledgeBases|ListDocuments|GetDocument)'`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/module/knowledge/repository.go internal/module/knowledge/storage.go internal/module/knowledge/memory_repository.go internal/module/knowledge/memory_storage.go internal/module/knowledge/service.go internal/module/knowledge/service_test.go
git commit -m "feat: add knowledge service and storage adapters"
```

### Task 3: 暴露知识库与文档上传 HTTP 接口

**Files:**
- Create: `internal/platform/http/knowledge_handler.go`
- Modify: `internal/platform/http/router.go`
- Test: `internal/platform/http/knowledge_handler_test.go`

**Step 1: Write the failing test**

- 编写 HTTP 测试，覆盖：创建知识库、列出知识库、上传文档、列出文档、查询文档详情、未带令牌访问被拒绝。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/http -run TestKnowledge`
Expected: FAIL because knowledge handlers and routes do not exist yet.

**Step 3: Write minimal implementation**

- 在 `knowledge_handler.go` 中实现知识库创建 / 查询、文档上传、文档元数据查询。
- 在 `router.go` 中注册受保护的 knowledge 路由。
- 使用 multipart/form-data 处理文件上传，不引入额外框架。
- 在上传入口、权限错误与元数据返回处补中文注释。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/platform/http -run TestKnowledge`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/platform/http/knowledge_handler.go internal/platform/http/router.go internal/platform/http/knowledge_handler_test.go
git commit -m "feat: add knowledge handlers and upload routes"
```

### Task 4: 完成应用装配、文档更新与回归验证

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`
- Modify: `README.md`
- Modify: `docs/workitems/issue-005-知识库与文档上传/04-验证记录.md`
- Modify: `docs/workitems/issue-005-知识库与文档上传/05-工作日志.md`
- Modify: `docs/workitems/issue-005-知识库与文档上传/06-复盘总结.md`

**Step 1: Write the failing test**

- 编写应用装配测试，验证在完整 app 启动后，具备知识写权限的用户可以创建知识库并上传文档。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/app -run TestNewWiresKnowledgeRoutes`
Expected: FAIL because app has not wired knowledge dependencies yet.

**Step 3: Write minimal implementation**

- 在 `app.go` 中装配 knowledge service。
- 更新 `app_test.go` 和 `README.md` 的运行与接口说明。
- 回填 issue #5 的验证记录、工作日志和复盘总结。

**Step 4: Run test to verify it passes**

Run: `go test ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/app/app.go internal/app/app_test.go README.md docs/workitems/issue-005-知识库与文档上传/04-验证记录.md docs/workitems/issue-005-知识库与文档上传/05-工作日志.md docs/workitems/issue-005-知识库与文档上传/06-复盘总结.md
git commit -m "feat: wire knowledge upload flow and document verification"
```
