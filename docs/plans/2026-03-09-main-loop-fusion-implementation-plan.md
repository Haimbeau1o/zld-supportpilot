# Main Loop Fusion Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 打通“统一 AI 受理 -> 未解决反馈 -> 工单处理 -> 知识候选 -> 审核入库 -> 下次问答命中”的主闭环，并把它沉淀为可复现的测试与演示脚本。

**Architecture:** 以 `internal/app` 的端到端测试作为主验收入口，优先复用 `#25/#26/#27` 已有能力，只在 RED 暴露真实缺口时补最小胶水代码。功能收口后新增一份可直接执行的 demo 脚本，并同步 README 和工作包文档。

**Tech Stack:** Go、`net/http`、`httptest`、现有 memory / postgres 双模式装配、shell demo script、GitHub Issue 工作包文档。

### Task 1: 写主闭环 RED 端到端测试

**Files:**
- Modify: `internal/app/app_test.go`
- Reference: `internal/app/app_rag_test.go`
- Reference: `internal/platform/http/ai_handler_test.go`

**Step 1: Write the failing test**

在 `internal/app/app_test.go` 新增：

```go
func TestNewMainLoopFusionClosesFeedbackToKnowledgeCycle(t *testing.T) {
    // 1. 注册并登录
    // 2. 创建知识库
    // 3. 预置“泛化但不够具体”的知识文档
    // 4. 用户走 /api/v1/ai/intakes 提问
    // 5. 用户走 /api/v1/ai/intakes/{id}/feedback 提交 unresolved
    // 6. 处理人走 /api/v1/ai/tickets/{id}/assist 获取 AI 辅助
    // 7. 处理人将 ticket 流转到 resolved
    // 8. 创建 knowledge candidate 并 approve
    // 9. 等待文档 ready
    // 10. 再次提问并断言命中新知识
}
```

第一次问题建议使用：`VPN 691 错误怎么处理？`

第一次预置的旧知识建议使用：`VPN 相关问题请联系 IT 服务台。`

第二次沉淀的新知识内容建议使用：`处理步骤：检查账号状态，重置账号拨号权限，重新连接验证。`

**Step 2: Run test to verify it fails**

Run:

```bash
GOCACHE=/tmp/go-build go test -count=1 ./internal/app -run TestNewMainLoopFusionClosesFeedbackToKnowledgeCycle
```

Expected:

- FAIL
- 失败点应落在主闭环链路中的真实缺口，而不是 JSON 拼写或 helper 错误

**Step 3: Write minimal implementation**

优先只在 `internal/app/app_test.go` 增补 helper：

```go
func createGenericKnowledgeDocument(...)
func submitUnresolvedFeedback(...)
func requestTicketAssist(...)
func resolveTicketViaHTTP(...)
func createCandidateViaHTTP(...)
func approveCandidateViaHTTP(...)
```

如果 RED 暴露的是生产代码缺口，再进入 Task 2 补最小修复。

**Step 4: Run test to verify it passes**

Run:

```bash
GOCACHE=/tmp/go-build go test -count=1 ./internal/app -run TestNewMainLoopFusionClosesFeedbackToKnowledgeCycle
```

Expected:

- PASS
- 断言至少覆盖：`ticket_id`、AI assist 成功、candidate approved、document ready、second answer answered

**Step 5: Commit**

```bash
git add internal/app/app_test.go
git commit -m "test: 补充 #28 主闭环端到端 RED/GREEN"
```

### Task 2: 补最小融合胶水代码

**Files:**
- Potential Modify: `internal/module/ai/service.go`
- Potential Modify: `internal/platform/http/ai_handler.go`
- Potential Modify: `internal/platform/http/ticket_handler.go`
- Potential Modify: `internal/app/app.go`
- Test: `internal/app/app_test.go`
- Test: `internal/platform/http/ai_handler_test.go`

**Step 1: Write the failing targeted regression**

如果 Task 1 的 RED 指向某个具体模块，则为该模块补一个更小的回归测试。例如：

```go
func TestSubmitAnswerFeedbackReturnsLinkedTicketForUnresolvedFlow(t *testing.T) {}
func TestAITicketAssistCanBeUsedOnFeedbackCreatedTicket(t *testing.T) {}
func TestApproveCandidateMakesClosedLoopKnowledgeSearchable(t *testing.T) {}
```

**Step 2: Run test to verify it fails**

Run one targeted package command, for example:

```bash
GOCACHE=/tmp/go-build go test -count=1 ./internal/module/ai -run TestSubmitAnswerFeedbackReturnsLinkedTicketForUnresolvedFlow
```

