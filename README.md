# ZLD SupportPilot

智能服务台后端项目，面向企业内部支持场景，聚焦知识沉淀、重复问答、工单流转和 AI 辅助处理。

## Project Background

`ZLD SupportPilot` 以企业内部服务支持场景为背景，由 3 人横向协作推进。我主要负责后端架构设计、核心业务模块和 AI 应用能力落地。

项目一期目标：

- 建立统一的知识问答与工单入口
- 支持多租户、RBAC、工单状态流转
- 支持文档导入、检索问答、引用溯源
- 支持 AI 工单分类、摘要和回复草稿

## Architecture

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
- `docs/architecture`：架构文档
- `docs/plans`：实施计划

## Current Bootstrap

当前首版骨架提供：

- `GET /healthz`
- 环境变量加载
- HTTP 服务器启动骨架
- Docker Compose 依赖占位
- GitHub Issue / PR 模板

## Quick Start

```bash
cp .env.example .env
make run
```

访问：

```bash
curl http://localhost:8080/healthz
```

## Roadmap

- Milestone 1: Repo Bootstrap
- Milestone 2: Auth + Tenant + RBAC
- Milestone 3: Ticket Workflow
- Milestone 4: Knowledge Base + RAG
- Milestone 5: AI Assistant + Observability

