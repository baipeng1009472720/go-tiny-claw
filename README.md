# go-tiny-claw

一个用于学习 AI Agent 核心机制的 Go 项目，支持有界 Planning / Action 循环、并发工具调用、终端事件报告和 Slack Socket Mode 消息入口。

## Slack Socket Mode

### 1. 创建 Slack App

在 Slack App 管理后台完成以下配置：

1. 开启 **Socket Mode**。
2. 创建 App-Level Token，并授予 `connections:write` 权限，得到 `xapp-` 开头的 Token。
3. 在 **OAuth & Permissions** 中添加 Bot Token Scopes：
   - `app_mentions:read`
   - `chat:write`
4. 在 **Event Subscriptions** 中订阅 Bot Event：`app_mention`。
5. 将 App 安装到 Workspace，得到 `xoxb-` 开头的 Bot Token。
6. 将机器人邀请进需要使用的频道。

### 2. 配置环境变量

```bash
export ZHIPU_API_KEY="你的模型 API Key"
export SLACK_BOT_TOKEN="xoxb-..."
export SLACK_APP_TOKEN="xapp-..."
```

### 3. 启动

```bash
go run ./cmd/claw
```

连接成功后，在已邀请机器人的 Slack 频道中发送：

```text
@机器人 请读取当前工作区的 README.md 并概括项目功能
```

机器人会先发送“正在思考”占位消息，再将它更新为最多 5 步的执行规划，最后单独发送最终回答。工具调用和结果只显示在运行终端，不会刷屏 Slack。由其他 Bot 产生的 mention 会被忽略，以避免机器人之间循环回复。

## 本地验证

```bash
GOCACHE=/tmp/go-tiny-claw-gocache go test ./...
GOCACHE=/tmp/go-tiny-claw-gocache go vet ./...
```
