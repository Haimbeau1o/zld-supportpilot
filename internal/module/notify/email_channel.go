package notify

import (
	"context"
	"encoding/json"
	"log"
	"strings"
)

// MailSender 抽象邮件底层发送动作，当前阶段先用日志 sender 稳定渠道边界。
type MailSender interface {
	Send(ctx context.Context, mail Mail) error
}

type EmailChannel struct {
	sender MailSender
}

func NewEmailChannel(sender MailSender) *EmailChannel {
	return &EmailChannel{sender: sender}
}

func (channel *EmailChannel) Name() string {
	return "email"
}

func (channel *EmailChannel) Deliver(ctx context.Context, request Request) error {
	if channel == nil || channel.sender == nil {
		return nil
	}
	to := make([]string, 0, len(request.Recipients))
	seen := make(map[string]struct{}, len(request.Recipients))
	for _, recipient := range request.Recipients {
		email := strings.TrimSpace(recipient.Email)
		if email == "" {
			continue
		}
		if _, exists := seen[email]; exists {
			continue
		}
		seen[email] = struct{}{}
		to = append(to, email)
	}
	if len(to) == 0 {
		return nil
	}
	return channel.sender.Send(ctx, Mail{
		To:       to,
		Subject:  request.Event.Subject,
		Content:  request.Event.Content,
		Metadata: request.Event.Metadata,
	})
}

type LogMailSender struct{}

func NewLogMailSender() *LogMailSender {
	return &LogMailSender{}
}

func (sender *LogMailSender) Send(_ context.Context, mail Mail) error {
	payload, err := json.Marshal(mail)
	if err != nil {
		return err
	}
	log.Printf("notification_email %s", string(payload))
	return nil
}
