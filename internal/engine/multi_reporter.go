package engine

import "context"

var _ Reporter = (*MultiReporter)(nil)

// MultiReporter 将同一个 Agent 生命周期事件转发给多个 Reporter。
type MultiReporter struct {
	reporters []Reporter
}

func NewMultiReporter(reporters ...Reporter) *MultiReporter {
	filtered := make([]Reporter, 0, len(reporters))
	for _, reporter := range reporters {
		if reporter != nil {
			filtered = append(filtered, reporter)
		}
	}
	return &MultiReporter{reporters: filtered}
}

func (r *MultiReporter) OnThinking(ctx context.Context) {
	for _, reporter := range r.reporters {
		reporter.OnThinking(ctx)
	}
}

func (r *MultiReporter) OnPlan(ctx context.Context, plan string) {
	for _, reporter := range r.reporters {
		reporter.OnPlan(ctx, plan)
	}
}

func (r *MultiReporter) OnToolCall(ctx context.Context, toolName string, args string) {
	for _, reporter := range r.reporters {
		reporter.OnToolCall(ctx, toolName, args)
	}
}

func (r *MultiReporter) OnToolResult(ctx context.Context, toolName string, output string, isError bool) {
	for _, reporter := range r.reporters {
		reporter.OnToolResult(ctx, toolName, output, isError)
	}
}

func (r *MultiReporter) OnMessage(ctx context.Context, content string) {
	for _, reporter := range r.reporters {
		reporter.OnMessage(ctx, content)
	}
}
