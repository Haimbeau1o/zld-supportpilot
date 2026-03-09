package notify

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// UserDirectory 抽象按 userID 查询用户资料的能力，避免通知模块反向依赖完整身份实现。
type UserDirectory interface {
	FindUserByID(userID string) (User, bool)
}

// Channel 表示一个可替换的通知投递渠道，例如邮件或 Webhook。
type Channel interface {
	Name() string
	Deliver(ctx context.Context, request Request) error
}

// Publisher 是业务模块依赖的最小通知发布接口。
type Publisher interface {
	Publish(ctx context.Context, event Event) error
}

// Service 负责把业务事件转换为标准投递请求，并扇出到多个渠道。
type Service struct {
	directory UserDirectory
	channels  []Channel
}

func NewService(directory UserDirectory, channels ...Channel) *Service {
	copiedChannels := make([]Channel, 0, len(channels))
	copiedChannels = append(copiedChannels, channels...)
	return &Service{directory: directory, channels: copiedChannels}
}

func (service *Service) Publish(ctx context.Context, event Event) error {
	if ctx == nil {
		ctx = context.Background()
	}
	request := Request{
		Event:      event,
		Recipients: service.resolveRecipients(event.RecipientUserIDs),
	}
	var publishErr error
	for _, channel := range service.channels {
		if channel == nil {
			continue
		}
		if err := channel.Deliver(ctx, request); err != nil {
			publishErr = errors.Join(publishErr, fmt.Errorf("channel %s: %w", channel.Name(), err))
		}
	}
	return publishErr
}

func (service *Service) resolveRecipients(userIDs []string) []User {
	if service.directory == nil {
		return nil
	}
	seen := make(map[string]struct{}, len(userIDs))
	recipients := make([]User, 0, len(userIDs))
	for _, rawUserID := range userIDs {
		userID := strings.TrimSpace(rawUserID)
		if userID == "" {
			continue
		}
		if _, exists := seen[userID]; exists {
			continue
		}
		seen[userID] = struct{}{}
		user, ok := service.directory.FindUserByID(userID)
		if !ok {
			continue
		}
		recipients = append(recipients, user)
	}
	return recipients
}
