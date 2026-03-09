# #28 问答-建单-处理-知识回流主闭环设计

## 背景

`#25`、`#26`、`#27` 已分别完成：

- 统一 AI 受理入口
- 答案反馈回流与升级规则
- 已解决工单沉淀为知识候选并审核入库

但这些能力当前仍主要以“单能力已完成”的方式存在。`#28` 的任务不是继续新增模块，而是把它们联调成一条真实、可演示、可验证的企业内部服务台闭环。

## 闭环场景

推荐使用如下业务故事作为标准验收链路：

1. 用户第一次提问，系统基于已有泛化知识给出回答
2. 用户提交“未解决”反馈，系统创建或关联工单
3. 处理人进入工单，调用 AI 工单辅助生成摘要 / 回复草稿，并把建议写入时间线
4. 处理人完成人工排障，将工单推进到 `resolved`
5. 处理人从该工单整理出知识候选
6. 审核人审核通过，系统自动生成 markdown 文档并复用现有文档处理流水线完成切块与索引
7. 后续用户再次提问相似问题，AI 能命中新沉淀的知识内容并返回引用

## 设计原则

### 1. 复用已有能力，不新造第二套流程

- 工单来源继续走 `HandleUnifiedIntake` 和 `SubmitAnswerFeedback`
- AI 辅助继续走 `GenerateTicketAssist`
- 知识沉淀继续走 `CreateKnowledgeCandidateFromTicket` 和 `ReviewKnowledgeCandidate`
- 入库后继续走 `UploadDocument -> scheduleDocumentProcessing -> ProcessTask`

### 2. 融合 issue 以联调、验证和演示为主

`#28` 的重点不在新增很多业务规则，而在：

- 把跨模块状态流串起来
- 补端到端测试覆盖
- 补一份可复现的操作脚本 / 演示文档
- 明确角色边界和数据流

### 3. 最小生产代码改动

如果端到端 RED 测试暴露缺口，只补最小胶水代码：

- 状态字段对齐
- 响应结构补充
- app 装配补强
- 权限或错误映射修正

避免在融合 issue 里继续引入新领域模型，防止作用域失控。

## 预期落点

### 代码层

- `internal/app/app_test.go`
  - 新增主闭环端到端测试
- `internal/platform/http/ai_handler_test.go`
  - 如有必要，补 HTTP 级回归断言
- `internal/module/ai/service.go`
  - 仅在 RED 暴露缺口时做最小修正
- `internal/platform/http/ai_handler.go`
  - 仅在 RED 暴露错误映射或响应字段缺口时修正
- `scripts/demo_issue_028_closed_loop.sh`
  - 新增可复现演示脚本

### 文档层

- `README.md`
  - 增加主闭环演示说明
- `docs/workitems/issue-028-问答建单处理知识回流主闭环/`
  - 沉淀完整工作包

## 风险与应对

### 风险 1：第一次问答阶段无法稳定触发“已回答但仍未解决”

应对：在测试场景中预置“泛化但不够具体”的知识内容，让第一次答案可返回，但仍合理触发 unresolved feedback。

### 风险 2：融合链路过长，排错成本高

应对：优先从 `internal/app` 写单条端到端测试，再根据 RED 位置拆回模块修复。

### 风险 3：`#27` 尚未 merge 到 `main`

应对：当前先基于 `codex/ticket-knowledge-candidates` 创建 stacked worktree；等 PR #35 merge 后再 rebase 到最新 `main`。

## 推荐结论

`#28` 采用“端到端测试先行 + 最小胶水修复 + demo 脚本 + 文档收口”的实现方式，最大化体现：

- 架构融合能力
- 串并行判断能力
- 真实 AI 业务闭环理解
- 可讲述、可演示、可复现的项目交付能力
