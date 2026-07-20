package slackchannel

import (
	"context"
	"log"

	"go-tiny-claw/internal/engine"

	goslack "github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

type socketConnection interface {
	EventStream() <-chan socketmode.Event
	Ack(request socketmode.Request)
	RunContext(ctx context.Context) error
}

type managedSocketConnection struct {
	client *socketmode.Client
}

func (c *managedSocketConnection) EventStream() <-chan socketmode.Event {
	return c.client.Events
}

func (c *managedSocketConnection) Ack(request socketmode.Request) {
	c.client.Ack(request)
}

func (c *managedSocketConnection) RunContext(ctx context.Context) error {
	return c.client.RunContext(ctx)
}

// NewBot 创建一个只处理 app_mention 的 Slack Socket Mode Bot。
func NewBot(runner AgentRunner, botToken string, appToken string, additionalReporters ...engine.Reporter) *Bot {
	apiClient := goslack.New(
		botToken,
		goslack.OptionAppLevelToken(appToken),
	)
	socketClient := socketmode.New(apiClient)

	return &Bot{
		runner:              runner,
		sender:              &slackMessageSender{client: apiClient},
		socket:              &managedSocketConnection{client: socketClient},
		additionalReporters: append([]engine.Reporter(nil), additionalReporters...),
	}
}

// Run 建立 Socket Mode 连接，并持续处理 Slack 事件直到上下文结束。
func (b *Bot) Run(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go b.consumeEvents(runCtx)
	return b.socket.RunContext(runCtx)
}

func (b *Bot) consumeEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-b.socket.EventStream():
			if !ok {
				return
			}
			b.handleSocketEvent(ctx, event)
		}
	}
}

func (b *Bot) handleSocketEvent(ctx context.Context, event socketmode.Event) {
	switch event.Type {
	case socketmode.EventTypeConnected:
		log.Println("[Slack] Socket Mode 已连接")
	case socketmode.EventTypeConnectionError:
		log.Println("[Slack] Socket Mode 连接失败，等待客户端重试")
	case socketmode.EventTypeEventsAPI:
		if event.Request != nil {
			b.socket.Ack(*event.Request)
		}

		eventsAPIEvent, ok := event.Data.(slackevents.EventsAPIEvent)
		if !ok || eventsAPIEvent.Type != slackevents.CallbackEvent {
			return
		}
		mention, ok := eventsAPIEvent.InnerEvent.Data.(*slackevents.AppMentionEvent)
		if !ok {
			return
		}

		go b.handleAppMention(ctx, mention)
	}
}
