package engine

import (
	"context"
	"fmt"
	"go-tiny-claw/internal/provider"
	"go-tiny-claw/internal/schema"
	"go-tiny-claw/internal/tools"
	"log"
	"sync"
)

// AgentEngine 是微型 OS 的核心驱动
type AgentEngine struct {
	provider provider.LLMProvider
	registry tools.Registry
	// WorkDir (工作区): 借鉴 OpenClaw 的理念，Agent 必须有一个明确的物理边界
	WorkDir        string
	EnableThinking bool // 【新增】慢思考模式开关
}

func NewAgentEngine(provider provider.LLMProvider, registry tools.Registry, workDir string, enableThinking bool) *AgentEngine {
	return &AgentEngine{
		provider:       provider,
		registry:       registry,
		WorkDir:        workDir,
		EnableThinking: enableThinking,
	}
}

// Run 启动 Agent 的生命周期
func (e *AgentEngine) Run(ctx context.Context, userPrompt string, reporter Reporter) error {
	log.Printf("[Engine] 引擎启动，锁定工作区: %s\n", e.WorkDir)
	log.Printf("[Engine] 慢思考模式 (Thinking Phase): %v\n", e.EnableThinking)
	contextHistory := []schema.Message{
		{
			Role:    schema.RoleSystem,
			Content: "You are go-tiny-claw, an expert coding assistant. You have full access to tools in the workspace.",
		},
		{
			Role:    schema.RoleUser,
			Content: userPrompt,
		},
	}
	availableTools := e.registry.GetAvailableTools()

	if e.EnableThinking {
		if reporter != nil {
			reporter.OnThinking(ctx)
		}

		planningHistory := append([]schema.Message(nil), contextHistory...)
		planningHistory = append(planningHistory, schema.Message{
			Role:    schema.RoleUser,
			Content: planningInstruction,
		})
		planningResp, err := e.provider.Generate(ctx, planningHistory, nil, provider.GenerateOptions{MaxTokens: planningMaxTokens})
		if err != nil {
			return fmt.Errorf("Planning 阶段生成失败: %w", err)
		}

		plan := normalizePlan(planningResp.Content)
		if reporter != nil {
			reporter.OnPlan(ctx, plan)
		}
		contextHistory = append(contextHistory, schema.Message{
			Role:    schema.RoleAssistant,
			Content: plan,
		})
	}

	for turn := 1; turn <= maxActionTurns; turn++ {
		log.Printf("========== [Action Turn %d/%d] 开始 ==========\n", turn, maxActionTurns)
		actionResp, err := e.provider.Generate(ctx, contextHistory, availableTools, provider.GenerateOptions{MaxTokens: actionMaxTokens})
		if err != nil {
			return fmt.Errorf("Action 阶段生成失败: %w", err)
		}
		contextHistory = append(contextHistory, *actionResp)

		if len(actionResp.ToolCalls) == 0 {
			if actionResp.Content != "" && reporter != nil {
				reporter.OnMessage(ctx, actionResp.Content)
			}
			log.Println("[Engine] 模型未请求调用工具，任务宣告完成。")
			return nil
		}
		if turn == maxActionTurns {
			return fmt.Errorf("达到最大执行轮数 %d，任务仍在请求工具调用", maxActionTurns)
		}

		obsMsgList := make([]schema.Message, len(actionResp.ToolCalls))
		var wg sync.WaitGroup
		log.Printf("[Engine] 模型请求调用 %d 个工具...\n", len(actionResp.ToolCalls))

		ch := make(chan struct{}, 5)
		for i, toolCall := range actionResp.ToolCalls {
			ch <- struct{}{}
			wg.Add(1)

			go func(idx int, call schema.ToolCall) {
				defer wg.Done()
				defer func() { <-ch }()

				if reporter != nil {
					reporter.OnToolCall(ctx, call.Name, string(call.Arguments))
				}

				result := e.registry.Execute(ctx, call)

				if reporter != nil {
					displayOutput := result.Output
					if len(displayOutput) > 200 {
						displayOutput = displayOutput[:200] + "... (已截断)"
					}
					reporter.OnToolResult(ctx, call.Name, displayOutput, result.IsError)
				}
				if result.IsError {
					log.Printf(" -> [Go-%d] ❌ 工具执行报错: %s\n", idx, result.Output)
				} else {
					log.Printf(" -> [Go-%d] ✅ 工具执行成功 (返回 %d 字节)\n", idx, len(result.Output))
				}
				obsMsgList[idx] = schema.Message{
					Role:       schema.RoleUser,
					Content:    result.Output,
					ToolCallID: call.ID,
				}
			}(i, toolCall)
		}
		wg.Wait()
		log.Println("[Engine] 所有并发工具执行完毕，开始聚合观察结果 (Observation)...")
		contextHistory = append(contextHistory, obsMsgList...)
	}

	return fmt.Errorf("达到最大执行轮数 %d", maxActionTurns)
}
