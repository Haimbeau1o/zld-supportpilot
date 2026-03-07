# ZLD SupportPilot Architecture Overview

## Positioning

ZLD SupportPilot 面向企业内部支持场景，目标是把知识问答与工单处理整合进一个统一入口，降低重复答复成本，提高支持效率与流程透明度。

## Architecture Style

项目一期采用模块化单体：

- 对外暴露统一 HTTP API
- 通过领域模块隔离业务边界
- 使用异步 worker 处理文档解析与 AI 后处理任务
- 保持单仓库、单主服务、单 worker 的低复杂度交付方式

## Core Modules

- `identity`：用户、组织、成员、角色权限
- `ticket`：工单生命周期与协作流
- `knowledge`：文档与知识库索引
- `ai`：RAG、摘要、分类、回复草稿

## Infrastructure Plan

- `PostgreSQL + pgvector`：业务数据与向量检索
- `Redis`：缓存、限流、任务队列依赖
- `MinIO`：原始文件存储
- `OpenTelemetry + Prometheus + Grafana`：可观测性

## Delivery Principle

先做可运行、可演示、可测试的 MVP，再逐步补 AI 深度与运维能力，避免仓库早期复杂度失控。

