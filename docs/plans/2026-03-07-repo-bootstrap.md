# Repository Bootstrap Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create the initial `ZLD SupportPilot` repository with a runnable Go API skeleton, project docs, and GitHub workflow scaffolding.

**Architecture:** Start with a minimal module-oriented monolith structure that runs locally without third-party dependencies. Add documentation and GitHub templates first so later feature work can be tracked through issues and PRs.

**Tech Stack:** Go stdlib, GitHub Issues/PRs, Docker Compose placeholders

### Task 1: Bootstrap repository structure

**Files:**
- Create: `README.md`
- Create: `.gitignore`
- Create: `go.mod`
- Create: `Makefile`

**Step 1:** Create the project identity and top-level files.

**Step 2:** Add a minimal Go module and local run target.

**Step 3:** Document project scope, architecture, and bootstrap status.

**Step 4:** Run `go test ./...` to verify the empty skeleton builds.

### Task 2: Add runnable API entry

**Files:**
- Create: `cmd/api/main.go`
- Create: `internal/app/app.go`
- Create: `internal/platform/config/config.go`
- Create: `internal/platform/http/router.go`

**Step 1:** Implement config loading with safe defaults.

**Step 2:** Wire a standard library HTTP server.

**Step 3:** Add `GET /healthz` for smoke testing.

**Step 4:** Run `go test ./...` and `go run ./cmd/api`.

### Task 3: Add collaboration scaffolding

**Files:**
- Create: `.github/ISSUE_TEMPLATE/feature.yml`
- Create: `.github/ISSUE_TEMPLATE/task.yml`
- Create: `.github/ISSUE_TEMPLATE/config.yml`
- Create: `.github/PULL_REQUEST_TEMPLATE.md`
- Create: `docs/architecture/overview.md`

**Step 1:** Add issue templates for delivery tracking.

**Step 2:** Add a PR template with validation checklist.

**Step 3:** Capture architecture intent for future contributors.

**Step 4:** Push the bootstrap branch and open the first PR.

