---
name: zld-supportpilot-delivery
description: "Use when Codex works in the ZLD SupportPilot repository on roadmap planning, Chinese issue or PR writing, dependency analysis, module delivery, or fusion planning across identity, ticket, knowledge, AI, and observability. Triggers: work under `docs/workitems/`, `docs/coordination/`, `.github/`, or feature planning for this repository."
---

# ZLD SupportPilot 项目交付

## 概览

把 `ZLD SupportPilot` 的业务交付和学习沉淀统一到同一套项目上下文中。始终以仓库现有模块边界和阶段路线图为准。

## 使用前先检查

先确认以下内容：

- 当前工作属于哪个阶段：`M0`、`M1`、`M2`、`M3`
- 当前 Issue 是主线、并行、融合、调研，还是复盘
- 当前 Issue 的 workitem 目录是否已创建
- `docs/coordination/` 是否已经更新依赖分析和执行波次

## 当前模块边界

- `identity`：用户、组织、成员、角色权限
- `ticket`：工单生命周期、评论、指派、审计事件
- `knowledge`：知识库、文档上传、切块、索引
- `ai`：RAG、摘要、分类、回复草稿
- `platform`：配置、HTTP、任务队列、观测和限流

详细说明见：`references/模块边界与依赖.md`

## 当前阶段约束

### M0 - 方法与协作基础

先建立模板、流程、skill 和依赖分析，再进入业务开发。

### M1 - 基础能力

优先完成认证、租户与 RBAC。这是后续工单和知识库的共同前置。

### M2 - 工单与知识库

在身份边界稳定后，允许 `ticket` 与 `knowledge` 两条主线并行。

### M3 - AI 与工程化

先完成知识链路，再接 RAG；先完成主链路，再做 AI 草稿类能力；工程化能力根据稳定度穿插推进。

## Issue 编排规则

### 主线 Issue

用于推进阶段目标。通常负责：

- 核心领域模型
- 主 API
- 主业务链路

### 并行 Issue

仅在边界稳定后创建。必须在正文中写清：

- 前置依赖
- 输入输出边界
- 可并行对象
- 融合 Issue

### 融合 Issue

专门用于以下工作：

- 接口对齐
- 数据流串联
- 权限边界校验
- 联调验证
- 阶段文档汇总

不要把融合 Issue 当成“顺手补一补”的杂项。

## 中文文档与代码要求

- Issue 标题和正文使用中文
- PR 标题和正文使用中文
- workitem 文档使用中文
- 业务代码中的关键逻辑写中文注释
- 注释解释设计意图、边界条件和维护注意事项

## 当前项目的并发性结论

- `#2` 必须先于 `#3` 与 `#5`
- `#3` 与 `#5` 可以并行
- `#4` 依赖 `#3`
- `#6` 依赖 `#5`
- `#7` 依赖 `#5` 与 `#6`
- `#8` 依赖 `#3` 与 `#7`
- `#9` 建议后续拆分

详细规划见：`references/当前路线图与融合策略.md`

## 常见错误

- 在 `#2` 之前直接推进工单或知识库主流程
- 在没有创建 workitem 目录时直接编码
- 在并行工作前没有约定接口和数据结构
- 把 AI 能力做成独立聊天 Demo，而不是嵌入服务台主流程
