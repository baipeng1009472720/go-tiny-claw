package engine_test

import (
	"bytes"
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-tiny-claw/internal/engine"
)

func TestTerminalReporterWritesEngineEvents(t *testing.T) {
	var output bytes.Buffer
	reporter := engine.NewTerminalReporter(&output)
	ctx := context.Background()

	reporter.OnThinking(ctx)
	reporter.OnPlan(ctx, "1. 读取文件\n2. 总结项目")
	reporter.OnToolCall(ctx, "read_file", `{"path":"a.txt"}`)
	reporter.OnToolResult(ctx, "read_file", "hello", false)
	reporter.OnToolResult(ctx, "bash", "permission denied", true)
	reporter.OnMessage(ctx, "完成")

	want := "" +
		"🧠 [Thinking] 模型正在思考...\n" +
		"🧠 [Plan] 1. 读取文件\n2. 总结项目\n" +
		"🛠️ [Tool Call] read_file {\"path\":\"a.txt\"}\n" +
		"✅ [Tool Result] read_file: hello\n" +
		"❌ [Tool Result] bash: permission denied\n" +
		"🤖 [Assistant] 完成\n"
	if got := output.String(); got != want {
		t.Fatalf("终端输出不符合预期\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestTerminalReporterSerializesConcurrentWrites(t *testing.T) {
	writer := &overlapWriter{}
	reporter := engine.NewTerminalReporter(writer)
	ctx := context.Background()

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reporter.OnMessage(ctx, "并发消息")
		}()
	}
	wg.Wait()

	if writer.overlapped.Load() {
		t.Fatal("检测到并发写入 io.Writer")
	}
}

type overlapWriter struct {
	active     atomic.Int32
	overlapped atomic.Bool
}

func (w *overlapWriter) Write(p []byte) (int, error) {
	if w.active.Add(1) > 1 {
		w.overlapped.Store(true)
	}
	time.Sleep(time.Millisecond)
	w.active.Add(-1)
	return len(p), nil
}
