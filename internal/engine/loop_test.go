package engine

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"go-tiny-claw/internal/provider"
	"go-tiny-claw/internal/schema"
	"go-tiny-claw/internal/tools"
)

func TestAgentEnginePlansOnceWithBoundedTokens(t *testing.T) {
	llm := &recordingProvider{responses: []schema.Message{
		{Role: schema.RoleAssistant, Content: "1. 读取 README。\n读取 README。\n2. 总结项目。"},
		{Role: schema.RoleAssistant, Content: "项目介绍完成"},
	}}
	registry := &recordingRegistry{definitions: []schema.ToolDefinition{{Name: "read_file"}}}
	reporter := &recordingReporter{}
	engine := NewAgentEngine(llm, registry, ".", true)

	if err := engine.Run(context.Background(), "介绍项目", reporter); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(llm.calls) != 2 {
		t.Fatalf("Generate() 调用次数 = %d, want 2", len(llm.calls))
	}
	if len(llm.calls[0].tools) != 0 || llm.calls[0].options.MaxTokens != planningMaxTokens {
		t.Fatalf("规划请求 = %#v, want 无工具且 MaxTokens=%d", llm.calls[0], planningMaxTokens)
	}
	if len(llm.calls[1].tools) != 1 || llm.calls[1].options.MaxTokens != actionMaxTokens {
		t.Fatalf("行动请求 = %#v, want 1 个工具且 MaxTokens=%d", llm.calls[1], actionMaxTokens)
	}
	if reporter.thinkingCount != 1 {
		t.Fatalf("OnThinking() 次数 = %d, want 1", reporter.thinkingCount)
	}
	wantPlan := "1. 读取 README\n2. 总结项目"
	if len(reporter.plans) != 1 || reporter.plans[0] != wantPlan {
		t.Fatalf("规划 = %#v, want %q", reporter.plans, wantPlan)
	}
	if got := llm.calls[1].messages[len(llm.calls[1].messages)-1].Content; got != wantPlan {
		t.Fatalf("行动上下文中的规划 = %q, want %q", got, wantPlan)
	}
	if len(reporter.messages) != 1 || reporter.messages[0] != "项目介绍完成" {
		t.Fatalf("最终消息 = %#v", reporter.messages)
	}
}

func TestAgentEngineOnlyReportsContentWithoutToolCalls(t *testing.T) {
	llm := &recordingProvider{responses: []schema.Message{
		{
			Role:    schema.RoleAssistant,
			Content: "我先读取文件",
			ToolCalls: []schema.ToolCall{{
				ID:        "call-1",
				Name:      "read_file",
				Arguments: []byte(`{"path":"README.md"}`),
			}},
		},
		{Role: schema.RoleAssistant, Content: "这是最终介绍"},
	}}
	registry := &recordingRegistry{definitions: []schema.ToolDefinition{{Name: "read_file"}}}
	reporter := &recordingReporter{}
	engine := NewAgentEngine(llm, registry, ".", false)

	if err := engine.Run(context.Background(), "介绍项目", reporter); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(reporter.messages) != 1 || reporter.messages[0] != "这是最终介绍" {
		t.Fatalf("OnMessage() 内容 = %#v, want 仅最终介绍", reporter.messages)
	}
}

func TestAgentEngineStopsAfterMaximumActionTurns(t *testing.T) {
	toolResponse := schema.Message{
		Role: schema.RoleAssistant,
		ToolCalls: []schema.ToolCall{{
			ID:        "call-loop",
			Name:      "read_file",
			Arguments: []byte(`{"path":"README.md"}`),
		}},
	}
	responses := make([]schema.Message, maxActionTurns)
	for i := range responses {
		responses[i] = toolResponse
	}
	llm := &recordingProvider{
		responses:      responses,
		exhaustedError: errors.New("fake provider 收到第 11 次调用"),
	}
	registry := &recordingRegistry{definitions: []schema.ToolDefinition{{Name: "read_file"}}}
	engine := NewAgentEngine(llm, registry, ".", false)

	err := engine.Run(context.Background(), "循环调用", &recordingReporter{})
	if err == nil || !strings.Contains(err.Error(), "达到最大执行轮数 10") {
		t.Fatalf("Run() error = %v, want 最大执行轮数错误", err)
	}
	if len(llm.calls) != maxActionTurns {
		t.Fatalf("Generate() 调用次数 = %d, want %d", len(llm.calls), maxActionTurns)
	}
}

type providerCall struct {
	messages []schema.Message
	tools    []schema.ToolDefinition
	options  provider.GenerateOptions
}

type recordingProvider struct {
	responses      []schema.Message
	exhaustedError error
	calls          []providerCall
}

func (p *recordingProvider) Generate(_ context.Context, messages []schema.Message, availableTools []schema.ToolDefinition, options provider.GenerateOptions) (*schema.Message, error) {
	p.calls = append(p.calls, providerCall{
		messages: append([]schema.Message(nil), messages...),
		tools:    append([]schema.ToolDefinition(nil), availableTools...),
		options:  options,
	})
	index := len(p.calls) - 1
	if index >= len(p.responses) {
		if p.exhaustedError != nil {
			return nil, p.exhaustedError
		}
		return nil, errors.New("fake provider 没有更多响应")
	}
	response := p.responses[index]
	return &response, nil
}

type recordingRegistry struct {
	mu          sync.Mutex
	definitions []schema.ToolDefinition
	executions  []schema.ToolCall
}

func (r *recordingRegistry) GetAvailableTools() []schema.ToolDefinition {
	return append([]schema.ToolDefinition(nil), r.definitions...)
}

func (r *recordingRegistry) Execute(_ context.Context, call schema.ToolCall) schema.ToolResult {
	r.mu.Lock()
	r.executions = append(r.executions, call)
	r.mu.Unlock()
	return schema.ToolResult{ToolCallID: call.ID, Output: "file content"}
}

func (r *recordingRegistry) Register(tools.BaseTool) {}

type recordingReporter struct {
	mu            sync.Mutex
	thinkingCount int
	plans         []string
	messages      []string
}

func (r *recordingReporter) OnThinking(context.Context) {
	r.mu.Lock()
	r.thinkingCount++
	r.mu.Unlock()
}

func (r *recordingReporter) OnPlan(_ context.Context, plan string) {
	r.mu.Lock()
	r.plans = append(r.plans, plan)
	r.mu.Unlock()
}

func (*recordingReporter) OnToolCall(context.Context, string, string) {}

func (*recordingReporter) OnToolResult(context.Context, string, string, bool) {}

func (r *recordingReporter) OnMessage(_ context.Context, content string) {
	r.mu.Lock()
	r.messages = append(r.messages, content)
	r.mu.Unlock()
}
