package notify

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type stubUserDirectory struct {
	users map[string]User
}

func (directory stubUserDirectory) FindUserByID(userID string) (User, bool) {
	user, ok := directory.users[userID]
	return user, ok
}

type stubChannel struct {
	name     string
	err      error
	requests []Request
}

func (channel *stubChannel) Name() string {
	return channel.name
}

func (channel *stubChannel) Deliver(_ context.Context, request Request) error {
	channel.requests = append(channel.requests, request)
	return channel.err
}

type stubMailSender struct {
	mails []Mail
	err   error
}

func (sender *stubMailSender) Send(_ context.Context, mail Mail) error {
	sender.mails = append(sender.mails, mail)
	return sender.err
}

type stubHTTPClient struct {
	requests []*http.Request
	body     string
	status   int
	err      error
}

func (client *stubHTTPClient) Do(request *http.Request) (*http.Response, error) {
	if client.err != nil {
		return nil, client.err
	}
	client.requests = append(client.requests, request)
	bodyBytes, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	client.body = string(bodyBytes)
	status := client.status
	if status == 0 {
		status = http.StatusAccepted
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(bytes.NewBufferString("ok"))}, nil
}

func TestServicePublishContinuesWhenChannelFails(t *testing.T) {
	failedChannel := &stubChannel{name: "webhook", err: errors.New("timeout")}
	successChannel := &stubChannel{name: "email"}
	service := NewService(stubUserDirectory{users: map[string]User{
		"user-1": {ID: "user-1", Name: "Alice", Email: "alice@example.com"},
	}}, failedChannel, successChannel)

	err := service.Publish(context.Background(), Event{
		Type:             EventTypeTicketAssigned,
		OrganizationID:   "org-1",
		TicketID:         "ticket-1",
		ActorID:          "agent-1",
		RecipientUserIDs: []string{"user-1"},
		Subject:          "工单已分派",
		Content:          "请尽快处理 ticket-1",
		CreatedAt:        time.Now(),
	})
	if err == nil {
		t.Fatalf("expected aggregated publish error when one channel fails")
	}
	if len(failedChannel.requests) != 1 {
		t.Fatalf("expected failed channel to be invoked once, got %d", len(failedChannel.requests))
	}
	if len(successChannel.requests) != 1 {
		t.Fatalf("expected success channel to still be invoked once, got %d", len(successChannel.requests))
	}
	if len(successChannel.requests[0].Recipients) != 1 || successChannel.requests[0].Recipients[0].Email != "alice@example.com" {
		t.Fatalf("expected recipients to be resolved before delivery, got %+v", successChannel.requests[0].Recipients)
	}
}

func TestEmailChannelSendsMailToResolvedRecipients(t *testing.T) {
	sender := &stubMailSender{}
	channel := NewEmailChannel(sender)

	err := channel.Deliver(context.Background(), Request{
		Event: Event{
			Type:      EventTypeTicketStatusChanged,
			TicketID:  "ticket-9",
			Subject:   "工单状态已更新",
			Content:   "ticket-9 已进入处理中",
			CreatedAt: time.Now(),
		},
		Recipients: []User{{ID: "user-1", Name: "Alice", Email: "alice@example.com"}, {ID: "user-2", Name: "NoMail"}},
	})
	if err != nil {
		t.Fatalf("deliver email: %v", err)
	}
	if len(sender.mails) != 1 {
		t.Fatalf("expected 1 mail to be sent, got %d", len(sender.mails))
	}
	if len(sender.mails[0].To) != 1 || sender.mails[0].To[0] != "alice@example.com" {
		t.Fatalf("expected only resolvable email recipient to be used, got %+v", sender.mails[0].To)
	}
	if sender.mails[0].Subject != "工单状态已更新" {
		t.Fatalf("expected subject to be propagated, got %q", sender.mails[0].Subject)
	}
}

func TestWebhookChannelPostsStructuredEvent(t *testing.T) {
	client := &stubHTTPClient{}
	channel := NewWebhookChannel([]string{"https://example.com/webhook"}, client, time.Second)
	err := channel.Deliver(context.Background(), Request{
		Event: Event{
			Type:             EventTypeAIEscalated,
			OrganizationID:   "org-1",
			TicketID:         "ticket-3",
			ActorID:          "user-1",
			RecipientUserIDs: []string{"user-1"},
			Subject:          "AI 已升级为人工跟进",
			Content:          "请关注 ticket-3",
			Metadata: map[string]string{
				"source": "feedback_unresolved",
			},
			CreatedAt: time.Now(),
		},
		Recipients: []User{{ID: "user-1", Name: "Alice", Email: "alice@example.com"}},
	})
	if err != nil {
		t.Fatalf("deliver webhook: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("expected 1 webhook request, got %d", len(client.requests))
	}
	for _, expected := range []string{"ai_escalated", "ticket-3", "feedback_unresolved", "alice@example.com"} {
		if !strings.Contains(client.body, expected) {
			t.Fatalf("expected webhook body to contain %q, got %s", expected, client.body)
		}
	}
}
