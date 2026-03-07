# Ticket Main Flow Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 `ZLD SupportPilot` 建立最小可用的工单主流程，在认证与租户边界之上支持创建、查询、指派和状态流转。

**Architecture:** 采用 interface-first 的 ticket 模块设计，在 `internal/module/ticket` 中定义工单领域模型、状态机、服务层和内存仓储；在 `internal/platform/http` 暴露受保护的工单 API，并复用当前 `identity` 模块提供的租户与权限上下文。当前阶段优先稳定主链路边界，不在本计划中引入评论、审计、数据库适配与 AI 能力。

**Tech Stack:** Go stdlib HTTP, in-memory repository, table-driven tests, current auth middleware

### Task 1: 定义工单领域模型与状态流转规则

**Files:**
- Create: `internal/module/ticket/model.go`
- Create: `internal/module/ticket/state_machine.go`
- Test: `internal/module/ticket/state_machine_test.go`

**Step 1: Write the failing test**

- 编写 `state_machine_test.go`，验证工单初始状态和允许 / 禁止的状态流转是否符合预期。
- 至少覆盖：`open -> in_progress`、`in_progress -> resolved`、`resolved -> closed` 成功，以及 `closed -> in_progress` 失败。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/module/ticket -run TestTicketStatusTransitions`
Expected: FAIL because ticket model and transition rules do not exist yet.

**Step 3: Write minimal implementation**

- 在 `model.go` 中定义 `Ticket`、`TicketStatus`、`TicketPriority`。
- 在 `state_machine.go` 中定义允许的状态流转规则。
- 在状态集合和流转规则处补中文注释，说明为什么当前阶段采用显式状态机。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/module/ticket -run TestTicketStatusTransitions`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/module/ticket/model.go internal/module/ticket/state_machine.go internal/module/ticket/state_machine_test.go
git commit -m "feat: add ticket models and status transitions"
```

### Task 2: 建立仓储接口、服务层与内存适配

**Files:**
- Create: `internal/module/ticket/repository.go`
- Create: `internal/module/ticket/memory_repository.go`
- Create: `internal/module/ticket/service.go`
- Test: `internal/module/ticket/service_test.go`

**Step 1: Write the failing test**

- 编写 `service_test.go`，覆盖：终端用户创建工单成功、终端用户只能查看自己的工单、客服可查看租户内工单、客服可指派工单、非法状态流转失败。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/module/ticket -run 'Test(CreateTicket|ListTickets|AssignTicket|TransitionTicketStatus)'`
Expected: FAIL because service and repository do not exist yet.

**Step 3: Write minimal implementation**

- 在 `repository.go` 中定义 `TicketRepository` 接口。
- 在 `memory_repository.go` 中实现线程安全的内存仓储。
- 在 `service.go` 中实现 `CreateTicket`、`GetTicket`、`ListTickets`、`AssignTicket`、`TransitionTicketStatus`。
- 在查询范围控制、跨租户保护和状态流转校验处补中文注释。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/module/ticket -run 'Test(CreateTicket|ListTickets|AssignTicket|TransitionTicketStatus)'`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/module/ticket/repository.go internal/module/ticket/memory_repository.go internal/module/ticket/service.go internal/module/ticket/service_test.go
git commit -m "feat: add ticket service and memory repository"
```

### Task 3: 暴露工单 HTTP 接口与受保护路由

**Files:**
- Create: `internal/platform/http/ticket_handler.go`
- Modify: `internal/platform/http/router.go`
- Test: `internal/platform/http/ticket_handler_test.go`

**Step 1: Write the failing test**

- 编写 HTTP 测试，覆盖：创建工单、查询自己的工单列表、客服查看租户工单详情、指派工单、状态流转、未带令牌访问被拒绝。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/http -run TestTicket`
Expected: FAIL because ticket handlers and routes do not exist yet.

**Step 3: Write minimal implementation**

- 在 `ticket_handler.go` 中实现 `create`、`list`、`detail`、`assign`、`transition status`。
- 在 `router.go` 中注册受保护的工单路由。
- 复用当前身份上下文，不重新发明认证逻辑。
- 在请求范围过滤、动作型接口设计与错误返回处补中文注释。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/platform/http -run TestTicket`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/platform/http/ticket_handler.go internal/platform/http/router.go internal/platform/http/ticket_handler_test.go
git commit -m "feat: add ticket handlers and protected routes"
```

### Task 4: 完成应用装配、文档更新与回归验证

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`
- Modify: `README.md`
- Modify: `docs/workitems/issue-003-工单主流程/04-验证记录.md`
- Modify: `docs/workitems/issue-003-工单主流程/05-工作日志.md`
- Modify: `docs/workitems/issue-003-工单主流程/06-复盘总结.md`

**Step 1: Write the failing test**

- 编写应用装配测试，验证在完整 app 启动后，已登录用户可以访问工单创建接口。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/app -run TestNewWiresTicketRoutes`
Expected: FAIL because app has not wired ticket dependencies yet.

**Step 3: Write minimal implementation**

- 在 `app.go` 中装配 ticket service。
- 更新 `app_test.go` 和 `README.md` 的运行与接口说明。
- 回填 issue #3 的验证记录、工作日志和复盘总结。

**Step 4: Run test to verify it passes**

Run: `go test ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/app/app.go internal/app/app_test.go README.md docs/workitems/issue-003-工单主流程/04-验证记录.md docs/workitems/issue-003-工单主流程/05-工作日志.md docs/workitems/issue-003-工单主流程/06-复盘总结.md
git commit -m "feat: wire ticket main flow and document verification"
```
