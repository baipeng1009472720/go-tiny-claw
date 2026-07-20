package slackchannel

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"

	"go-tiny-claw/internal/engine"

	"github.com/slack-go/slack/slackevents"
)

func TestBotAlsoReportsEventsToTerminal(t *testing.T) {
	var terminal bytes.Buffer
	sender := &recordingSender{}
	runner := &recordingRunner{response: "Agent 回答"}
	bot := &Bot{
		runner:              runner,
		sender:              sender,
		additionalReporters: []engine.Reporter{engine.NewTerminalReporter(&terminal)},
	}

	bot.handleAppMention(context.Background(), &slackevents.AppMentionEvent{
		Text:      "<@U123> 执行任务",
		Channel:   "C123",
		TimeStamp: "1710000000.000100",
	})

	if got := terminal.String(); got != "🤖 [Assistant] Agent 回答\n" {
		t.Fatalf("终端输出 = %q", got)
	}
	if len(sender.messages) != 1 || sender.messages[0].text != "Agent 回答" {
		t.Fatalf("Slack 输出 = %#v", sender.messages)
	}
}

func TestBotHandlesAppMentionInSlackThread(t *testing.T) {
	tests := []struct {
		name            string
		threadTimestamp string
		wantThread      string
	}{
		{name: "首次提及创建线程", wantThread: "1710000000.000100"},
		{name: "线程内提及继续原线程", threadTimestamp: "1709999999.000090", wantThread: "1709999999.000090"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &recordingSender{}
			runner := &recordingRunner{response: "Agent 回答"}
			bot := &Bot{runner: runner, sender: sender}

			bot.handleAppMention(context.Background(), &slackevents.AppMentionEvent{
				Text:            "  <@U123>  请读取文件  ",
				Channel:         "C123",
				TimeStamp:       "1710000000.000100",
				ThreadTimeStamp: tt.threadTimestamp,
			})

			if !reflect.DeepEqual(runner.prompts, []string{"请读取文件"}) {
				t.Fatalf("传给 Agent 的提示词错误: %#v", runner.prompts)
			}
			wantMessages := []sentMessage{{
				channelID:       "C123",
				threadTimestamp: tt.wantThread,
				text:            "Agent 回答",
			}}
			if !reflect.DeepEqual(sender.messages, wantMessages) {
				t.Fatalf("Slack 线程回复错误\nwant: %#v\ngot:  %#v", wantMessages, sender.messages)
			}
		})
	}
}

func TestBotRejectsEmptyAppMention(t *testing.T) {
	sender := &recordingSender{}
	runner := &recordingRunner{}
	bot := &Bot{runner: runner, sender: sender}

	bot.handleAppMention(context.Background(), &slackevents.AppMentionEvent{
		Text:      "<@U123>   ",
		Channel:   "C123",
		TimeStamp: "1710000000.000100",
	})

	if len(runner.prompts) != 0 {
		t.Fatalf("空提示不应调用 Agent: %#v", runner.prompts)
	}
	want := []sentMessage{{
		channelID:       "C123",
		threadTimestamp: "1710000000.000100",
		text:            "请在 @机器人 后输入任务。",
	}}
	if !reflect.DeepEqual(sender.messages, want) {
		t.Fatalf("空提示回复错误\nwant: %#v\ngot:  %#v", want, sender.messages)
	}
}

func TestBotReportsAgentError(t *testing.T) {
	sender := &recordingSender{}
	runner := &recordingRunner{err: errors.New("model unavailable")}
	bot := &Bot{runner: runner, sender: sender}

	bot.handleAppMention(context.Background(), &slackevents.AppMentionEvent{
		Text:      "<@U123> 执行任务",
		Channel:   "C123",
		TimeStamp: "1710000000.000100",
	})

	want := []sentMessage{{
		channelID:       "C123",
		threadTimestamp: "1710000000.000100",
		text:            "❌ Agent 执行失败: model unavailable",
	}}
	if !reflect.DeepEqual(sender.messages, want) {
		t.Fatalf("Agent 错误回复不符合预期\nwant: %#v\ngot:  %#v", want, sender.messages)
	}
}

func TestBotIgnoresBotGeneratedMention(t *testing.T) {
	sender := &recordingSender{}
	runner := &recordingRunner{}
	bot := &Bot{runner: runner, sender: sender}

	bot.handleAppMention(context.Background(), &slackevents.AppMentionEvent{
		Text:      "<@U123> 不应执行",
		Channel:   "C123",
		TimeStamp: "1710000000.000100",
		BotID:     "B456",
	})

	if len(runner.prompts) != 0 {
		t.Fatalf("Bot 生成的 mention 不应调用 Agent: %#v", runner.prompts)
	}
	if len(sender.messages) != 0 {
		t.Fatalf("Bot 生成的 mention 不应产生回复: %#v", sender.messages)
	}
}

type recordingRunner struct {
	prompts  []string
	response string
	err      error
}

func (r *recordingRunner) Run(ctx context.Context, prompt string, reporter engine.Reporter) error {
	r.prompts = append(r.prompts, prompt)
	if r.response != "" {
		reporter.OnMessage(ctx, r.response)
	}
	return r.err
}
