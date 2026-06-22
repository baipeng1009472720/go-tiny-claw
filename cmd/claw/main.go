// cmd/claw/main.go
package main

import (
	"context"
	"go-tiny-claw/internal/tools"
	"log"
	"os"

	"go-tiny-claw/internal/engine"
	"go-tiny-claw/internal/provider"
)

func main() {
	// 确保已设置 ZHIPU_API_KEY
	if os.Getenv("ZHIPU_API_KEY") == "" {
		log.Fatal("请先导出 ZHIPU_API_KEY 环境变量")
	}

	workDir, _ := os.Getwd()

	// 1. 初始化真实的 Provider大脑
	// 这里你可以任意切换 NewDouaoClaudeProvider 或 NewZhipuOpenAIProvider，效果完全一致！
	llmProvider := provider.NewDouaoClaudeProvider("doubao-seed-code-preview-latest")

	// 3. 初始化真实的 Tool Registry
	registry := tools.NewRegistry()

	// 4. 将真实的 ReadFile 工具挂载到注册表中

	readFileTool := tools.NewReadFileTool(workDir)

	registry.Register(readFileTool)

	// 5. 实例化并运行引擎，开启 EnableThinking = true (开启慢思考阶段！)
	eng := engine.NewAgentEngine(llmProvider, registry, workDir, false)

	// 6. 下发一个必须通过真实工具才能完成的任务
	prompt := "请调用工具读取一下当前工作区目录下 hello.txt 文件的内容，并用一句话向我总结它说了什么。"

	err := eng.Run(context.Background(), prompt)
	if err != nil {
		log.Fatalf("引擎运行崩溃: %v", err)
	}
}
