package provider

import (
	"context"
	"go-tiny-claw/internal/schema"
)

const defaultMaxTokens int64 = 4096

// GenerateOptions 控制单次模型生成请求。
type GenerateOptions struct {
	MaxTokens int64
}

func normalizedMaxTokens(options GenerateOptions) int64 {
	if options.MaxTokens > 0 {
		return options.MaxTokens
	}
	return defaultMaxTokens
}

// LLMProvider 定义了与大模型通信的统一契约
type LLMProvider interface {
	Generate(ctx context.Context, messages []schema.Message, availableTools []schema.ToolDefinition, options GenerateOptions) (*schema.Message, error)
}
