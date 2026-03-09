# 答案反馈回流与升级规则 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 `ZLD SupportPilot` 增加 AI 结果反馈与升级闭环，让一次统一受理结果可追踪、可反馈、可自动升级，并对后台暴露基础反馈统计。

**Architecture:** 在 `internal/module/ai` 中补齐统一受理主记录与反馈记录的领域模型、repository 和 service；统一受理接口在返回结果时写入 intake 主记录并返回 `intake_id`；反馈接口基于 intake 记录提交 `resolved / unresolved / inaccurate`，其中 `unresolved` 复用 `ticket.Service.CreateTicket` 执行确定性升级。HTTP 层新增 feedback / stats API，app 层负责 memory / postgres 双装配与 schema 落地。

**Tech Stack:** Go、模块化单体、memory / postgres 双持久化、标准库 `net/http`、现有身份权限体系、现有测试体系

### Task 1: 为 intake / feedback service 补 RED 测试

**Files:**
- Modify: `internal/module/ai/model.go`
- Modify: `internal/module/ai/service.go`
- Modify: `internal/module/ai/service_test.go`
- Create: `internal/module/ai/feedback_test.go`

**Step 1: Write the failing test**

新增测试覆盖：
- 统一受理返回 `intake_id`
- `resolved` 反馈只记录，不建单
- `unresolved` 反馈在无工单时自动建单
- 重复 `unresolved` 反馈不重复建单
- `inaccurate` 反馈只计入统计
- 无权限查看 stats 时返回错误

**Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/module/ai`
Expected: FAIL，提示 intake / feedback 模型、repository 或 service 方法缺失

**Step 3: Write minimal implementation**

在 `ai` 模块中新增 intake / feedback 领域模型、repository 接口、memory 实现和 service 方法，最小实现自动升级规则。

**Step 4: Run test to verify it passes**

Run: `go test -count=1 ./internal/module/ai`
Expected: PASS

### Task 2: 为 postgres repository 与 schema 补 RED 测试

**Files:**
- Create: `internal/module/ai/repository.go`
- Create: `internal/module/ai/memory_repository.go`
- Create: `internal/module/ai/postgres_repository.go`
- Create: `internal/module/ai/postgres_repository_test.go`
- Modify: `internal/platform/persistence/schema.sql`

**Step 1: Write the failing test**

覆盖：
- intake 记录可保存并读取
- feedback 可按 `(intake_id, actor_id)` 更新
- 统计查询能返回正确计数

**Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/module/ai -run 'TestPostgres'`
Expected: FAIL

**Step 3: Write minimal implementation**

补齐 memory / postgres repository 和 schema 表结构。

**Step 4: Run test to verify it passes**

Run: `go test -count=1 ./internal/module/ai -run 'TestPostgres'`
Expected: PASS

### Task 3: 为 feedback / stats HTTP API 补 RED 测试

**Files:**
- Modify: `internal/platform/http/ai_handler.go`
- Modify: `internal/platform/http/ai_handler_test.go`

**Step 1: Write the failing test**

覆盖：
- 提交 feedback 成功
- `unresolved` 返回建单结果
- stats 查询成功
- 未登录访问失败
- 终端用户访问 stats 被拒绝
- 非法请求体失败

**Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/platform/http -run 'TestAI(Feedback|FeedbackStats|Intake)'`
Expected: FAIL

**Step 3: Write minimal implementation**

新增 request / response 结构、handler 方法和路由注册，并把 `intake_id` 写入统一受理响应。

**Step 4: Run test to verify it passes**

Run: `go test -count=1 ./internal/platform/http -run 'TestAI(Feedback|FeedbackStats|Intake)'`
Expected: PASS

### Task 4: 为 app 装配与集成验证补实现

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`
- Modify: `README.md`
- Modify: `docs/workitems/issue-026-答案反馈回流与升级规则/*.md`

**Step 1: Write the failing test**

增加 app 集成测试，验证：
- 统一受理返回 `intake_id`
- 反馈接口可直接升级工单
- stats 接口在 agent / admin 角色下可用

**Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/app -run 'TestNewWiresAI(Feedback|Intake)'`
Expected: FAIL

**Step 3: Write minimal implementation**

接入新 repository / service / handler，更新 README 与工作文档。

**Step 4: Run final verification**

Run: `go test -count=1 ./...`
Expected: PASS
