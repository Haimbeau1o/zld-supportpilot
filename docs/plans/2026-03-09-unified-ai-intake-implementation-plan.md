# 统一 AI 受理入口 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 `ZLD SupportPilot` 增加单一 AI 受理入口，让用户只提交一次问题即可得到直接回答或自动升级工单的结果。

**Architecture:** 在 `internal/module/ai` 内新增统一受理模型与 service 方法，复用现有 `AskKnowledgeQuestion` 做置信度判断，并在低置信度场景下联动 `ticket.Service.CreateTicket` 自动建单。HTTP 层新增单一路由 `POST /api/v1/ai/intakes`，保持现有问答与 ticket assist 接口不回归。

**Tech Stack:** Go、现有模块化单体分层、AI service 复用、ticket service 复用、HTTP routeRuntime、现有测试体系

### Task 1: 为统一受理 service 补 RED 测试

**Files:**
- Modify: `internal/module/ai/model.go`
- Modify: `internal/module/ai/service_test.go`
- Modify: `internal/module/ai/ticket_assist_test.go`

**Step 1: Write the failing test**

新增测试覆盖：
- 高置信度时直接返回 `answered`
- 低置信度时自动创建工单并返回 `ticket_id`
- 问题为空时返回输入错误

**Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/module/ai`
Expected: FAIL，提示统一受理模型或方法缺失

**Step 3: Write minimal implementation**

在 `ai` 模块中新增统一受理输入输出模型与 service 方法，最小复用现有问答结果与建单能力。

**Step 4: Run test to verify it passes**

Run: `go test -count=1 ./internal/module/ai`
Expected: PASS

### Task 2: 为统一受理 HTTP API 补 RED 测试

**Files:**
- Modify: `internal/platform/http/ai_handler.go`
- Modify: `internal/platform/http/ai_handler_test.go`

**Step 1: Write the failing test**

覆盖：
- 成功直接回答
- 成功自动建单
- 未登录访问失败
- 请求体非法失败

**Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/platform/http -run 'TestAIIntake'`
Expected: FAIL

**Step 3: Write minimal implementation**

新增统一受理请求/响应结构、handler 方法和路由注册。

**Step 4: Run test to verify it passes**

Run: `go test -count=1 ./internal/platform/http -run 'TestAIIntake'`
Expected: PASS

### Task 3: 为应用装配补集成验证

**Files:**
- Modify: `internal/app/app_test.go`
- Modify: `README.md`
- Modify: `docs/workitems/issue-025-统一AI受理入口/*.md`

**Step 1: Write the failing test**

增加 `app` 层集成测试，验证新路由在真实装配下可直接回答或返回已升级工单。

**Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/app -run 'TestNewWiresAIIntakeRoute'`
Expected: FAIL

**Step 3: Write minimal implementation**

接入新 handler，更新 README 与工作文档。

**Step 4: Run final verification**

Run: `go test -count=1 ./...`
Expected: PASS
