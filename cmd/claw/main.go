package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	slackchannel "go-tiny-claw/internal/channel/slack"
	"go-tiny-claw/internal/engine"
	"go-tiny-claw/internal/provider"
	"go-tiny-claw/internal/tools"
)

func main() {
	// 检查必要的环境变量
	if err := checkRequiredEnvVars(); err != nil {
		log.Fatal(err)
	}

	// 获取工作目录
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("无法获取当前工作目录: %v", err)
	}

	// 初始化 LLM 提供者
	llmProvider := provider.NewDouaoClaudeProvider("doubao-seed-code-preview-latest")

	// 初始化工具注册表并注册工具
	registry := tools.NewRegistry()
	registry.Register(tools.NewReadFileTool(workDir))
	registry.Register(tools.NewWriteFileTool(workDir))
	registry.Register(tools.NewBashTool(workDir))
	registry.Register(tools.NewEditFileTool(workDir))

	// 实例化并运行引擎
	eng := engine.NewAgentEngine(llmProvider, registry, workDir, true)
	bot := slackchannel.NewBot(
		eng,
		os.Getenv("SLACK_BOT_TOKEN"),
		os.Getenv("SLACK_APP_TOKEN"),
		engine.NewTerminalReporter(os.Stdout),
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("🚀 go-tiny-claw Slack Socket Mode 已启动，等待 @机器人 消息")
	if err := bot.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("Slack Bot 运行失败: %v", err)
	}
}

// 检查必要的环境变量
func checkRequiredEnvVars() error {
	for _, key := range []string{"ZHIPU_API_KEY", "SLACK_BOT_TOKEN", "SLACK_APP_TOKEN"} {
		if os.Getenv(key) == "" {
			return fmt.Errorf("请先导出 %s 环境变量", key)
		}
	}
	return nil
}
