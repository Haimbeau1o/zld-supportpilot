# AI Ticket Assist Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为工单处理方提供一个统一的 AI 辅助接口，生成分类建议、处理摘要和回复草稿，并将 AI 结果写入工单内部时间线。

**Architecture:** 在 `internal/module/ai` 中新增 ticket assist 服务，依赖 ticket 读取/写入边界和现有知识增强回答能力。AI 输出只作为建议返回，并复用 `internal note` 记录执行轨迹，不自动修改工单主数据。

**Tech Stack:** Go stdlib HTTP, in-memory repositories, current ticket collaboration service, current RAG service, table-driven tests

### Task 1: 定义 ticket assist 输入输出与依赖边界

**Files:**
- Modify: `internal/module/ai/model.go`
- Modify: `internal/module/ai/service.go`
- Test: `internal/module/ai/service_test.go`

**Step 1: Write the failing test**

- 为 ticket assist 新增输入输出模型测试与服务用例骨架。
- 先写断言：返回结构包含 `category_suggestion`、`summary`、`reply_draft`、`citations`、`recorded_comment_id`。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/module/ai -run 'TestGenerateTicketAssist'`
Expected: FAIL because models and service method do not exist yet.

**Step 3: Write minimal implementation**

- 在 `model.go` 中定义 `GenerateTicketAssistInput`、`TicketAssistResult`。
- 在 `service.go` 中定义 ticket assist 依赖接口。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/module/ai -run 'TestGenerateTicketAssist'`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/module/ai/model.go internal/module/ai/service.go internal/module/ai/service_test.go
git commit -m "feat: add ticket assist models and boundaries"
```

### Task 2: 实现 AI 分类、摘要与回复草稿生成

**Files:**
- Modify: `internal/module/ai/service.go`
- Test: `internal/module/ai/service_test.go`

**Step 1: Write the failing test**

- 补充测试覆盖：
  - 处理方可触发 ticket assist
  - 终端用户不能触发
  - 返回分类建议、摘要与回复草稿
  - 当知识检索降级时，摘要仍可生成，但草稿带降级提示

**Step 2: Run test to verify it fails**

Run: `go test ./internal/module/ai -run 'Test(TicketAssist|EndUserCannotGenerateTicketAssist|TicketAssistDegraded)'`
Expected: FAIL because service behavior does not exist yet.

**Step 3: Write minimal implementation**

- 从 ticket 读取工单详情和时间线。
- 复用现有知识回答能力生成知识增强结果。
- 基于工单上下文 + 检索结果生成分类建议、摘要和回复草稿。
- 在建议型输出与降级边界处补中文注释。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/module/ai -run 'Test(TicketAssist|EndUserCannotGenerateTicketAssist|TicketAssistDegraded)'`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/module/ai/service.go internal/module/ai/service_test.go
git commit -m "feat: implement ticket assist generation flow"
```

### Task 3: 记录 AI 执行轨迹到工单内部时间线

**Files:**
- Modify: `internal/module/ai/service.go`
- Modify: `internal/module/ticket/service.go` (only if interface adaptation is needed)
- Test: `internal/module/ai/service_test.go`
- Test: `internal/module/ticket/collaboration_test.go`

**Step 1: Write the failing test**

- 断言成功生成 AI 辅助结果后，会写入一条 `internal_note` 类型的执行记录。
- 断言执行记录对终端用户不可见。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/module/ai ./internal/module/ticket -run 'Test(TicketAssistRecordsInternalNote|EndUserTimelineHidesInternalNotes)'`
Expected: FAIL because execution record is not written yet.

**Step 3: Write minimal implementation**

- 将本次 AI 输出格式化为结构化内部备注。
- 把记录写入 ticket 时间线，并返回 `recorded_comment_id`。
- 在“为何先复用 internal note”处补中文注释。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/module/ai ./internal/module/ticket -run 'Test(TicketAssistRecordsInternalNote|EndUserTimelineHidesInternalNotes)'`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/module/ai/service.go internal/module/ai/service_test.go internal/module/ticket/service.go internal/module/ticket/collaboration_test.go
git commit -m "feat: record ai ticket assist trace in timeline"
```

### Task 4: 暴露 HTTP 接口并完成应用装配

**Files:**
- Modify: `internal/platform/http/ai_handler.go`
- Modify: `internal/platform/http/ai_handler_test.go`
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_rag_test.go` or Create: `internal/app/app_ticket_ai_test.go`

**Step 1: Write the failing test**

- 新增 HTTP 测试：处理方请求成功、终端用户被拒绝、响应含 `recorded_comment_id`。
- 新增 app 测试：登录 -> 创建工单 -> 调用 AI assist -> 查询时间线看到内部备注。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/http ./internal/app -run 'Test(AITicketAssist|NewWiresAITicketAssist)'`
Expected: FAIL because routes and wiring do not exist yet.

**Step 3: Write minimal implementation**

- 新增接口：`POST /api/v1/ai/tickets/{id}/assist`
- 在 app 层装配新的 ticket assist 服务。
- 保持返回结构与现有 AI / ticket 风格一致。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/platform/http ./internal/app -run 'Test(AITicketAssist|NewWiresAITicketAssist)'`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/platform/http/ai_handler.go internal/platform/http/ai_handler_test.go internal/app/app.go internal/app/app_ticket_ai_test.go
git commit -m "feat: expose ai ticket assist endpoint"
```

### Task 5: 回填文档与全量验证

**Files:**
- Modify: `README.md`
- Modify: `docs/workitems/issue-008-AI分类摘要与回复草稿/04-验证记录.md`
- Modify: `docs/workitems/issue-008-AI分类摘要与回复草稿/05-工作日志.md`
- Modify: `docs/workitems/issue-008-AI分类摘要与回复草稿/06-复盘总结.md`

**Step 1: Write the failing test**

- 本任务不新增功能测试，以全量回归作为验收依据。

**Step 2: Run test to verify baseline**

Run: `go test ./...`
Expected: PASS

**Step 3: Write minimal implementation**

- 更新 README 中 AI 工单辅助能力说明与调用示例。
- 回填验证记录、工作日志和复盘总结。

**Step 4: Run test to verify it passes**

Run: `go test ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add README.md docs/workitems/issue-008-AI分类摘要与回复草稿/04-验证记录.md docs/workitems/issue-008-AI分类摘要与回复草稿/05-工作日志.md docs/workitems/issue-008-AI分类摘要与回复草稿/06-复盘总结.md
git commit -m "docs: finalize issue 8 verification and notes"
```
