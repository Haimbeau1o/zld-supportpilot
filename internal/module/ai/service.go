package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/knowledge"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/notify"
	"github.com/Haimbeau1o/zld-supportpilot/internal/module/ticket"
)

var (
	ErrInvalidAIInput        = errors.New("invalid ai input")
	ErrUnifiedIntakeNotFound = errors.New("unified intake not found")
)

// ChunkSource 定义 AI 服务读取 ready chunk 的边界，避免 AI 直接介入知识处理流水线内部细节。
type ChunkSource interface {
	ListIndexedChunks(actor identity.IdentityContext, knowledgeBaseID string) ([]knowledge.DocumentChunk, error)
}

// AnswerGenerator 负责把检索结果组织成最终回答文本。
type AnswerGenerator interface {
	GenerateAnswer(ctx context.Context, question string, retrievedChunks []RetrievedChunk) (string, error)
}

// KnowledgeAnswerer 抽象已有的知识问答能力，便于 ticket assist 与统一受理直接复用 RAG 结果。
type KnowledgeAnswerer interface {
	AskKnowledgeQuestion(ctx context.Context, actor identity.IdentityContext, input AskKnowledgeQuestionInput) (AnswerResult, error)
}

// TicketWorkspace 定义 AI 读取工单、读取时间线、写备注与创建工单的边界。
type TicketWorkspace interface {
	GetTicket(actor identity.IdentityContext, ticketID string) (ticket.Ticket, error)
	ListTicketTimeline(actor identity.IdentityContext, ticketID string) ([]ticket.TicketTimelineItem, error)
	AddTicketComment(actor identity.IdentityContext, input ticket.AddTicketCommentInput) (ticket.TicketComment, error)
	CreateTicket(actor identity.IdentityContext, input ticket.CreateTicketInput) (ticket.Ticket, error)
}

// TicketAnalyzer 负责把工单上下文和知识结果转换成 AI 建议。
type TicketAnalyzer interface {
	AnalyzeTicket(input TicketAnalysisInput) TicketAnalysisResult
}

type NotificationPublisher interface {
	Publish(ctx context.Context, event notify.Event) error
}

type ServiceDependencies struct {
	ChunkSource           ChunkSource
	Retriever             Retriever
	AnswerGenerator       AnswerGenerator
	MinConfidence         float64
	TicketWorkspace       TicketWorkspace
	TicketAnalyzer        TicketAnalyzer
	KnowledgeAnswerer     KnowledgeAnswerer
	IntakeRepository      UnifiedIntakeRepository
	FeedbackRepository    AnswerFeedbackRepository
	NotificationPublisher NotificationPublisher
}

type Service struct {
	chunkSource           ChunkSource
	retriever             Retriever
	answerGenerator       AnswerGenerator
	minConfidence         float64
	ticketWorkspace       TicketWorkspace
	ticketAnalyzer        TicketAnalyzer
	knowledgeAnswerer     KnowledgeAnswerer
	intakeRepository      UnifiedIntakeRepository
	feedbackRepository    AnswerFeedbackRepository
	notificationPublisher NotificationPublisher
}

func NewService(dependencies ServiceDependencies) *Service {
	minConfidence := dependencies.MinConfidence
	if minConfidence <= 0 {
		minConfidence = 0.15
	}
	service := &Service{
		chunkSource:           dependencies.ChunkSource,
		retriever:             dependencies.Retriever,
		answerGenerator:       dependencies.AnswerGenerator,
		minConfidence:         minConfidence,
		ticketWorkspace:       dependencies.TicketWorkspace,
		ticketAnalyzer:        dependencies.TicketAnalyzer,
		knowledgeAnswerer:     dependencies.KnowledgeAnswerer,
		intakeRepository:      dependencies.IntakeRepository,
		feedbackRepository:    dependencies.FeedbackRepository,
		notificationPublisher: dependencies.NotificationPublisher,
	}
	if service.knowledgeAnswerer == nil && service.chunkSource != nil && service.retriever != nil && service.answerGenerator != nil {
		service.knowledgeAnswerer = service
	}
	return service
}

