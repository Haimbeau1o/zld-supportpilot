# Auth / Tenant / RBAC Foundation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 `ZLD SupportPilot` 建立最小可用的认证、租户上下文与 RBAC 基础，并通过中文注释和测试固化核心设计意图。

**Architecture:** 采用 interface-first 的身份模块设计，在 `internal/module/identity` 内定义领域模型、权限映射、服务层与内存适配；在 `internal/platform/http` 暴露注册、登录和身份解析接口，并通过中间件把身份上下文注入后续模块。当前阶段优先建立稳定边界，不在本计划中引入数据库适配与高级 IAM 特性。

**Tech Stack:** Go stdlib HTTP, `github.com/golang-jwt/jwt/v5`, `crypto/sha256` or password hash utility, table-driven tests

### Task 1: 定义身份领域模型与权限映射

**Files:**
- Create: `internal/module/identity/model.go`
- Create: `internal/module/identity/permission.go`
- Test: `internal/module/identity/permission_test.go`

**Step 1: Write the failing test**

- 编写 `permission_test.go`，验证不同角色映射到的权限集合是否符合预期。
- 至少覆盖：`tenant_admin`、`agent`、`end_user` 三种角色。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/module/identity -run TestRolePermissions`
Expected: FAIL because permission mapping does not exist yet.

**Step 3: Write minimal implementation**

- 在 `model.go` 中定义 `User`、`Organization`、`Membership`、`Role`、`Permission`、`IdentityContext`。
- 在 `permission.go` 中定义角色到权限的映射表。
- 在映射表和默认角色设计处补中文注释，说明为什么当前阶段采用固定角色集合。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/module/identity -run TestRolePermissions`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/module/identity/model.go internal/module/identity/permission.go internal/module/identity/permission_test.go
git commit -m "feat: add identity models and permission mapping"
```

### Task 2: 建立仓储接口与服务层

**Files:**
- Create: `internal/module/identity/repository.go`
- Create: `internal/module/identity/service.go`
- Create: `internal/module/identity/memory_repository.go`
- Test: `internal/module/identity/service_test.go`

**Step 1: Write the failing test**

- 编写 `service_test.go`，覆盖：注册成功、重复邮箱注册失败、登录成功、密码错误失败。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/module/identity -run 'Test(Register|Login)'`
Expected: FAIL because service and repository do not exist yet.

**Step 3: Write minimal implementation**

- 在 `repository.go` 中定义 `UserRepository` 和 `MembershipRepository` 接口。
- 在 `memory_repository.go` 中实现线程安全的内存仓储。
- 在 `service.go` 中实现 `Register`、`Login`、`GetIdentityContext`。
- 在默认组织创建、默认角色分配、错误语义处补中文注释。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/module/identity -run 'Test(Register|Login)'`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/module/identity/repository.go internal/module/identity/service.go internal/module/identity/memory_repository.go internal/module/identity/service_test.go
git commit -m "feat: add identity service and memory repositories"
```

### Task 3: 增加密码与令牌能力

**Files:**
- Modify: `go.mod`
- Create: `internal/module/identity/password.go`
- Create: `internal/module/identity/token.go`
- Test: `internal/module/identity/password_test.go`
- Test: `internal/module/identity/token_test.go`

**Step 1: Write the failing test**

- 为密码校验和令牌签发 / 解析编写失败测试。
- 至少覆盖：密码哈希后可验证、错误密码不可通过、令牌可解析、篡改令牌失败。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/module/identity -run 'Test(Password|Token)'`
Expected: FAIL because token and password helpers do not exist yet.

**Step 3: Write minimal implementation**

- 在 `password.go` 中实现密码摘要和验证工具。
- 在 `token.go` 中实现 JWT 签发和解析。
- 在令牌 Claims、签名密钥使用、失效时间判断处补中文注释。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/module/identity -run 'Test(Password|Token)'`
Expected: PASS

**Step 5: Commit**

```bash
git add go.mod internal/module/identity/password.go internal/module/identity/token.go internal/module/identity/password_test.go internal/module/identity/token_test.go
git commit -m "feat: add password and token utilities"
```

### Task 4: 暴露认证接口与鉴权中间件

**Files:**
- Create: `internal/platform/http/auth_handler.go`
- Create: `internal/platform/http/auth_middleware.go`
- Create: `internal/platform/http/context.go`
- Modify: `internal/platform/http/router.go`
- Test: `internal/platform/http/auth_handler_test.go`
- Test: `internal/platform/http/auth_middleware_test.go`

**Step 1: Write the failing test**

- 编写 HTTP 测试，覆盖：注册、登录、获取当前身份、未带令牌访问受保护路由被拒绝。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/http -run 'Test(Auth|Middleware)'`
Expected: FAIL because handlers and middleware do not exist yet.

**Step 3: Write minimal implementation**

- 在 `auth_handler.go` 中实现 `register`、`login`、`me`。
- 在 `auth_middleware.go` 中解析 Authorization Header 并注入 `IdentityContext`。
- 在 `context.go` 中定义上下文读写辅助函数。
- 在 `router.go` 中注册公开路由和受保护路由。
- 在身份上下文注入与权限错误返回处补中文注释。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/platform/http -run 'Test(Auth|Middleware)'`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/platform/http/auth_handler.go internal/platform/http/auth_middleware.go internal/platform/http/context.go internal/platform/http/router.go internal/platform/http/auth_handler_test.go internal/platform/http/auth_middleware_test.go
git commit -m "feat: add auth handlers and identity middleware"
```

### Task 5: 装配配置、更新文档并做回归验证

**Files:**
- Modify: `internal/platform/config/config.go`
- Modify: `internal/app/app.go`
- Modify: `.env.example`
- Modify: `README.md`
- Modify: `docs/workitems/issue-002-认证-租户与RBAC基础/04-验证记录.md`
- Modify: `docs/workitems/issue-002-认证-租户与RBAC基础/05-工作日志.md`
- Modify: `docs/workitems/issue-002-认证-租户与RBAC基础/06-复盘总结.md`

**Step 1: Write the failing test**

- 编写配置加载和应用装配测试，验证认证配置缺失时采用默认值或给出明确错误。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/app ./internal/platform/config`
Expected: FAIL because auth config wiring is incomplete.

**Step 3: Write minimal implementation**

- 在 `config.go` 中加入 `AUTH_SIGNING_KEY`、`AUTH_TOKEN_TTL_SECONDS` 等配置项。
- 在 `app.go` 中装配 identity service 和 HTTP handlers。
- 更新 `.env.example` 和 `README.md` 的运行说明。
- 回填 issue #2 的验证记录、工作日志和复盘结论。

**Step 4: Run test to verify it passes**

Run: `go test ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/platform/config/config.go internal/app/app.go .env.example README.md docs/workitems/issue-002-认证-租户与RBAC基础/04-验证记录.md docs/workitems/issue-002-认证-租户与RBAC基础/05-工作日志.md docs/workitems/issue-002-认证-租户与RBAC基础/06-复盘总结.md
git commit -m "feat: wire auth foundation and document verification"
```
