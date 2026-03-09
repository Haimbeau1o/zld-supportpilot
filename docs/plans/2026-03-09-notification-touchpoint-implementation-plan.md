# #29 企业通知与待办触达实施计划

## 实施顺序

1. 建立 `issue-029` 工作包与调研文档
2. 为通知模块补 RED 测试
3. 实现通知抽象、邮件与 Webhook 渠道
4. 在 ticket / ai 服务挂接通知发布
5. 补齐配置与 README 说明
6. 跑模块回归与全量回归
7. 提交分支、创建 PR、同步 issue

## RED 清单

### 通知模块

- `TestServicePublishContinuesWhenChannelFails`
- `TestEmailChannelResolvesUserRecipients`
- `TestWebhookChannelPostsStructuredEvent`

### 业务挂点

- `TestAssignTicketPublishesNotification`
- `TestAssignTicketIgnoresNotificationFailure`
- `TestTransitionTicketStatusPublishesNotification`
- `TestHandleUnifiedIntakePublishesEscalationNotificationWhenKnowledgeIsDegraded`
- `TestSubmitAnswerFeedbackPublishesEscalationNotificationWhenTicketCreated`

## 实现边界

### 本次包含

- 通知抽象与渠道实现
- 业务服务事件发布
- identity 用户查找能力补齐（仅满足收件人解析需要）
- 配置与 README 说明

### 本次不包含

- SMTP 正式接入
- IM Bot 双向会话
- 消息中心列表页
- 通知投递持久化

## 验证命令

- `GOCACHE=/tmp/go-build go test -count=1 ./internal/module/notify`
- `GOCACHE=/tmp/go-build go test -count=1 ./internal/module/ticket`
- `GOCACHE=/tmp/go-build go test -count=1 ./internal/module/ai`
- `GOCACHE=/tmp/go-build go test -count=1 ./...`
