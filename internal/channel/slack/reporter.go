package slackchannel

import (
	"context"
	"log"
	"sync"

	"go-tiny-claw/internal/engine"
)

var _ engine.Reporter = (*reporter)(nil)

type messageSender interface {
	SendMessage(ctx context.Context, channelID string, threadTimestamp string, text string) (string, error)
	UpdateMessage(ctx context.Context, channelID string, timestamp string, text string) error
}

type reporter struct {
	mu              sync.Mutex
	sender          messageSender
	channelID       string
	threadTimestamp string
	thinkingTS      string
}

func newReporter(sender messageSender, channelID string, threadTimestamp string) *reporter {
	return &reporter{
		sender:          sender,
		channelID:       channelID,
		threadTimestamp: threadTimestamp,
	}
}

func (r *reporter) OnThinking(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()

	timestamp, err := r.sender.SendMessage(ctx, r.channelID, r.threadTimestamp, "🧠 模型正在思考...")
	if err != nil {
		log.Printf("[Slack] 发送思考状态失败: %v", err)
		return
	}
	r.thinkingTS = timestamp
}

func (r *reporter) OnPlan(ctx context.Context, plan string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	text := "🧠 执行规划：\n" + plan
	if r.thinkingTS != "" {
		if err := r.sender.UpdateMessage(ctx, r.channelID, r.thinkingTS, text); err == nil {
			return
		} else {
			log.Printf("[Slack] 更新思考状态失败，改为发送新消息: %v", err)
		}
	}

	if _, err := r.sender.SendMessage(ctx, r.channelID, r.threadTimestamp, text); err != nil {
		log.Printf("[Slack] 发送规划失败: %v", err)
	}
}

func (*reporter) OnToolCall(context.Context, string, string) {}

func (*reporter) OnToolResult(context.Context, string, string, bool) {}

func (r *reporter) OnMessage(ctx context.Context, content string) {
	r.send(ctx, content)
}

func (r *reporter) send(ctx context.Context, text string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, err := r.sender.SendMessage(ctx, r.channelID, r.threadTimestamp, text); err != nil {
		log.Printf("[Slack] 发送消息失败: %v", err)
	}
}
