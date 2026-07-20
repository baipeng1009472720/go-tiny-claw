package engine_test

import (
	"bytes"
	"context"
	"testing"

	"go-tiny-claw/internal/engine"
)

func TestMultiReporterForwardsEveryEvent(t *testing.T) {
	var first bytes.Buffer
	var second bytes.Buffer
	reporter := engine.NewMultiReporter(
		engine.NewTerminalReporter(&first),
		engine.NewTerminalReporter(&second),
	)
	ctx := context.Background()

	reporter.OnThinking(ctx)
	reporter.OnPlan(ctx, "1. 读取文件")
	reporter.OnToolCall(ctx, "read_file", `{"path":"README.md"}`)
	reporter.OnToolResult(ctx, "read_file", "content", false)
	reporter.OnMessage(ctx, "完成")

	if first.String() == "" {
		t.Fatal("第一个 Reporter 没有收到事件")
	}
	if second.String() != first.String() {
		t.Fatalf("两个 Reporter 收到的事件不同\nfirst:\n%s\nsecond:\n%s", first.String(), second.String())
	}
}
