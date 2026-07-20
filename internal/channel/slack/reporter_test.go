package slackchannel

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestReporterUpdatesThinkingMessageAndOnlySendsFinalAnswer(t *testing.T) {
	sender := &recordingSender{nextTimestamp: "1710000001.000200"}
	reporter := newReporter(sender, "C123", "1710000000.000100")
	ctx := context.Background()

	reporter.OnThinking(ctx)
	reporter.OnPlan(ctx, "1. 读取文件\n2. 总结项目")
	reporter.OnToolCall(ctx, "read_file", `{"path":"a.txt"}`)
	reporter.OnToolResult(ctx, "read_file", "hello", false)
	reporter.OnToolResult(ctx, "bash", "permission denied", true)
	reporter.OnMessage(ctx, "任务完成")

	wantMessages := []sentMessage{
		{channelID: "C123", threadTimestamp: "1710000000.000100", text: "🧠 模型正在思考..."},
		{channelID: "C123", threadTimestamp: "1710000000.000100", text: "任务完成"},
	}
	if !reflect.DeepEqual(sender.messages, wantMessages) {
		t.Fatalf("Slack 发送消息不符合预期\nwant: %#v\ngot:  %#v", wantMessages, sender.messages)
	}

	wantUpdates := []updatedMessage{
		{channelID: "C123", timestamp: "1710000001.000200", text: "🧠 执行规划：\n1. 读取文件\n2. 总结项目"},
	}
	if !reflect.DeepEqual(sender.updates, wantUpdates) {
		t.Fatalf("Slack 更新消息不符合预期\nwant: %#v\ngot:  %#v", wantUpdates, sender.updates)
	}
}

func TestReporterSendsPlanWhenThinkingMessageCannotBeUpdated(t *testing.T) {
	sender := &recordingSender{
		nextTimestamp: "1710000001.000200",
		updateError:   errors.New("update failed"),
	}
	reporter := newReporter(sender, "C123", "1710000000.000100")
	ctx := context.Background()

	reporter.OnThinking(ctx)
	reporter.OnPlan(ctx, "1. 读取文件")

	want := []sentMessage{
		{channelID: "C123", threadTimestamp: "1710000000.000100", text: "🧠 模型正在思考..."},
		{channelID: "C123", threadTimestamp: "1710000000.000100", text: "🧠 执行规划：\n1. 读取文件"},
	}
	if !reflect.DeepEqual(sender.messages, want) {
		t.Fatalf("降级消息不符合预期\nwant: %#v\ngot:  %#v", want, sender.messages)
	}
}

type sentMessage struct {
	channelID       string
	threadTimestamp string
	text            string
}

type updatedMessage struct {
	channelID string
	timestamp string
	text      string
}

type recordingSender struct {
	messages      []sentMessage
	updates       []updatedMessage
	nextTimestamp string
	sendError     error
	updateError   error
}

func (s *recordingSender) SendMessage(_ context.Context, channelID string, threadTimestamp string, text string) (string, error) {
	s.messages = append(s.messages, sentMessage{
		channelID:       channelID,
		threadTimestamp: threadTimestamp,
		text:            text,
	})
	return s.nextTimestamp, s.sendError
}

func (s *recordingSender) UpdateMessage(_ context.Context, channelID string, timestamp string, text string) error {
	s.updates = append(s.updates, updatedMessage{
		channelID: channelID,
		timestamp: timestamp,
		text:      text,
	})
	return s.updateError
}