func (service *Service) AskKnowledgeQuestion(ctx context.Context, actor identity.IdentityContext, input AskKnowledgeQuestionInput) (AnswerResult, error) {
	if service.chunkSource == nil || service.retriever == nil || service.answerGenerator == nil {
		return AnswerResult{}, fmt.Errorf("%w: ai service dependencies are incomplete", ErrInvalidAIInput)
	}

	knowledgeBaseID := strings.TrimSpace(input.KnowledgeBaseID)
	question := strings.TrimSpace(input.Question)
	if knowledgeBaseID == "" {
		return AnswerResult{}, fmt.Errorf("%w: knowledge base id is required", ErrInvalidAIInput)
	}
	if question == "" {
		return AnswerResult{}, fmt.Errorf("%w: question is required", ErrInvalidAIInput)
	}

	topK := input.TopK
	if topK == 0 {
		topK = 3
	}
	if topK < 0 {
		return AnswerResult{}, fmt.Errorf("%w: top_k must be positive", ErrInvalidAIInput)
	}

	chunks, err := service.chunkSource.ListIndexedChunks(actor, knowledgeBaseID)
	if err != nil {
		return AnswerResult{}, err
	}

	retrievedChunks, err := service.retriever.Retrieve(ctx, question, chunks, topK)
	if err != nil {
		return AnswerResult{}, err
	}
	if len(retrievedChunks) == 0 {
		return degradedAnswerResult(), nil
	}

	confidence := retrievedChunks[0].Score
	if confidence < service.minConfidence {
		return degradedAnswerResult(), nil
	}

	answer, err := service.answerGenerator.GenerateAnswer(ctx, question, retrievedChunks)
	if err != nil {
		return AnswerResult{}, err
	}

	citations := make([]Citation, 0, len(retrievedChunks))
	for _, item := range retrievedChunks {
		citations = append(citations, Citation{
			ChunkID:    item.Chunk.ID,
			DocumentID: item.Chunk.DocumentID,
			Score:      item.Score,
			Content:    item.Chunk.Content,
		})
	}

	return AnswerResult{
		Status:     AnswerStatusAnswered,
		Answer:     answer,
		Confidence: confidence,
		Citations:  citations,
	}, nil
}

