package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HTTPDoer 抽象 HTTP 客户端，方便测试 Webhook 渠道。
type HTTPDoer interface {
	Do(request *http.Request) (*http.Response, error)
}

type WebhookChannel struct {
	urls    []string
	client  HTTPDoer
	timeout time.Duration
}

func NewWebhookChannel(urls []string, client HTTPDoer, timeout time.Duration) *WebhookChannel {
	copiedURLs := make([]string, 0, len(urls))
	for _, url := range urls {
		trimmed := strings.TrimSpace(url)
		if trimmed == "" {
			continue
		}
		copiedURLs = append(copiedURLs, trimmed)
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	return &WebhookChannel{urls: copiedURLs, client: client, timeout: timeout}
}

func (channel *WebhookChannel) Name() string {
	return "webhook"
}

func (channel *WebhookChannel) Deliver(ctx context.Context, request Request) error {
	if channel == nil || len(channel.urls) == 0 {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, channel.timeout)
	defer cancel()

	payload, err := json.Marshal(struct {
		Type       EventType `json:"type"`
		Event      Event     `json:"event"`
		Recipients []User    `json:"recipients"`
	}{
		Type:       request.Event.Type,
		Event:      request.Event,
		Recipients: request.Recipients,
	})
	if err != nil {
		return err
	}

	var deliveryErr error
	for _, url := range channel.urls {
		httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			deliveryErr = errors.Join(deliveryErr, fmt.Errorf("build webhook request for %s: %w", url, err))
			continue
		}
		httpRequest.Header.Set("Content-Type", "application/json")
		response, err := channel.client.Do(httpRequest)
		if err != nil {
			deliveryErr = errors.Join(deliveryErr, fmt.Errorf("post webhook %s: %w", url, err))
			continue
		}
		_ = response.Body.Close()
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			deliveryErr = errors.Join(deliveryErr, fmt.Errorf("post webhook %s: unexpected status %d", url, response.StatusCode))
		}
	}
	return deliveryErr
}
