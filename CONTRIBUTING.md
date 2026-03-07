# Contributing Guide

## Collaboration Model

`ZLD SupportPilot` is maintained as a production-style portfolio project. The repository uses lightweight team conventions so work can be demonstrated in a realistic GitHub flow.

## Workflow

1. Start from an open GitHub issue.
2. Create a branch with the `codex/*` prefix.
3. Keep each PR focused on one issue or one tightly related slice.
4. Run local validation before opening or updating a PR.
5. Merge only after the PR description clearly explains scope, risk, and validation.

## Branch Naming

- `codex/bootstrap-repo`
- `codex/auth-rbac-foundation`
- `codex/ticket-workflow`
- `codex/rag-answer-api`

## Pull Requests

- Link the issue in the PR body with `Closes #<number>`
- Describe `What`, `Why`, `How`, `Risk`, and `Validation`
- Keep PRs small enough to review in one sitting

## Milestones

- `M1 - Foundation`: bootstrap, environment, auth foundation
- `M2 - Ticket & Knowledge`: ticket workflow, document ingestion, retrieval preparation
- `M3 - AI & Ops`: RAG, AI assistance, observability, release polish

## Verification

For Go code changes, run:

```bash
gofmt -w ./cmd ./internal ./pkg
go test ./...
```

For documentation-only changes, verify links and referenced issue numbers are still correct.