func (service *Service) HandleUnifiedIntake(ctx context.Context, actor identity.IdentityContext, input UnifiedIntakeInput) (UnifiedIntakeResult, error) {
	if service.knowledgeAnswerer == nil || service.ticketWorkspace == nil || service.intakeRepository == nil {
		return UnifiedIntakeResult{}, fmt.Errorf("%w: unified intake dependencies are incomplete", ErrInvalidAIInput)
	}

	knowledgeBaseID := strings.TrimSpace(input.KnowledgeBaseID)
	question := strings.TrimSpace(input.Question)
	if knowledgeBaseID == "" {
		return UnifiedIntakeResult{}, fmt.Errorf("%w: knowledge base id is required", ErrInvalidAIInput)
	}
	if question == "" {
		return UnifiedIntakeResult{}, fmt.Errorf("%w: question is required", ErrInvalidAIInput)
	}

	knowledgeResult, err := service.knowledgeAnswerer.AskKnowledgeQuestion(ctx, actor, AskKnowledgeQuestionInput{
		KnowledgeBaseID: knowledgeBaseID,
		Question:        question,
		TopK:            input.TopK,
	})
	if err != nil {
		return UnifiedIntakeResult{}, err
	}

	now := time.Now()
	result := UnifiedIntakeResult{
		AnswerStatus: knowledgeResult.Status,
		Answer:       knowledgeResult.Answer,
		Confidence:   knowledgeResult.Confidence,
		Citations:    knowledgeResult.Citations,
	}
	record := UnifiedIntakeRecord{
		OrganizationID:  actor.OrganizationID,
		RequesterID:     actor.UserID,
		KnowledgeBaseID: knowledgeBaseID,
		Question:        question,
		AnswerStatus:    knowledgeResult.Status,
		Answer:          knowledgeResult.Answer,
		Confidence:      knowledgeResult.Confidence,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if knowledgeResult.Status == AnswerStatusAnswered {
		result.ResultType = UnifiedIntakeResultTypeAnswered
		record.ResultType = UnifiedIntakeResultTypeAnswered
	} else {
		createdTicket, err := service.ticketWorkspace.CreateTicket(actor, ticket.CreateTicketInput{
			Title:       buildUnifiedIntakeTicketTitle(question),
			Description: question,
			Category:    "general",
			Priority:    ticket.TicketPriorityMedium,
		})
		if err != nil {
			return UnifiedIntakeResult{}, err
		}
		result.ResultType = UnifiedIntakeResultTypeTicketCreated
		result.TicketID = createdTicket.ID
		record.ResultType = UnifiedIntakeResultTypeTicketCreated
		record.TicketID = createdTicket.ID
		service.publishEscalationNotification(ctx, actor, createdTicket, question, "intake_degraded")
	}

	savedRecord := service.intakeRepository.Save(record)
	result.IntakeID = savedRecord.ID
	return result, nil
}

func (service *Service) SubmitAnswerFeedback(ctx context.Context, actor identity.IdentityContext, input AnswerFeedbackInput) (AnswerFeedbackResult, error) {
	_ = ctx
	if service.intakeRepository == nil || service.feedbackRepository == nil || service.ticketWorkspace == nil {
		return AnswerFeedbackResult{}, fmt.Errorf("%w: answer feedback dependencies are incomplete", ErrInvalidAIInput)
	}

	intakeID := strings.TrimSpace(input.IntakeID)
	comment := strings.TrimSpace(input.Comment)
	if intakeID == "" {
		return AnswerFeedbackResult{}, fmt.Errorf("%w: intake id is required", ErrInvalidAIInput)
	}
	if !isValidAnswerFeedbackStatus(input.Status) {
		return AnswerFeedbackResult{}, fmt.Errorf("%w: unsupported feedback status", ErrInvalidAIInput)
	}

	intakeRecord, ok := service.intakeRepository.FindByID(intakeID)
	if !ok {
		return AnswerFeedbackResult{}, ErrUnifiedIntakeNotFound
	}
	if actor.OrganizationID != intakeRecord.OrganizationID {
		return AnswerFeedbackResult{}, ticket.ErrTicketForbidden
	}

	ticketAction := AnswerFeedbackTicketActionNone
	ticketID := ""
	if input.Status == AnswerFeedbackStatusUnresolved {
		// 未解决反馈的目标不是重复问一遍 AI，而是确保问题真正进入人工跟进流程。
		if linkedTicketID := resolveLinkedTicketID(intakeRecord); linkedTicketID != "" {
			ticketAction = AnswerFeedbackTicketActionTicketExisting
			ticketID = linkedTicketID
		} else {
			createdTicket, err := service.ticketWorkspace.CreateTicket(actor, ticket.CreateTicketInput{
				Title:       buildUnifiedIntakeTicketTitle(intakeRecord.Question),
				Description: buildFeedbackEscalationDescription(intakeRecord, comment),
				Category:    "general",
				Priority:    ticket.TicketPriorityMedium,
			})
			if err != nil {
				return AnswerFeedbackResult{}, err
			}
			ticketAction = AnswerFeedbackTicketActionTicketCreated
			ticketID = createdTicket.ID
			intakeRecord.EscalatedTicketID = createdTicket.ID
			intakeRecord.UpdatedAt = time.Now()
			service.intakeRepository.Save(intakeRecord)
		}
	}

	now := time.Now()
	feedback := AnswerFeedback{
		IntakeID:       intakeRecord.ID,
		OrganizationID: actor.OrganizationID,
		ActorID:        actor.UserID,
		Status:         input.Status,
		Comment:        comment,
		TicketAction:   ticketAction,
		TicketID:       ticketID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if existing, ok := service.feedbackRepository.FindByIntakeAndActor(intakeRecord.ID, actor.UserID); ok {
		feedback.ID = existing.ID
		feedback.CreatedAt = existing.CreatedAt
	}

	savedFeedback := service.feedbackRepository.Save(feedback)
	if ticketAction == AnswerFeedbackTicketActionTicketCreated && ticketID != "" {
		service.publishEscalationNotification(ctx, actor, ticket.Ticket{
			ID:             ticketID,
			OrganizationID: actor.OrganizationID,
			RequesterID:    intakeRecord.RequesterID,
			Title:          buildUnifiedIntakeTicketTitle(intakeRecord.Question),
		}, intakeRecord.Question, "feedback_unresolved")
	}
	return AnswerFeedbackResult{
		FeedbackID:   savedFeedback.ID,
		IntakeID:     savedFeedback.IntakeID,
		Status:       savedFeedback.Status,
		TicketAction: savedFeedback.TicketAction,
		TicketID:     savedFeedback.TicketID,
	}, nil
}

func (service *Service) GetAnswerFeedbackStats(_ context.Context, actor identity.IdentityContext) (AnswerFeedbackStats, error) {
	if service.feedbackRepository == nil {
		return AnswerFeedbackStats{}, fmt.Errorf("%w: answer feedback dependencies are incomplete", ErrInvalidAIInput)
	}
	if !canReadAnswerFeedbackStats(actor) {
		return AnswerFeedbackStats{}, ticket.ErrTicketForbidden
	}

	items := service.feedbackRepository.ListByOrganization(actor.OrganizationID)
	stats := AnswerFeedbackStats{}
	for _, item := range items {
		stats.Total++
		switch item.Status {
		case AnswerFeedbackStatusResolved:
			stats.Resolved++
		case AnswerFeedbackStatusUnresolved:
			stats.Unresolved++
		case AnswerFeedbackStatusInaccurate:
			stats.Inaccurate++
		}
	}
	return stats, nil
}

func (service *Service) GenerateTicketAssist(ctx context.Context, actor identity.IdentityContext, input GenerateTicketAssistInput) (TicketAssistResult, error) {
	if service.ticketWorkspace == nil || service.ticketAnalyzer == nil || service.knowledgeAnswerer == nil {
		return TicketAssistResult{}, fmt.Errorf("%w: ticket assist dependencies are incomplete", ErrInvalidAIInput)
	}
	if !canGenerateTicketAssist(actor) {
		return TicketAssistResult{}, ticket.ErrTicketForbidden
	}

	ticketID := strings.TrimSpace(input.TicketID)
	knowledgeBaseID := strings.TrimSpace(input.KnowledgeBaseID)
	if ticketID == "" {
		return TicketAssistResult{}, fmt.Errorf("%w: ticket id is required", ErrInvalidAIInput)
	}
	if knowledgeBaseID == "" {
		return TicketAssistResult{}, fmt.Errorf("%w: knowledge base id is required", ErrInvalidAIInput)
	}

	currentTicket, err := service.ticketWorkspace.GetTicket(actor, ticketID)
	if err != nil {
		return TicketAssistResult{}, err
	}
	timeline, err := service.ticketWorkspace.ListTicketTimeline(actor, ticketID)
	if err != nil {
		return TicketAssistResult{}, err
	}

	knowledgeResult, err := service.knowledgeAnswerer.AskKnowledgeQuestion(ctx, actor, AskKnowledgeQuestionInput{
		KnowledgeBaseID: knowledgeBaseID,
		Question:        buildTicketAssistQuestion(currentTicket, timeline),
		TopK:            3,
	})
	if err != nil {
		return TicketAssistResult{}, err
	}

	analysis := service.ticketAnalyzer.AnalyzeTicket(TicketAnalysisInput{
		Ticket:          currentTicket,
		Timeline:        timeline,
		KnowledgeAnswer: knowledgeResult,
	})

	// 当前阶段只把 AI 结果写入内部备注，而不自动改工单主数据，这样既能沉淀痕迹，也不会越权执行。
	recordedComment, err := service.ticketWorkspace.AddTicketComment(actor, ticket.AddTicketCommentInput{
		TicketID: currentTicket.ID,
		Type:     ticket.TicketCommentTypeInternalNote,
		Content:  formatTicketAssistRecord(actor, knowledgeBaseID, analysis, knowledgeResult),
	})
	if err != nil {
		return TicketAssistResult{}, err
	}

	return TicketAssistResult{
		CategorySuggestion: analysis.CategorySuggestion,
		Summary:            analysis.Summary,
		ReplyDraft:         analysis.ReplyDraft,
		AnswerStatus:       knowledgeResult.Status,
		Confidence:         knowledgeResult.Confidence,
		Citations:          knowledgeResult.Citations,
		RecordedCommentID:  recordedComment.ID,
	}, nil
}

func (service *Service) publishEscalationNotification(ctx context.Context, actor identity.IdentityContext, createdTicket ticket.Ticket, question string, source string) {
	if service.notificationPublisher == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	// AI 升级通知的目标是把“自助失败已进入人工流程”明确触达出去，因此这里最佳努力发送，但不影响建单结果本身。
	_ = service.notificationPublisher.Publish(ctx, notify.Event{
		Type:             notify.EventTypeAIEscalated,
		OrganizationID:   actor.OrganizationID,
		TicketID:         createdTicket.ID,
		ActorID:          actor.UserID,
		RecipientUserIDs: uniqueRecipientUserIDs(actor.UserID, createdTicket.RequesterID),
		Subject:          "AI 已升级为人工跟进",
		Content:          fmt.Sprintf("问题“%s”已升级为人工工单，当前工单号为 %s。", strings.TrimSpace(question), createdTicket.ID),
		Metadata: map[string]string{
			"source": source,
		},
		CreatedAt: time.Now(),
	})
}

// TemplateAnswerGenerator 先用可解释的模板式回答把接口契约稳定下来，后续可平滑替换成真实 LLM 生成器。
type TemplateAnswerGenerator struct{}

func (TemplateAnswerGenerator) GenerateAnswer(_ context.Context, question string, retrievedChunks []RetrievedChunk) (string, error) {
	if len(retrievedChunks) == 0 {
		return degradedAnswerResult().Answer, nil
	}

	builder := strings.Builder{}
	builder.WriteString("根据知识库资料，建议优先执行以下操作：")
	for index, chunk := range retrievedChunks {
		builder.WriteString(fmt.Sprintf(" [%d] %s", index+1, strings.TrimSpace(chunk.Chunk.Content)))
		if index >= 1 {
			break
		}
	}
	builder.WriteString(" 如仍无法解决，建议转人工进一步排查。")
	return builder.String(), nil
}

// TemplateTicketAnalyzer 用规则化模板先把 AI 辅助接口契约稳定下来，后续可以平滑替换为真实模型生成。
type TemplateTicketAnalyzer struct{}

func (TemplateTicketAnalyzer) AnalyzeTicket(input TicketAnalysisInput) TicketAnalysisResult {
	categorySuggestion := suggestTicketCategory(input)
	summary := buildTicketSummary(input)
	replyDraft := buildReplyDraft(input)
	return TicketAnalysisResult{
		CategorySuggestion: categorySuggestion,
		Summary:            summary,
		ReplyDraft:         replyDraft,
	}
}

func degradedAnswerResult() AnswerResult {
	// 当检索置信度不足时，宁可显式降级，也不返回“看似合理”的猜测性答案。
	return AnswerResult{
		Status:     AnswerStatusDegraded,
		Answer:     "当前知识库暂无足够依据可靠回答该问题，建议转人工处理或补充相关文档后重试。",
		Confidence: 0,
		Citations:  []Citation{},
	}
}

func isValidAnswerFeedbackStatus(status AnswerFeedbackStatus) bool {
	switch status {
	case AnswerFeedbackStatusResolved, AnswerFeedbackStatusUnresolved, AnswerFeedbackStatusInaccurate:
		return true
	default:
		return false
	}
}

func canReadAnswerFeedbackStats(actor identity.IdentityContext) bool {
	return actor.Permissions.Contains(identity.PermissionTicketRead)
}

func resolveLinkedTicketID(record UnifiedIntakeRecord) string {
	if strings.TrimSpace(record.EscalatedTicketID) != "" {
		return strings.TrimSpace(record.EscalatedTicketID)
	}
	return strings.TrimSpace(record.TicketID)
}

func buildFeedbackEscalationDescription(record UnifiedIntakeRecord, feedbackComment string) string {
	builder := strings.Builder{}
	builder.WriteString("来源：AI 答案反馈升级。\n")
	builder.WriteString(fmt.Sprintf("原问题：%s\n", strings.TrimSpace(record.Question)))
	if answer := strings.TrimSpace(record.Answer); answer != "" {
		builder.WriteString(fmt.Sprintf("AI 原回答：%s\n", answer))
	}
	if comment := strings.TrimSpace(feedbackComment); comment != "" {
		builder.WriteString(fmt.Sprintf("用户反馈：%s\n", comment))
	}
	builder.WriteString("说明：用户反馈该答案未解决，需升级到人工继续处理。")
	return strings.TrimSpace(builder.String())
}

func canGenerateTicketAssist(actor identity.IdentityContext) bool {
	return actor.Permissions.Contains(identity.PermissionTicketWrite)
}

func buildTicketAssistQuestion(currentTicket ticket.Ticket, timeline []ticket.TicketTimelineItem) string {
	builder := strings.Builder{}
	builder.WriteString(strings.TrimSpace(currentTicket.Title))
	if description := strings.TrimSpace(currentTicket.Description); description != "" {
		builder.WriteString("。")
		builder.WriteString(description)
	}
	for index := len(timeline) - 1; index >= 0; index-- {
		item := timeline[index]
		content := strings.TrimSpace(item.Content)
		if item.ItemType != ticket.TicketTimelineItemTypeComment || content == "" {
			continue
		}
		// 生成新一轮建议时要跳过 AI 自己的执行记录，避免把旧的 AI 输出再次喂回检索链路造成递归污染。
		if item.CommentType == ticket.TicketCommentTypeInternalNote && strings.HasPrefix(content, "【AI 执行记录】") {
			continue
		}
		builder.WriteString("。补充信息：")
		builder.WriteString(content)
		break
	}
	return builder.String()
}

func buildUnifiedIntakeTicketTitle(question string) string {
	trimmed := strings.TrimSpace(question)
	trimmed = strings.TrimSuffix(trimmed, "？")
	trimmed = strings.TrimSuffix(trimmed, "?")
	if trimmed == "" {
		return "AI 受理问题"
	}
	if utf8.RuneCountInString(trimmed) <= 32 {
		return trimmed
	}
	runes := []rune(trimmed)
	return string(runes[:32]) + "..."
}

func suggestTicketCategory(input TicketAnalysisInput) string {
	contextText := strings.ToLower(strings.Join([]string{
		input.Ticket.Title,
		input.Ticket.Description,
		latestTimelineContent(input.Timeline),
		input.KnowledgeAnswer.Answer,
	}, " "))
	switch {
	case strings.Contains(contextText, "vpn") || strings.Contains(contextText, "网络") || strings.Contains(contextText, "连接"):
		return "network"
	case strings.Contains(contextText, "共享盘") || strings.Contains(contextText, "存储") || strings.Contains(contextText, "磁盘"):
		return "storage"
	case strings.Contains(contextText, "账号") || strings.Contains(contextText, "密码") || strings.Contains(contextText, "登录") || strings.Contains(contextText, "锁定"):
		return "account"
	case strings.Contains(contextText, "邮箱") || strings.Contains(contextText, "邮件"):
		return "mail"
	case strings.TrimSpace(input.Ticket.Category) != "":
		return strings.TrimSpace(input.Ticket.Category)
	default:
		return "general"
	}
}

func buildTicketSummary(input TicketAnalysisInput) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("工单“%s”当前状态为 %s。", strings.TrimSpace(input.Ticket.Title), input.Ticket.Status))
	if description := strings.TrimSpace(input.Ticket.Description); description != "" {
		builder.WriteString("用户描述：")
		builder.WriteString(description)
		builder.WriteString("。")
	}
	if latestContent := latestTimelineContent(input.Timeline); latestContent != "" {
		builder.WriteString("最近补充信息：")
		builder.WriteString(latestContent)
		builder.WriteString("。")
	}
	if input.KnowledgeAnswer.Status == AnswerStatusAnswered {
		builder.WriteString("知识库已有可参考处置建议。")
	} else {
		builder.WriteString("当前知识库未提供足够高置信度依据，建议进一步人工排查。")
	}
	return builder.String()
}

