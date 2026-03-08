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

## 当前已实现

当前首版基础能力已提供：

- `GET /healthz`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `GET /api/v1/auth/me`
- `POST /api/v1/tickets`
- `GET /api/v1/tickets`
- `GET /api/v1/tickets/{id}`
- `POST /api/v1/tickets/{id}/assign`
- `POST /api/v1/tickets/{id}/status`
- `POST /api/v1/tickets/{id}/comments`
- `GET /api/v1/tickets/{id}/timeline`
- `POST /api/v1/knowledge/bases`
- `GET /api/v1/knowledge/bases`
- `POST /api/v1/knowledge/bases/{id}/documents`
- `GET /api/v1/knowledge/bases/{id}/documents`
- `GET /api/v1/knowledge/documents/{id}`
- `GET /api/v1/knowledge/documents/{id}/tasks`
- `POST /api/v1/knowledge/documents/{id}/retry`
- `POST /api/v1/ai/knowledge/bases/{id}/answers`
- `POST /api/v1/ai/tickets/{id}/assist`
- `GET /debug/metrics/http`
- 环境变量加载与应用装配骨架
- 基于 JWT 的最小认证链路
- 基于租户角色的 RBAC 权限映射
- 基于显式状态机的工单主流程
- 基于知识库容器与元数据记录的文档上传链路
- 基于后台 worker 的异步文档处理流水线（解析 / 切块 / 索引）
- 基于内存向量检索与引用返回的 RAG 回答接口
- 基于 `X-Request-ID` 的请求关联与结构化访问日志
- 基于进程内聚合的 HTTP 指标快照
- 基于令牌桶的关键路径限流（登录 / 文档上传 / 重试 / AI 问答）

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
go run ./cmd/api
```

默认关键环境变量：

- `APP_NAME`：应用名，默认 `zld-supportpilot`
- `APP_ENV`：运行环境，默认 `dev`
- `HTTP_ADDR`：监听地址，默认 `:8080`
- `AUTH_SIGNING_KEY`：JWT 签名密钥，本地默认 `dev-only-signing-key`
- `AUTH_TOKEN_TTL_SECONDS`：访问令牌有效期秒数，默认 `3600`
- `HTTP_RATE_LIMIT_ENABLED`：是否启用应用层限流，默认 `true`
- `HTTP_RATE_LIMIT_RPS`：限流令牌桶每秒补充速率，默认 `5`
- `HTTP_RATE_LIMIT_BURST`：限流令牌桶突发容量，默认 `10`

## 可观测性与限流说明

### 请求关联

- 每个已注册接口都会返回 `X-Request-ID`
- 如果客户端已经传入 `X-Request-ID`，服务端会复用该值，方便串联调用链与日志
- 访问日志会输出 `route`、`path`、`status`、`duration_ms`，在鉴权成功的接口上还会带出 `user_id` 与 `organization_id`

### 指标查看

```bash
curl http://localhost:8080/debug/metrics/http
```

返回的是项目内可直接阅读的 JSON 快照，适合本地调试与面试演示。当前会聚合：

- 请求总数 `requests_total`
- 状态码分布 `status_totals`
- 路由维度统计 `route_totals`
- 延迟桶分布 `latency_ms_buckets`
- 限流命中次数 `rate_limited_total`

### 限流边界

当前阶段只对“高风险或高成本”的关键路径施加限流：

- `POST /api/v1/auth/login`
- `POST /api/v1/knowledge/bases/{id}/documents`
- `POST /api/v1/knowledge/documents/{id}/retry`
- `POST /api/v1/ai/knowledge/bases/{id}/answers`
- `POST /api/v1/ai/tickets/{id}/assist`

触发限流时会返回：

- HTTP 状态码 `429 Too Many Requests`
- 统一错误结构 `{"error":{"code":"rate_limited",...}}`
- `Retry-After` 响应头

访问健康检查：

```bash
curl http://localhost:8080/healthz
```

注册：

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Alice",
    "email": "alice@example.com",
    "password": "secret123",
    "tenant_name": "中联数据智能服务台"
  }'
```

