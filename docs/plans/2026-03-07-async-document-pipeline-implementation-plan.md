# Async Document Pipeline Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为知识库文档建立异步处理流水线，让上传接口只负责存储和入队，后台再完成解析、切块、索引，并支持失败追踪与人工重试。

**Architecture:** 在 `knowledge` 模块中新增显式处理任务模型和 chunk 模型；服务层负责创建任务、推进任务状态和更新文档状态；后台 worker 通过内存队列消费任务并串行执行 parser/chunker/indexer 三段式处理。HTTP 层只暴露任务查询与重试入口，不引入真实 MQ 或向量库。

**Tech Stack:** Go stdlib, goroutines + channels + context, in-memory repositories, table-driven tests, current auth middleware

### Task 1: 扩展领域模型与基础接口

**Files:**
- Modify: `internal/module/knowledge/model.go`
- Modify: `internal/module/knowledge/model_test.go`
- Modify: `internal/module/knowledge/repository.go`
- Modify: `internal/module/knowledge/storage.go`
- Modify: `internal/module/knowledge/memory_repository.go`
- Modify: `internal/module/knowledge/memory_storage.go`

**Step 1: Write the failing test**

- 在 `model_test.go` 中新增断言：处理任务默认状态为 `queued`，阶段为 `parse`；文档默认处理扩展字段为零值但可写入。
- 为对象存储读取能力补测试准备。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/module/knowledge -run 'Test(DocumentDefaults|ProcessingTaskDefaults)'`
Expected: FAIL because task model and storage read capability do not exist yet.

**Step 3: Write minimal implementation**

- 在 `model.go` 中定义 `DocumentProcessingTask`、`ProcessingTaskStatus`、`ProcessingStage`、`DocumentChunk`。
- 扩展 `Document` 的处理元数据字段，并在关键边界处补中文注释。
- 在 `repository.go` 中新增处理任务仓储与 chunk 仓储接口。
- 在 `storage.go` 中为对象存储增加读取能力；在内存适配中落地线程安全实现。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/module/knowledge -run 'Test(DocumentDefaults|ProcessingTaskDefaults)'`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/module/knowledge/model.go internal/module/knowledge/model_test.go internal/module/knowledge/repository.go internal/module/knowledge/storage.go internal/module/knowledge/memory_repository.go internal/module/knowledge/memory_storage.go
git commit -m "feat: add knowledge processing models and storage reads"
```

### Task 2: 实现处理任务服务与解析/切块/索引边界

**Files:**
- Modify: `internal/module/knowledge/service.go`
- Create: `internal/module/knowledge/processing.go`
- Create: `internal/module/knowledge/processing_test.go`

**Step 1: Write the failing test**

- 编写 `processing_test.go`，覆盖：
  - 上传后会创建处理任务并记录 attempt
  - 手动执行 `ProcessTask` 后，文档进入 `ready` 且生成 chunk
  - 解析失败时，任务与文档都进入 `failed`
  - 失败后可通过 `RetryDocumentProcessing` 创建新任务

**Step 2: Run test to verify it fails**

Run: `go test ./internal/module/knowledge -run 'Test(UploadDocumentCreatesProcessingTask|ProcessTaskTransitionsDocumentToReady|ProcessTaskFailureAllowsRetry)'`
Expected: FAIL because processing services and boundaries do not exist yet.

**Step 3: Write minimal implementation**

- 在 `processing.go` 中定义 `DocumentParser`、`DocumentChunker`、`ChunkIndexer`、`ProcessingTaskDispatcher` 和 `AsyncProcessor`。
- 在 `service.go` 中实现任务入队、任务查询、重试和同步 `ProcessTask` 逻辑。
- 保证失败不回滚原始文档，而是沉淀失败任务和错误信息。
- 在上传入队、阶段推进、失败状态和重试策略处补中文注释。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/module/knowledge -run 'Test(UploadDocumentCreatesProcessingTask|ProcessTaskTransitionsDocumentToReady|ProcessTaskFailureAllowsRetry)'`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/module/knowledge/service.go internal/module/knowledge/processing.go internal/module/knowledge/processing_test.go
git commit -m "feat: add async knowledge processing pipeline"
```

### Task 3: 增加异步 worker 与 HTTP 可观测入口

**Files:**
- Modify: `internal/platform/http/knowledge_handler.go`
- Modify: `internal/platform/http/router.go`
- Modify: `internal/platform/http/knowledge_handler_test.go`
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`

**Step 1: Write the failing test**

- 在 `knowledge_handler_test.go` 中新增：任务列表查询、失败文档重试。
- 在 `app_test.go` 中补充关闭资源或等待后台处理的最小校验。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/http -run 'TestKnowledge(ListTasks|RetryDocument)'`
Expected: FAIL because routes and handlers do not exist yet.

**Step 3: Write minimal implementation**

- 在 handler 中新增 `GET /api/v1/knowledge/documents/{id}/tasks` 和 `POST /api/v1/knowledge/documents/{id}/retry`。
- 在 `app.go` 中装配内存任务仓储、chunk 仓储、队列、默认 parser/chunker/indexer，并启动后台 worker。
- 提供 `Close` 或等价回收手段，避免测试残留 goroutine。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/platform/http -run 'TestKnowledge(ListTasks|RetryDocument)'`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/platform/http/knowledge_handler.go internal/platform/http/router.go internal/platform/http/knowledge_handler_test.go internal/app/app.go internal/app/app_test.go
git commit -m "feat: expose knowledge processing status and retry routes"
```

### Task 4: 回填文档与完成全量验证

**Files:**
- Modify: `README.md`
- Modify: `docs/workitems/issue-006-异步文档处理流水线/04-验证记录.md`
- Modify: `docs/workitems/issue-006-异步文档处理流水线/05-工作日志.md`
- Modify: `docs/workitems/issue-006-异步文档处理流水线/06-复盘总结.md`

**Step 1: Write the failing test**

- 本任务以文档与回归验证为主，不新增功能测试；只确认前述测试覆盖已足够支撑验收。

**Step 2: Run test to verify baseline**

Run: `go test ./...`
Expected: PASS

**Step 3: Write minimal implementation**

- 更新 README 中的知识链路说明。
- 回填验证记录、工作日志、复盘总结，沉淀实现思路、测试结果和后续演进点。

**Step 4: Run test to verify it passes**

Run: `go test ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add README.md docs/workitems/issue-006-异步文档处理流水线/04-验证记录.md docs/workitems/issue-006-异步文档处理流水线/05-工作日志.md docs/workitems/issue-006-异步文档处理流水线/06-复盘总结.md
git commit -m "docs: finalize issue 6 verification and notes"
```