func buildReplyDraft(input TicketAnalysisInput) string {
	if input.KnowledgeAnswer.Status == AnswerStatusDegraded {
		// 当知识依据不足时，摘要仍可生成，但对外回复草稿必须显式保守，避免制造“AI 已经给出答案”的错觉。
		return "您好，已收到您的问题，我们正在进一步排查。当前知识库暂无足够依据可靠回答该问题，建议转人工深入处理，我们会继续跟进。"
	}
	return fmt.Sprintf("您好，已收到您关于“%s”的反馈。根据当前知识库建议，%s 如仍未恢复，请继续反馈具体现象，我们会进一步协助处理。", strings.TrimSpace(input.Ticket.Title), strings.TrimSpace(input.KnowledgeAnswer.Answer))
}

func uniqueRecipientUserIDs(userIDs ...string) []string {
	seen := make(map[string]struct{}, len(userIDs))
	recipients := make([]string, 0, len(userIDs))
	for _, rawUserID := range userIDs {
		userID := strings.TrimSpace(rawUserID)
		if userID == "" {
			continue
		}
		if _, exists := seen[userID]; exists {
			continue
		}
		seen[userID] = struct{}{}
		recipients = append(recipients, userID)
	}
	return recipients
}

func latestTimelineContent(timeline []ticket.TicketTimelineItem) string {
	for index := len(timeline) - 1; index >= 0; index-- {
		if strings.TrimSpace(timeline[index].Content) == "" {
			continue
		}
		return strings.TrimSpace(timeline[index].Content)
	}
	return ""
}

