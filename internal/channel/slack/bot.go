package slackchannel

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"go-tiny-claw/internal/engine"

	"github.com/slack-go/slack/slackevents"
)

var leadingMentionPattern = regexp.MustCompile(`^\s*<@[^>]+>\s*`)

// AgentRunner 是 Slack Bot 调用 Agent 引擎所需的最小接口。
type AgentRunner interface {
	Run(ctx context.Context, userPrompt string, reporter engine.Reporter) error
}

// Bot 将 Slack 中的 app_mention 事件转发给 Agent。
type Bot struct {
	runner              AgentRunner
	sender              messageSender
	socket              socketConnection
	additionalReporters []engine.Reporter
}

func (b *Bot) handleAppMention(ctx context.Context, event *slackevents.AppMentionEvent) {
	if event.BotID != "" {
		return
	}

	threadTimestamp := event.ThreadTimeStamp
	if threadTimestamp == "" {
		threadTimestamp = event.TimeStamp
	}
	slackReporter := newReporter(b.sender, event.Channel, threadTimestamp)
	reporters := make([]engine.Reporter, 0, len(b.additionalReporters)+1)
	reporters = append(reporters, slackReporter)
	reporters = append(reporters, b.additionalReporters...)
	reporter := engine.NewMultiReporter(reporters...)

	prompt := strings.TrimSpace(leadingMentionPattern.ReplaceAllString(event.Text, ""))
	if prompt == "" {
		reporter.OnMessage(ctx, "请在 @机器人 后输入任务。")
		return
	}

	if err := b.runner.Run(ctx, prompt, reporter); err != nil {
		reporter.OnMessage(ctx, fmt.Sprintf("❌ Agent 执行失败: %v", err))
	}
}
