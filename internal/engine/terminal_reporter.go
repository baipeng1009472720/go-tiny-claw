package engine

import (
	"context"
	"fmt"
	"io"
	"sync"
)

var _ Reporter = (*TerminalReporter)(nil)

// TerminalReporter 将 Agent 生命周期事件输出到终端或其他 io.Writer。
type TerminalReporter struct {
	mu     sync.Mutex
	writer io.Writer
}

// NewTerminalReporter 创建一个并发安全的终端 Reporter。
func NewTerminalReporter(writer io.Writer) *TerminalReporter {
	return &TerminalReporter{writer: writer}
}

func (r *TerminalReporter) OnThinking(context.Context) {
	r.write("🧠 [Thinking] 模型正在思考...\n")
}

func (r *TerminalReporter) OnPlan(_ context.Context, plan string) {
	r.write("🧠 [Plan] %s\n", plan)
}

func (r *TerminalReporter) OnToolCall(_ context.Context, toolName string, args string) {
	r.write("🛠️ [Tool Call] %s %s\n", toolName, args)
}

func (r *TerminalReporter) OnToolResult(_ context.Context, toolName string, output string, isError bool) {
	status := "✅"
	if isError {
		status = "❌"
	}
	r.write("%s [Tool Result] %s: %s\n", status, toolName, output)
}

func (r *TerminalReporter) OnMessage(_ context.Context, content string) {
	r.write("🤖 [Assistant] %s\n", content)
}

func (r *TerminalReporter) write(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, _ = fmt.Fprintf(r.writer, format, args...)
}