Expected:

- FAIL with a real business gap

**Step 3: Write minimal implementation**

只补当前 RED 暴露的最小修复，例如：

```go
// 确保 unresolved feedback 返回的 ticket_id 可继续用于 ticket assist
feedback.TicketID = ticketID

// 或确保 app 装配中相关依赖没有遗漏
AIService: aiService,
```

不要新增新的业务模型，不要在融合 issue 中扩大范围。

**Step 4: Run test to verify it passes**

Run:

```bash
GOCACHE=/tmp/go-build go test -count=1 ./internal/module/ai ./internal/platform/http ./internal/app
```

Expected:

- PASS
- 主闭环测试仍然保持绿色

**Step 5: Commit**

```bash
git add internal/module/ai/service.go internal/platform/http/ai_handler.go internal/platform/http/ticket_handler.go internal/app/app.go internal/platform/http/ai_handler_test.go internal/app/app_test.go
git commit -m "fix: 补齐 #28 主闭环融合胶水"
```

### Task 3: 新增可复现 demo 脚本

**Files:**
- Create: `scripts/demo_issue_028_closed_loop.sh`
- Modify: `README.md`
- Test: manual curl run or shell dry run

**Step 1: Write the failing script stub**

创建脚本骨架：

```bash
#!/usr/bin/env bash
set -euo pipefail

BASE_URL=${BASE_URL:-http://localhost:8080}
EMAIL=${EMAIL:-alice@example.com}
PASSWORD=${PASSWORD:-secret123}
```

脚本步骤按真实链路组织：

1. 注册 / 登录
2. 创建知识库
3. 上传泛化旧知识
4. intake 提问
5. unresolved feedback
6. ticket assist
7. ticket resolved
8. candidate create
9. candidate approve
10. second answer

**Step 2: Run script to verify it fails safely**

Run:

```bash
bash scripts/demo_issue_028_closed_loop.sh
```

Expected:

- 如果本地服务未启动，应在最前面报清晰错误
- 不允许静默失败

**Step 3: Write minimal implementation**

补齐脚本中的：

- `curl` 调用
- `jq`/`sed` 解析逻辑（若仓库内已有更简单方式，优先复用）
- 对关键返回值做显式校验

示例校验：

```bash
[ -n "$TICKET_ID" ] || { echo "missing ticket id"; exit 1; }
[ -n "$CANDIDATE_ID" ] || { echo "missing candidate id"; exit 1; }
```

**Step 4: Run script to verify it passes**

Run:

```bash
bash scripts/demo_issue_028_closed_loop.sh
```

Expected:

- 输出完整链路成功日志
- 最终 second answer 命中新的知识内容

**Step 5: Commit**

```bash
git add scripts/demo_issue_028_closed_loop.sh README.md
git commit -m "docs: 增加 #28 主闭环演示脚本"
```

### Task 4: 回归验证与工作包收口

**Files:**
- Modify: `docs/workitems/issue-028-问答建单处理知识回流主闭环/04-验证记录.md`
- Modify: `docs/workitems/issue-028-问答建单处理知识回流主闭环/05-工作日志.md`
- Modify: `docs/workitems/issue-028-问答建单处理知识回流主闭环/06-复盘总结.md`
- Modify: `README.md`

**Step 1: Update verification record**

补充：

- RED / GREEN 命令
- 主闭环关键断言
- demo 脚本执行结果
- 全量回归结果

**Step 2: Run all verification commands**

Run:

```bash
GOCACHE=/tmp/go-build go test -count=1 ./internal/app
GOCACHE=/tmp/go-build go test -count=1 ./internal/platform/http
GOCACHE=/tmp/go-build go test -count=1 ./internal/module/ai
GOCACHE=/tmp/go-build go test -count=1 ./internal/module/knowledge
GOCACHE=/tmp/go-build go test -count=1 ./...
```

Expected:

- 全绿

**Step 3: Update work log and retrospective**

明确写出：

- 为什么 #28 必须串行做融合收口
- 为什么采用“测试先行 + demo 脚本”而不是再扩业务模型
- 哪些点证明了这是真实的内部 AI 应用闭环

**Step 4: Push branch and sync GitHub**

Run:

```bash
git push origin codex/issue-028-main-loop
gh issue comment 28 --body-file /tmp/issue28-progress.md
```

Expected:

- issue 中能看到当前分支、验证结果和下一步计划

**Step 5: Commit**

```bash
git add README.md docs/workitems/issue-028-问答建单处理知识回流主闭环

git commit -m "docs: 完成 #28 主闭环验证与收口"
```
