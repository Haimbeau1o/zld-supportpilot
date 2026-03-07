# RAG Answer API Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为知识库提供最小可用的 RAG 检索与回答接口，支持引用返回与低置信度降级。

**Architecture:** 在 `internal/module/ai` 中建立独立的检索与回答服务，消费 `knowledge` 模块输出的 ready chunk；采用内存向量检索和模板式回答生成，先稳定检索排序、引用组装和降级边界，再为未来真实向量库和 LLM 留可替换接口。

**Tech Stack:** Go stdlib HTTP, in-memory vector retrieval, cosine similarity, template answer generation, current JWT auth, table-driven tests

### Task 1: 建立 AI 领域模型与向量检索抽象

**Files:**
- Create: `internal/module/ai/model.go`
- Create: `internal/module/ai/retriever.go`
- Create: `internal/module/ai/retriever_test.go`

**Step 1: Write the failing test**

- 编写 `retriever_test.go`，覆盖：
  - 问题与 chunk 的向量相似度排序符合预期
  - TopK 截断生效
  - 空 chunk 集合返回空结果而不是报错

**Step 2: Run test to verify it fails**

Run: `go test ./internal/module/ai -run 'Test(VectorRetriever|HashingEmbedder)'`
Expected: FAIL because AI models and retriever do not exist yet.

**Step 3: Write minimal implementation**

- 在 `model.go` 中定义回答状态、引用、回答结果和检索结果模型。
- 在 `retriever.go` 中定义 `VectorEmbedder`、`Retriever`，并实现内存哈希向量与余弦相似度检索。
- 在当前阶段用中文注释说明为什么选择“可解释的内存向量检索”作为过渡方案。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/module/ai -run 'Test(VectorRetriever|HashingEmbedder)'`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/module/ai/model.go internal/module/ai/retriever.go internal/module/ai/retriever_test.go
git commit -m "feat: add rag retriever foundation"
```

### Task 2: 实现回答服务、引用组装与降级逻辑

**Files:**
- Create: `internal/module/ai/service.go`
- Create: `internal/module/ai/service_test.go`
- Modify: `internal/module/knowledge/service.go`

**Step 1: Write the failing test**

- 编写 `service_test.go`，覆盖：
  - 命中相关 chunk 时返回 `answered` 和 citations
  - 问题无关或分数过低时返回 `degraded`
  - knowledge 无权限或知识库不存在时返回明确错误

**Step 2: Run test to verify it fails**

Run: `go test ./internal/module/ai -run 'TestAskKnowledge'`
Expected: FAIL because answer service and knowledge chunk source do not exist yet.

**Step 3: Write minimal implementation**

- 在 `service.go` 中实现 `AskKnowledgeQuestion`。
- 在 `knowledge.Service` 中补充“列出某知识库 ready chunk”的读取能力。
- 回答生成先使用模板式生成器，把引用和回答文本稳定下来。
- 在低置信度降级、引用返回和模块边界处补中文注释。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/module/ai -run 'TestAskKnowledge'`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/module/ai/service.go internal/module/ai/service_test.go internal/module/knowledge/service.go
git commit -m "feat: add rag answer service and degradation"
```

### Task 3: 暴露 AI 回答 HTTP 接口并装配应用

**Files:**
- Create: `internal/platform/http/ai_handler.go`
- Create: `internal/platform/http/ai_handler_test.go`
- Modify: `internal/platform/http/router.go`
- Modify: `internal/app/app.go`
- Create: `internal/app/app_rag_test.go`

**Step 1: Write the failing test**

- 编写 HTTP 测试，覆盖：
  - 正常回答返回 `answered`
  - 低置信度时返回 `degraded`
  - 未登录访问被拒绝
- 编写 app 测试，覆盖：上传文档 -> 等待 ready -> 调用回答接口 -> 返回引用。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/http -run TestAI`
Expected: FAIL because AI routes and handlers do not exist yet.

**Step 3: Write minimal implementation**

- 新增 `POST /api/v1/ai/knowledge/bases/{id}/answers`。
- 在 `router.go` 中接入 AI 模块依赖。
- 在 `app.go` 中装配 AI 服务与默认检索器/回答生成器。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/platform/http -run TestAI`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/platform/http/ai_handler.go internal/platform/http/ai_handler_test.go internal/platform/http/router.go internal/app/app.go internal/app/app_rag_test.go
git commit -m "feat: expose rag answer api"
```

### Task 4: 完成文档更新与全量验证

**Files:**
- Modify: `README.md`
- Modify: `docs/workitems/issue-007-RAG检索与回答接口/04-验证记录.md`
- Modify: `docs/workitems/issue-007-RAG检索与回答接口/05-工作日志.md`
- Modify: `docs/workitems/issue-007-RAG检索与回答接口/06-复盘总结.md`

**Step 1: Write the failing test**

- 本任务不新增功能测试；以上述分层测试和全量回归为验证依据。

**Step 2: Run test to verify baseline**

Run: `go test ./...`
Expected: PASS

**Step 3: Write minimal implementation**

- 更新 README 中的 AI 回答接口说明。
- 回填验证记录、工作日志和复盘总结。

**Step 4: Run test to verify it passes**

Run: `go test ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add README.md docs/workitems/issue-007-RAG检索与回答接口/04-验证记录.md docs/workitems/issue-007-RAG检索与回答接口/05-工作日志.md docs/workitems/issue-007-RAG检索与回答接口/06-复盘总结.md
git commit -m "docs: finalize issue 7 verification and notes"
```
