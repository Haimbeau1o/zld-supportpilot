# 可观测性、限流与发布整理 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 `zld-supportpilot` 增加最小可用的请求关联、结构化访问日志、HTTP 指标、关键路径限流和演示说明。

**Architecture:** 保持 Go 标准库优先：以统一 HTTP handler 为入口注入请求 ID，在路由注册点按路径挂载观测 / 限流中间件，并以进程内指标快照和令牌桶实现基础工程化能力。当前阶段明确不引入外部 APM 或分布式限流设施，避免超出单进程 demo 的合理复杂度。

**Tech Stack:** Go、`net/http`、`log`、`encoding/json`、`sync`、`time`、现有内存仓储与测试框架

### Task 1: 补齐配置层限流参数

**Files:**
- Modify: `internal/platform/config/config.go`
- Modify: `internal/platform/config/config_test.go`

**Step 1: Write the failing test**

在 `internal/platform/config/config_test.go` 中新增测试，覆盖：
- 默认值：`HTTPRateLimitEnabled=true`、`HTTPRateLimitRPS=5`、`HTTPRateLimitBurst=10`
- 环境变量覆盖：布尔、浮点和整数均可被正确解析
- 非法值回退默认值

**Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/platform/config`
Expected: FAIL，提示新字段不存在或读取逻辑缺失

**Step 3: Write minimal implementation**

在 `config.go` 中新增字段与读取函数：
- `HTTPRateLimitEnabled bool`
- `HTTPRateLimitRPS float64`
- `HTTPRateLimitBurst int`

补充 `getenvBool`、`getenvFloat64`、`getenvInt` 一类的最小工具函数。

**Step 4: Run test to verify it passes**

Run: `go test -count=1 ./internal/platform/config`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/platform/config/config.go internal/platform/config/config_test.go
git commit -m "feat: add http rate limit config"
```

### Task 2: 为 HTTP 层建立请求关联与指标采集

**Files:**
- Modify: `internal/platform/http/context.go`
- Create: `internal/platform/http/observability_middleware.go`
- Create: `internal/platform/http/observability_middleware_test.go`
- Modify: `internal/platform/http/router.go`
- Modify: `internal/platform/http/auth_handler_test.go`
- Modify: `internal/platform/http/ai_handler_test.go`
- Modify: `internal/platform/http/knowledge_handler_test.go`
- Modify: `internal/platform/http/knowledge_processing_handler_test.go`
- Modify: `internal/platform/http/ticket_handler_test.go`

**Step 1: Write the failing test**

在 `observability_middleware_test.go` 中先写失败测试，覆盖：
- 已注册路由响应头会返回 `X-Request-ID`
- 如果请求头已经带 `X-Request-ID`，响应会复用该值
- 访问日志包含 `request_id`、`route`、`status`、`duration_ms`
- 指标接口能返回请求总数和状态分布

**Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/platform/http`
Expected: FAIL，提示缺少 request id / middleware / metrics handler

**Step 3: Write minimal implementation**

实现：
- 请求 ID 上下文读写函数
- 请求 ID 注入中间件
- 响应状态记录器
- 访问日志中间件
- 进程内 HTTP 指标聚合器与 `GET /debug/metrics/http`
- `NewMux*` 统一返回 `http.Handler`

**Step 4: Run targeted tests**

Run: `go test -count=1 ./internal/platform/http -run 'Test(Observability|Middleware|AI|Auth|Knowledge|Ticket)'`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/platform/http/context.go internal/platform/http/observability_middleware.go internal/platform/http/observability_middleware_test.go internal/platform/http/router.go internal/platform/http/*_test.go
git commit -m "feat: add request tracing and http metrics"
```

### Task 3: 为关键路径增加限流保护

**Files:**
- Create: `internal/platform/http/rate_limit_middleware.go`
- Create: `internal/platform/http/rate_limit_middleware_test.go`
- Modify: `internal/platform/http/router.go`
- Modify: `internal/platform/http/auth_handler.go`

**Step 1: Write the failing test**

在 `rate_limit_middleware_test.go` 中先写失败测试，覆盖：
- 限流允许首个请求通过
- 连续超限请求返回 `429`
- 响应包含 `Retry-After`
- 错误结构仍使用统一 JSON 形状
- 指标中能看到限流计数

**Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/platform/http -run 'TestRateLimit'`
Expected: FAIL，提示缺少限流中间件或预期 `429` 未触发

**Step 3: Write minimal implementation**

实现：
- 进程内令牌桶
- 身份优先 / 地址兜底的限流键提取
- `Retry-After` 计算
- 对选定路由挂载限流：`POST /api/v1/auth/login`、`POST /api/v1/knowledge/bases/{id}/documents`、`POST /api/v1/knowledge/documents/{id}/retry`、`POST /api/v1/ai/knowledge/bases/{id}/answers`

**Step 4: Run targeted tests**

Run: `go test -count=1 ./internal/platform/http -run 'TestRateLimit|TestKnowledge|TestAI|TestAuth'`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/platform/http/rate_limit_middleware.go internal/platform/http/rate_limit_middleware_test.go internal/platform/http/router.go internal/platform/http/auth_handler.go
git commit -m "feat: protect critical routes with rate limit"
```

### Task 4: 接入应用入口并补齐演示文档

**Files:**
- Modify: `internal/app/app.go`
- Modify: `README.md`
- Modify: `docs/workitems/issue-009-可观测性限流与发布整理/04-验证记录.md`
- Modify: `docs/workitems/issue-009-可观测性限流与发布整理/05-工作日志.md`
- Modify: `docs/workitems/issue-009-可观测性限流与发布整理/06-复盘总结.md`

**Step 1: Write the failing / missing verification target**

补一条集成验证：
- 应用启动后关键路由能返回请求 ID
- README 中能按步骤查看日志、指标并触发限流

**Step 2: Run test to verify current gap**

Run: `go test -count=1 ./internal/app`
Expected: 若入口未接入统一 handler，则相关集成检查无法覆盖

**Step 3: Write minimal implementation**

实现：
- `app.go` 使用统一装配后的 handler
- `README.md` 增加观测、限流、演示与发布说明
- 回填验证、日志与复盘文档

**Step 4: Run final verification**

Run:
- `go test -count=1 ./internal/app`
- `go test -count=1 ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/app/app.go README.md docs/workitems/issue-009-可观测性限流与发布整理/*.md
git commit -m "docs: finalize observability and release workflow"
```
