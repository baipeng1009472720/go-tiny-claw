package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go-tiny-claw/internal/engine"
	"go-tiny-claw/internal/provider"
	"go-tiny-claw/internal/tools"
)

const (
	// 任务描述常量
	taskPrompt = `请修改当前目录下的 internal/example/server_example.go 文件，
将 "TODO: 增加鉴权逻辑" 下面的 if 语句替换为：
if user == nil {
    fmt.Println("Forbidden!")
    return
}`
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
	eng := engine.NewAgentEngine(llmProvider, registry, workDir, false)

	// 执行任务
	if err := eng.Run(context.Background(), taskPrompt); err != nil {
		log.Fatalf("引擎运行崩溃: %v", err)
	}
}

// 检查必要的环境变量
func checkRequiredEnvVars() error {
	if os.Getenv("ZHIPU_API_KEY") == "" {
		return fmt.Errorf("请先导出 ZHIPU_API_KEY 环境变量")
	}
	return nil
}
