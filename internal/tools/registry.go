package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"go-tiny-claw/internal/schema"
	"log"
)

// BaseTool 是所有具体工具必须实现的通用接口
type BaseTool interface {
	Name() string
	Definition() schema.ToolDefinition
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

// Registry 定义了工具的注册与分发执行接口
type Registry interface {

	// GetAvailableTools 返回当前系统挂载的所有可用工具的 Schema
	GetAvailableTools() []schema.ToolDefinition

	// Execute 实际执行模型请求的工具，并返回结果
	Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult

	Register(tool BaseTool)
}
type registryImpl struct {
	tools map[string]BaseTool
}

func NewRegistry() Registry {
	return registryImpl{tools: make(map[string]BaseTool)}
}

func (r registryImpl) Register(tool BaseTool) {
	name := tool.Name()
	if _, ok := r.tools[name]; ok {
		log.Printf("Tool %s already registered", name)
	}
	r.tools[name] = tool
	log.Printf("Tool %s registered", name)
}
func (r registryImpl) GetAvailableTools() []schema.ToolDefinition {
	var tools []schema.ToolDefinition
	for _, tool := range r.tools {
		tools = append(tools, tool.Definition())
	}
	return tools
}
func (r registryImpl) Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult {
	// 1. 路由查找：如果在注册表中找不到该工具，这是模型产生了幻觉，直接向模型抛出错误
	tool, ok := r.tools[call.Name]
	if !ok {
		err := fmt.Sprintf("Tool %s not registered", call.Name)
		return schema.ToolResult{
			ToolCallID: call.ID,
			Output:     err,
			IsError:    true}
	}
	// 2. 执行工具逻辑：将原始的 JSON 字节流直接丢给具体工具
	execute, err := tool.Execute(ctx, call.Arguments)
	if err != nil {
		errMsg := fmt.Sprintf("Error executing %s: %v", call.Name, err)
		return schema.ToolResult{ToolCallID: call.ID, Output: errMsg, IsError: true}
	}
	return schema.ToolResult{
		ToolCallID: call.ID,
		Output:     execute,
		IsError:    false,
	}
}
