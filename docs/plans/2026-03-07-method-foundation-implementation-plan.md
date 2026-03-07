# 方法与协作基础 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 `ZLD SupportPilot` 建立学习型研发流程、任务编排规范、双层 skill 和中文 Issue / PR 工作方式。

**Architecture:** 先建立方法层文档、模板与技能，再反向约束后续业务 Issue 的推进方式。通过仓库内模板与 skill 双层结构同时保证“可追踪”和“可复用”。

**Tech Stack:** Markdown, GitHub Issues/PRs, repository-local skills

### Task 1: 建立方法层文档骨架

**Files:**
- Create: `docs/templates/README.md`
- Create: `docs/coordination/README.md`
- Create: `docs/workitems/README.md`
- Modify: `README.md`
- Modify: `CONTRIBUTING.md`

**Step 1:** 建立模板目录、协作目录和工作包目录。
**Step 2:** 补充仓库级方法说明和中文协作规范。
**Step 3:** 明确每个 Issue 的 7 件套交付标准。
**Step 4:** 复查 README、CONTRIBUTING 与路线图是否一致。

### Task 2: 建立双层 skill

**Files:**
- Modify: `skills/learning-rd-workflow/SKILL.md`
- Create: `skills/learning-rd-workflow/references/交付物清单.md`
- Create: `skills/learning-rd-workflow/references/并发性分析法.md`
- Modify: `skills/zld-supportpilot-delivery/SKILL.md`
- Create: `skills/zld-supportpilot-delivery/references/模块边界与依赖.md`
- Create: `skills/zld-supportpilot-delivery/references/当前路线图与融合策略.md`

**Step 1:** 先定义通用 skill 的触发条件和固定输出。
**Step 2:** 再定义项目专属 skill 的模块边界与阶段策略。
**Step 3:** 用校验脚本检查 skill 结构是否合法。

### Task 3: 建立依赖分析与执行波次

**Files:**
- Create: `docs/coordination/当前Issue依赖与并发分析.md`
- Create: `docs/coordination/执行波次规划.md`
- Create: `docs/workitems/issue-011-学习型研发流程与任务编排规范/*`

**Step 1:** 对现有 Issue 做依赖分析，判断串行与并行。
**Step 2:** 给出执行波次和融合 Issue 建议。
**Step 3:** 为 M0 主线 Issue 建立完整工作包文档。

### Task 4: 改造 GitHub 工作方式

**Files:**
- Modify: `.github/ISSUE_TEMPLATE/*.yml`
- Modify: `.github/PULL_REQUEST_TEMPLATE.md`
- GitHub: 当前 milestones、issues、PR

**Step 1:** 将模板改成中文并加入并发性字段。
**Step 2:** 创建 M0 相关 Issue 与 milestone。
**Step 3:** 将现有英文 Issue / PR 改成中文。
**Step 4:** 验证 issue 标题、阶段划分和本地路线图一致。
