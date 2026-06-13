package provider

import (
	"context"
	"go-tiny-claw/internal/schema"
)

// LLMProvider 定义了与大模型通信的统一契约
type LLMProvider interface {
	Generate(ctx context.Context, messages []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error)
}
