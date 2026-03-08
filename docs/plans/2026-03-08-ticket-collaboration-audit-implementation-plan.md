# Ticket Collaboration Audit Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为工单主流程补齐评论、内部备注、指派/状态审计事件和统一时间线接口。

**Architecture:** 在 `internal/module/ticket` 中新增评论和审计事件模型，服务层负责统一记录和读取；HTTP 层只负责受保护的评论写入与时间线查询；权限与可见性判断全部收敛在服务层，避免不同入口产生行为漂移。

**Tech Stack:** Go stdlib HTTP, in-memory repositories, table-driven tests, current JWT auth, current ticket status machine

### Task 1: 扩展 ticket 领域模型与仓储接口

**Files:**
- Modify: `internal/module/ticket/model.go`
- Modify: `internal/module/ticket/repository.go`
- Modify: `internal/module/ticket/memory_repository.go`
- Test: `internal/module/ticket/service_test.go`

**Step 1: Write the failing test**

- 在 `service_test.go` 中新增断言：评论、内部备注、审计事件和时间线模型存在并按预期工作。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/module/ticket -run 'Test(AddTicketComment|ListTicketTimeline)'`
Expected: FAIL because collaboration models and repositories do not exist yet.

**Step 3: Write minimal implementation**

- 在 `model.go` 中定义 `TicketComment`、`TicketCommentType`、`TicketAuditEvent`、`TicketTimelineItem`。
- 在 `repository.go` 中定义评论和审计事件仓储接口。
- 在 `memory_repository.go` 中增加线程安全的内存适配。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/module/ticket -run 'Test(AddTicketComment|ListTicketTimeline)'`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/module/ticket/model.go internal/module/ticket/repository.go internal/module/ticket/memory_repository.go internal/module/ticket/service_test.go
git commit -m "feat: add ticket collaboration models and repositories"
```

### Task 2: 实现评论、内部备注和自动审计事件

**Files:**
- Modify: `internal/module/ticket/service.go`
- Test: `internal/module/ticket/service_test.go`

**Step 1: Write the failing test**

- 补充测试覆盖：
  - 客服可写公开评论和内部备注
  - 终端用户只能写公开评论
  - 指派和状态流转后自动生成审计事件
  - 时间线按时间排序输出，且终端用户看不到内部备注

**Step 2: Run test to verify it fails**

Run: `go test ./internal/module/ticket -run 'Test(AddTicketComment|InternalNote|AssignTicketRecordsAudit|TransitionTicketStatusRecordsAudit|ListTicketTimeline)'`
Expected: FAIL because service methods do not exist yet.

**Step 3: Write minimal implementation**

- 在 `service.go` 中实现 `AddTicketComment`、`ListTicketTimeline`。
- 在 `AssignTicket` 与 `TransitionTicketStatus` 中自动写入审计事件。
- 在内部备注可见性、审计记录写入点和时间线输出处补中文注释。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/module/ticket -run 'Test(AddTicketComment|InternalNote|AssignTicketRecordsAudit|TransitionTicketStatusRecordsAudit|ListTicketTimeline)'`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/module/ticket/service.go internal/module/ticket/service_test.go
git commit -m "feat: add ticket collaboration service and audit timeline"
```

### Task 3: 暴露 HTTP 接口并完成应用验证

**Files:**
- Modify: `internal/platform/http/ticket_handler.go`
- Modify: `internal/platform/http/ticket_handler_test.go`
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`

**Step 1: Write the failing test**

- 新增 HTTP 测试：评论写入成功、终端用户不能写内部备注、时间线查询返回事件。
- 新增 app 测试：登录后创建工单、追加评论、查询时间线。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/http -run TestTicket`
Expected: FAIL because collaboration routes and handlers do not exist yet.

**Step 3: Write minimal implementation**

- 在 handler 中新增 `POST /api/v1/tickets/{id}/comments` 和 `GET /api/v1/tickets/{id}/timeline`。
- 在 app 层继续装配扩展后的 ticket service。
- 保持返回结构与现有 ticket 风格一致。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/platform/http -run TestTicket`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/platform/http/ticket_handler.go internal/platform/http/ticket_handler_test.go internal/app/app.go internal/app/app_test.go
git commit -m "feat: expose ticket collaboration and audit routes"
```

### Task 4: 回填文档与全量验证

**Files:**
- Modify: `README.md`
- Modify: `docs/workitems/issue-004-工单协作与审计事件/04-验证记录.md`
- Modify: `docs/workitems/issue-004-工单协作与审计事件/05-工作日志.md`
- Modify: `docs/workitems/issue-004-工单协作与审计事件/06-复盘总结.md`

**Step 1: Write the failing test**

- 本任务不新增功能测试，以全量回归为验收依据。

**Step 2: Run test to verify baseline**

Run: `go test ./...`
Expected: PASS

**Step 3: Write minimal implementation**

- 更新 README 中的 ticket 协作能力说明。
- 回填验证记录、工作日志和复盘总结。

**Step 4: Run test to verify it passes**

Run: `go test ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add README.md docs/workitems/issue-004-工单协作与审计事件/04-验证记录.md docs/workitems/issue-004-工单协作与审计事件/05-工作日志.md docs/workitems/issue-004-工单协作与审计事件/06-复盘总结.md
git commit -m "docs: finalize issue 4 verification and notes"
```
