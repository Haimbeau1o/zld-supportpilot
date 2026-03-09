# 持久化底座升级 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 `ZLD SupportPilot` 增加可切换的 PostgreSQL 持久化与文件系统文档存储底座，使核心业务数据在重启后可保留。

**Architecture:** 保持现有 service 只依赖 repository 接口，在 `internal/platform/persistence` 提供 PostgreSQL 连接与幂等 schema 初始化，在各 module 内新增 PostgreSQL repository adapter，并在 `app.New()` 里根据配置切换 memory / postgres 模式。文档原始内容先采用文件系统存储，确保范围可控且本地可验证。

**Tech Stack:** Go、`database/sql`、PostgreSQL（pgx stdlib 驱动）、文件系统存储、现有模块化单体分层

### Task 1: 补齐配置与启动模式测试

**Files:**
- Modify: `internal/platform/config/config.go`
- Modify: `internal/platform/config/config_test.go`
- Modify: `internal/app/app_test.go`

**Step 1: Write the failing test**

新增测试覆盖：
- 默认 `APP_PERSISTENCE_MODE=memory`
- 环境变量可覆盖 `APP_PERSISTENCE_MODE`、`POSTGRES_DSN`、`DOCUMENT_STORAGE_MODE`、`DOCUMENT_STORAGE_ROOT`
- postgres 模式缺失 DSN 时 `app.New()` 返回错误

**Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/platform/config ./internal/app`
Expected: FAIL，提示新配置字段或错误分支缺失

**Step 3: Write minimal implementation**

在 `config.Config` 中增加持久化相关字段，并在 `app.New()` 中增加模式校验。

**Step 4: Run test to verify it passes**

Run: `go test -count=1 ./internal/platform/config ./internal/app`
Expected: PASS

### Task 2: 增加文件系统对象存储

**Files:**
- Create: `internal/module/knowledge/file_storage.go`
- Create: `internal/module/knowledge/file_storage_test.go`

**Step 1: Write the failing test**

覆盖：
- `Save` 后可 `Load`
- 自动创建目录
- 不存在对象返回 `ErrKnowledgeObjectNotFound`

**Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/module/knowledge -run 'TestFileSystemObjectStorage'`
Expected: FAIL

**Step 3: Write minimal implementation**

实现基于根目录的文件系统对象存储，使用 `StorageKey` 作为相对路径。

**Step 4: Run test to verify it passes**

Run: `go test -count=1 ./internal/module/knowledge -run 'TestFileSystemObjectStorage'`
Expected: PASS

### Task 3: 增加 PostgreSQL 平台层与 schema 初始化

**Files:**
- Create: `internal/platform/persistence/postgres.go`
- Create: `internal/platform/persistence/schema.sql`

**Step 1: Write the failing test**

为 schema 初始化与 postgres 打开流程补 smoke test，至少覆盖空 DSN / 成功配置路径。

**Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/app`
Expected: FAIL 或缺少实现

**Step 3: Write minimal implementation**

实现 PostgreSQL 打开、ping、schema 初始化。

**Step 4: Run targeted tests**

Run: `go test -count=1 ./internal/app`
Expected: PASS

### Task 4: 增加 module PostgreSQL repository

**Files:**
- Create: `internal/module/identity/postgres_repository.go`
- Create: `internal/module/identity/postgres_repository_test.go`
- Create: `internal/module/ticket/postgres_repository.go`
- Create: `internal/module/ticket/postgres_repository_test.go`
- Create: `internal/module/knowledge/postgres_repository.go`
- Create: `internal/module/knowledge/postgres_repository_test.go`

**Step 1: Write the failing test**

以 repository 为单位写失败测试，覆盖 save / find / list / replace 的最小契约。

**Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/module/identity ./internal/module/ticket ./internal/module/knowledge`
Expected: FAIL

**Step 3: Write minimal implementation**

实现各模块 PostgreSQL repository，保持当前 service 契约不变。

**Step 4: Run test to verify it passes**

Run: `go test -count=1 ./internal/module/identity ./internal/module/ticket ./internal/module/knowledge`
Expected: PASS

### Task 5: 接入应用装配并更新文档

**Files:**
- Modify: `internal/app/app.go`
- Modify: `.env.example`
- Modify: `README.md`
- Modify: `docs/workitems/issue-024-持久化底座升级/*.md`

**Step 1: Write the failing verification target**

补一条装配测试：memory 模式不回归，postgres 模式缺配置时报错。

**Step 2: Run test to verify current gap**

Run: `go test -count=1 ./internal/app`
Expected: 若装配未完成则 FAIL

**Step 3: Write minimal implementation**

在 `app.New()` 中根据配置实例化 memory / postgres / filesystem adapter，补 `.env.example` 和 README。

**Step 4: Run final verification**

Run: `go test -count=1 ./...`
Expected: PASS
