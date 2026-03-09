# 工单结论沉淀为知识候选与审核流 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 `ZLD SupportPilot` 增加“已解决工单 -> 知识候选 -> 审核入库 -> 问答命中”的最小真实业务闭环。

**Architecture:** 在 `internal/module/knowledge` 中新增知识候选模型、repository 与 service，并通过 ticket 只读边界读取已解决工单上下文。候选审核通过后不新造处理链路，而是把候选转成 markdown 文档，复用现有 `knowledge` 文档入库、对象存储和异步处理能力。HTTP 层补充候选创建、列表和审核接口，app 层负责 memory / postgres 双装配与 schema 接入。

**Tech Stack:** Go、模块化单体、现有 ticket / knowledge 模块、memory / postgres 双持久化、标准库 `net/http`、现有异步 processing pipeline

### Task 1: 为知识候选 service 补 RED 测试

**Files:**
- Modify: `internal/module/knowledge/model.go`
- Modify: `internal/module/knowledge/service.go`
- Create: `internal/module/knowledge/candidate_test.go`

**Step 1: Write the failing test**

新增测试覆盖：
- 只有 `resolved` 工单可创建候选
- 候选创建后状态为 `pending_review`
- 审核通过时会生成文档并创建处理任务
- 审核驳回时不生成文档
- 审批后的知识内容可被后续处理链路消费

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test -count=1 ./internal/module/knowledge`
Expected: FAIL，提示 candidate 模型或 service 方法缺失

**Step 3: Write minimal implementation**

在 `knowledge` 模块中补齐候选模型、ticket 读取边界与 service 最小规则实现。

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test -count=1 ./internal/module/knowledge`
Expected: PASS

### Task 2: 为 memory / postgres repository 与 schema 补 RED 测试

**Files:**
- Modify: `internal/module/knowledge/repository.go`
- Modify: `internal/module/knowledge/memory_repository.go`
- Modify: `internal/module/knowledge/postgres_repository.go`
- Modify: `internal/module/knowledge/postgres_repository_test.go`
- Modify: `internal/platform/persistence/schema.sql`

**Step 1: Write the failing test**

覆盖：
- 候选保存与按知识库列表查询
- 审核状态更新
- 审核通过后 `approved_document_id` 持久化

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test -count=1 ./internal/module/knowledge -run 'TestPostgres.*Candidate'`
Expected: FAIL

**Step 3: Write minimal implementation**

补齐候选 repository 的 memory / postgres 双实现与 schema 表结构。

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test -count=1 ./internal/module/knowledge -run 'TestPostgres.*Candidate'`
Expected: PASS

### Task 3: 为候选创建 / 列表 / 审核 API 补 RED 测试

**Files:**
- Modify: `internal/platform/http/knowledge_handler.go`
- Modify: `internal/platform/http/knowledge_handler_test.go`

**Step 1: Write the failing test**

覆盖：
- 成功创建知识候选
- 成功查询候选列表
- 审核通过成功并返回生成文档信息
- 审核驳回成功
- end user 无权限执行上述动作

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test -count=1 ./internal/platform/http -run 'TestKnowledgeCandidate'`
Expected: FAIL

**Step 3: Write minimal implementation**

新增候选 request / response、handler 方法和路由注册。

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test -count=1 ./internal/platform/http -run 'TestKnowledgeCandidate'`
Expected: PASS

### Task 4: 为 app 装配与端到端联动补实现

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`
- Modify: `README.md`
- Modify: `docs/workitems/issue-027-工单结论沉淀为知识候选与审核流/*.md`

**Step 1: Write the failing test**

增加集成测试，验证：
- resolved 工单可生成候选
- 审核通过后候选进入文档处理链路
- 新知识可以被后续问答命中

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test -count=1 ./internal/app -run 'TestKnowledgeCandidate'`
Expected: FAIL

**Step 3: Write minimal implementation**

接入 knowledge candidate repository / service / handler，更新 README 与工作文档。

**Step 4: Run final verification**

Run: `GOCACHE=/tmp/go-build go test -count=1 ./...`
Expected: PASS
