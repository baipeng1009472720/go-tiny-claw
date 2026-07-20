package slackchannel

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

func TestBotRunAcknowledgesAndHandlesAppMention(t *testing.T) {
	connection := newFakeSocketConnection()
	sender := &notifyingSender{messages: make(chan sentMessage, 1)}
	runner := &recordingRunner{response: "Agent 回答"}
	bot := &Bot{runner: runner, sender: sender, socket: connection}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- bot.Run(ctx)
	}()

	connection.events <- socketmode.Event{
		Type: socketmode.EventTypeEventsAPI,
		Data: slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: "app_mention",
				Data: &slackevents.AppMentionEvent{
					Text:      "<@U123> 执行任务",
					Channel:   "C123",
					TimeStamp: "1710000000.000100",
				},
			},
		},
		Request: &socketmode.Request{EnvelopeID: "env-123"},
	}

	select {
	case envelopeID := <-connection.acks:
		if envelopeID != "env-123" {
			t.Fatalf("ACK 信封错误: %s", envelopeID)
		}
	case <-time.After(time.Second):
		t.Fatal("未在超时前 ACK Slack 事件")
	}

	select {
	case message := <-sender.messages:
		want := sentMessage{channelID: "C123", threadTimestamp: "1710000000.000100", text: "Agent 回答"}
		if message != want {
			t.Fatalf("Slack 回复错误\nwant: %#v\ngot:  %#v", want, message)
		}
	case <-time.After(time.Second):
		t.Fatal("未在超时前处理 app_mention")
	}

	cancel()
	if err := <-errCh; !errors.Is(err, context.Canceled) {
		t.Fatalf("Bot.Run() error = %v, want context.Canceled", err)
	}
}

type fakeSocketConnection struct {
	events chan socketmode.Event
	acks   chan string
}

func newFakeSocketConnection() *fakeSocketConnection {
	return &fakeSocketConnection{
		events: make(chan socketmode.Event, 1),
		acks:   make(chan string, 1),
	}
}

func (c *fakeSocketConnection) EventStream() <-chan socketmode.Event {
	return c.events
}

func (c *fakeSocketConnection) Ack(request socketmode.Request) {
	c.acks <- request.EnvelopeID
}

func (c *fakeSocketConnection) RunContext(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

type notifyingSender struct {
	messages chan sentMessage
}

func (s *notifyingSender) SendMessage(_ context.Context, channelID string, threadTimestamp string, text string) (string, error) {
	s.messages <- sentMessage{channelID: channelID, threadTimestamp: threadTimestamp, text: text}
	return "1710000001.000200", nil
}

func (*notifyingSender) UpdateMessage(context.Context, string, string, string) error { return nil }