func formatTicketAssistRecord(actor identity.IdentityContext, knowledgeBaseID string, analysis TicketAnalysisResult, knowledgeResult AnswerResult) string {
	builder := strings.Builder{}
	// AI 执行记录先复用 internal note，是为了复用现有时间线与权限模型；等字段稳定后再考虑抽独立仓储。
	builder.WriteString("【AI 执行记录】\n")
	builder.WriteString(fmt.Sprintf("触发人：%s\n", actor.UserID))
	builder.WriteString(fmt.Sprintf("知识库：%s\n", knowledgeBaseID))
	builder.WriteString(fmt.Sprintf("分类建议：%s\n", analysis.CategorySuggestion))
	builder.WriteString(fmt.Sprintf("处理摘要：%s\n", analysis.Summary))
	builder.WriteString(fmt.Sprintf("回复草稿：%s\n", analysis.ReplyDraft))
	builder.WriteString(fmt.Sprintf("知识状态：%s\n", knowledgeResult.Status))
	if len(knowledgeResult.Citations) > 0 {
		builder.WriteString("引用：")
		for index, citation := range knowledgeResult.Citations {
			builder.WriteString(fmt.Sprintf("[%d]%s ", index+1, strings.TrimSpace(citation.Content)))
			if index >= 1 {
				break
			}
		}
	}
	return strings.TrimSpace(builder.String())
}