登录：

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "alice@example.com",
    "password": "secret123"
  }'
```

创建工单：

```bash
curl -X POST http://localhost:8080/api/v1/tickets \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <access_token>' \
  -d '{
    "title": "VPN 无法连接",
    "description": "今天上午开始无法连接公司 VPN",
    "category": "network",
    "priority": "high"
  }'
```

追加公开评论：

```bash
curl -X POST http://localhost:8080/api/v1/tickets/<ticket_id>/comments \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <access_token>' \
  -d '{
    "type": "comment",
    "content": "已补充故障截图，请继续排查。"
  }'
```

查询工单时间线：

```bash
curl http://localhost:8080/api/v1/tickets/<ticket_id>/timeline \
  -H 'Authorization: Bearer <access_token>'
```

创建知识库：

```bash
curl -X POST http://localhost:8080/api/v1/knowledge/bases \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <access_token>' \
  -d '{
    "name": "IT 支持知识库",
    "description": "用于沉淀 IT 文档"
  }'
```

上传文档：

```bash
curl -X POST http://localhost:8080/api/v1/knowledge/bases/<knowledge_base_id>/documents \
  -H 'Authorization: Bearer <access_token>' \
  -F 'file=@./vpn-guide.pdf'
```

上传成功后，文档会先返回 `uploaded` 状态，后台 worker 会异步推进到 `ready` 或 `failed`。

查询文档处理任务：

```bash
curl http://localhost:8080/api/v1/knowledge/documents/<document_id>/tasks \
  -H 'Authorization: Bearer <access_token>'
```

重试失败文档：

```bash
curl -X POST http://localhost:8080/api/v1/knowledge/documents/<document_id>/retry \
  -H 'Authorization: Bearer <access_token>'
```

知识库问答：

```bash
curl -X POST http://localhost:8080/api/v1/ai/knowledge/bases/<knowledge_base_id>/answers \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <access_token>' \
  -d '{
    "question": "VPN 无法连接怎么办？",
    "top_k": 3
  }'
```

当检索置信度不足时，接口会返回 `degraded` 状态，而不是编造答案。

生成工单 AI 辅助建议：

```bash
curl -X POST http://localhost:8080/api/v1/ai/tickets/<ticket_id>/assist   -H 'Content-Type: application/json'   -H 'Authorization: Bearer <access_token>'   -d '{
    "knowledge_base_id": "<knowledge_base_id>"
  }'
```

该接口会返回分类建议、处理摘要、回复草稿，并将本次 AI 结果写入工单内部备注；系统不会自动修改工单主数据或自动回复用户。

## 演示建议

建议按下面顺序做本地演示，这样最容易把“AI 应用能力 + 传统后端工程化”讲清楚：

1. 启动服务后先调用 `/healthz`，展示响应头中的 `X-Request-ID`
2. 注册并登录，拿到 `access_token`
3. 创建知识库并上传一个简单文本或 PDF 文档
4. 调用 AI 问答接口，展示返回的答案、引用和可能的 `degraded` 降级状态
5. 调用工单 AI 辅助接口，展示分类建议、摘要、回复草稿和内部备注沉淀
6. 访问 `/debug/metrics/http`，展示请求数、状态码分布和限流计数
7. 连续触发登录或 AI 问答 / 工单 AI 辅助接口，演示 `429` 与 `Retry-After`

如果要在面试或汇报里强调架构取舍，可以直接说明：当前仓库故意选择“请求 ID + 结构化日志 + 进程内指标 + 进程内令牌桶”的最小工程化闭环，而不是提前接入完整外部观测栈，这样更符合单进程 demo 阶段的真实复杂度。

查询知识库列表：

```bash
curl http://localhost:8080/api/v1/knowledge/bases \
  -H 'Authorization: Bearer <access_token>'
```
