# ZLD SupportPilot

面向企业内部支持场景的智能服务台后端项目，聚焦知识沉淀、重复问答、工单流转和 AI 辅助处理。

## 项目背景

`ZLD SupportPilot` 以企业内部服务支持场景为背景，由 3 人横向协作推进。我主要负责后端架构设计、核心业务模块和 AI 应用能力落地。

项目一期目标：

- 建立统一的知识问答与工单入口
- 支持多租户、RBAC、工单状态流转
- 支持文档导入、检索问答、引用溯源
- 支持 AI 工单分类、摘要和回复草稿

## 技术架构

当前仓库采用模块化单体设计：

- `cmd/api`：HTTP API 入口
- `internal/app`：应用装配
- `internal/platform`：配置、HTTP、基础设施适配层
- `internal/module/identity`：身份、组织、角色权限
- `internal/module/ticket`：工单、评论、状态流转
- `internal/module/knowledge`：知识库、文档、切块、索引
- `internal/module/ai`：RAG、Prompt、AI 工作流
- `pkg/llm`：模型抽象层
- `deployments/docker`：本地依赖环境

## 当前骨架

当前首版骨架已提供：

- `GET /healthz`
- 环境变量加载
- HTTP 服务器启动骨架
- Docker Compose 依赖占位
- GitHub Issue / PR 模板

## 学习型研发流程

本仓库不是“只交付功能”的作品仓库，而是“边做边学、边做边沉淀”的学习型工程仓库。

固定规则：

- GitHub `Issue` 和 `PR` 全部使用中文
- 每个 `Issue` 必须沉淀到 `docs/workitems/issue-xxx-主题/`
- 每个 `Issue` 固定输出 7 份文档：任务卡、调研记录、方案设计、实施计划、验证记录、工作日志、复盘总结
- 实现代码需要在关键业务逻辑、边界条件、设计意图处补充中文注释
- 每次开始实现前先分析依赖关系，判断串行、并行和融合点
- 并行工作完成后，必须通过“融合 Issue”收口接口、数据流、联调和阶段结论

## 文档索引

- `docs/templates/`：统一模板
- `docs/coordination/`：依赖图、执行波次、融合策略
- `docs/workitems/`：按 Issue 维度沉淀完整工作包
- `docs/plans/2026-03-07-delivery-roadmap.md`：当前路线图与阶段规划
- `skills/learning-rd-workflow/`：通用学习型研发流程 skill
- `skills/zld-supportpilot-delivery/`：项目专属交付 skill

## 路线图

- `M0 - 方法与协作基础`：模板、任务编排规范、双层 skill、中文 issue/PR 流程
- `M1 - 基础能力`：认证、租户、RBAC
- `M2 - 工单与知识库`：工单主流程、协作记录、文档上传、异步处理
- `M3 - AI 与工程化`：RAG 问答、AI 辅助能力、观测、限流、发布整理

## 快速启动

```bash
cp .env.example .env
make run
```

访问：

```bash
curl http://localhost:8080/healthz
```
