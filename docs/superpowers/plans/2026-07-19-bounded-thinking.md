# Bounded Thinking Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将每轮无限慢思考改为一次 512-token 有界规划，并在终端和 Slack 同时展示规划与最终回答。

**Architecture:** Engine 先执行一次无工具 Planning，再进入最多 10 轮的有工具 Action 循环。规划经独立函数按行去重、截断为 5 项；Reporter 通过新增的 `OnPlan` 展示规划，Slack 用消息更新时间戳把“正在思考”占位消息替换成规划。

**Tech Stack:** Go 1.24、Anthropic Go SDK、OpenAI Go SDK v3、slack-go v0.17.3、标准库 `testing`

## Global Constraints

- 每个任务只规划一次。
- 规划请求最大输出为 512 tokens，整理后最多 5 条。
- Action 请求最大输出为 4096 tokens，最多执行 10 轮。
- 仅无工具调用的 Action 文本属于最终回答。
- 终端展示规划和工具细节；Slack 只展示规划与最终回答。
- 保留工作区已有修改；未经用户授权不执行 `git add`、`commit`、`tag` 或 `push`。

---

### Task 1: 显式控制 Provider 输出上限

**Files:**
- Modify: `internal/provider/interface.go`
- Modify: `internal/provider/claude.go`
- Modify: `internal/provider/openai.go`
- Test: `internal/engine/loop_test.go`（Task 3 的 fake provider 会验证传入值）

**Interfaces:**
- Produces: `provider.GenerateOptions{MaxTokens int64}`
- Produces: `LLMProvider.Generate(context.Context, []schema.Message, []schema.ToolDefinition, provider.GenerateOptions) (*schema.Message, error)`

- [ ] **Step 1: 修改接口并让编译失败暴露全部调用点**

```go
type GenerateOptions struct {
    MaxTokens int64
}

type LLMProvider interface {
    Generate(ctx context.Context, messages []schema.Message, availableTools []schema.ToolDefinition, options GenerateOptions) (*schema.Message, error)
}
```

- [ ] **Step 2: 运行编译验证旧实现不满足新接口**

Run: `env GOCACHE=/tmp/go-tiny-claw-gocache go test ./internal/provider ./internal/engine`

Expected: FAIL，错误包含 `not enough arguments in call to ... Generate` 或 Provider 方法签名不匹配。

- [ ] **Step 3: 在 Claude 与 OpenAI 请求中应用 `MaxTokens`**

```go
func normalizedMaxTokens(options GenerateOptions) int64 {
    if options.MaxTokens > 0 {
        return options.MaxTokens
    }
    return 4096
}
```

Claude 使用 `MaxTokens: normalizedMaxTokens(options)`；OpenAI 使用 `MaxTokens: openai.Int(normalizedMaxTokens(options))`。

- [ ] **Step 4: 运行 Provider 编译测试**

Run: `env GOCACHE=/tmp/go-tiny-claw-gocache go test ./internal/provider`

Expected: PASS。

### Task 2: 扩展 Reporter 的规划事件

**Files:**
- Modify: `internal/engine/reporter.go`
- Modify: `internal/engine/terminal_reporter.go`
- Modify: `internal/engine/terminal_reporter_test.go`

**Interfaces:**
- Produces: `Reporter.OnPlan(context.Context, string)`
- Produces: `TerminalReporter.OnPlan(context.Context, string)`，输出 `🧠 [Plan] <plan>`

- [ ] **Step 1: 先在 TerminalReporter 测试中调用 `OnPlan`**

```go
reporter.OnThinking(ctx)
reporter.OnPlan(ctx, "1. 读取文件\n2. 总结项目")
```

期望输出增加：

```text
🧠 [Plan] 1. 读取文件
2. 总结项目
```

- [ ] **Step 2: 运行测试确认缺少方法**

Run: `env GOCACHE=/tmp/go-tiny-claw-gocache go test ./internal/engine -run TestTerminalReporterWritesEngineEvents -count=1`

Expected: FAIL，错误包含 `OnPlan undefined`。

- [ ] **Step 3: 添加接口与终端实现**

```go
func (r *TerminalReporter) OnPlan(_ context.Context, plan string) {
    r.write("🧠 [Plan] %s\n", plan)
}
```

- [ ] **Step 4: 运行 TerminalReporter 测试**

Run: `env GOCACHE=/tmp/go-tiny-claw-gocache go test ./internal/engine -run TerminalReporter -count=1`

Expected: PASS。

### Task 3: 实现一次有界规划与有限 Action 循环

**Files:**
- Create: `internal/engine/planning.go`
- Create: `internal/engine/loop_test.go`
- Modify: `internal/engine/loop.go`

**Interfaces:**
- Consumes: `provider.GenerateOptions`
- Consumes: `Reporter.OnPlan(context.Context, string)`
- Produces: `normalizePlan(string) string`
- Produces: 常量 `planningMaxTokens = 512`、`actionMaxTokens = 4096`、`maxPlanItems = 5`、`maxActionTurns = 10`

- [ ] **Step 1: 写规划清洗失败测试**

```go
func TestNormalizePlanDeduplicatesAndLimitsItems(t *testing.T) {
    raw := "1. 读取 README。\n读取 README。\n2. 检查入口。\n3. 总结结构。\n4. 给出用法。\n5. 补充注意事项。\n6. 不应出现。"
    want := "1. 读取 README\n2. 检查入口\n3. 总结结构\n4. 给出用法\n5. 补充注意事项"
    if got := normalizePlan(raw); got != want {
        t.Fatalf("normalizePlan() = %q, want %q", got, want)
    }
}
```

- [ ] **Step 2: 运行测试确认函数不存在**

Run: `env GOCACHE=/tmp/go-tiny-claw-gocache go test ./internal/engine -run TestNormalizePlan -count=1`

Expected: FAIL，错误包含 `undefined: normalizePlan`。

- [ ] **Step 3: 实现按行清洗、去重、限制与统一编号**

`normalizePlan` 使用正则 `^\\s*(?:[-*•]|\\d+[.)、])\\s*` 去除列表前缀，使用 `strings.TrimRight(item, "。.!！？ \\t")` 去除句末标点，以清洗后的完整文本作为去重键，最多保留 5 项；空结果回退为 `1. 分析任务并按需调用可用工具`。

- [ ] **Step 4: 写 Engine 行为失败测试**

使用记录每次 `tools` 与 `GenerateOptions` 的 fake provider，覆盖：

```go
// 开启 Thinking：第一次无工具且 MaxTokens=512，后续 Action 有工具且 MaxTokens=4096。
// 重复规划只通过 OnPlan 报告一次。
// 带 ToolCalls 的中间 Content 不通过 OnMessage 报告。
// 第 10 轮后仍有 ToolCalls 时返回“达到最大执行轮数 10”。
```

- [ ] **Step 5: 运行 Engine 测试确认旧循环行为失败**

Run: `env GOCACHE=/tmp/go-tiny-claw-gocache go test ./internal/engine -run 'TestAgentEngine|TestNormalizePlan' -count=1`

Expected: FAIL，至少包含规划调用次数、token 上限或中间消息断言失败。

- [ ] **Step 6: 重写 Run 为 Planning → Action**

规划请求使用复制的历史并追加专用用户指令：

```text
请先制定简洁的执行规划，不要回答最终结果，也不要假装已经调用工具。每行一个具体步骤，输出 3 到 5 步，不要重复。
```

规划仅执行一次，清洗后调用 `OnPlan` 并作为 assistant 消息加入历史。Action 使用 `for turn := 1; turn <= maxActionTurns; turn++`；仅当 `len(actionResp.ToolCalls) == 0` 时调用 `OnMessage` 并返回。第 10 轮仍含工具调用时返回 `达到最大执行轮数 10`。

- [ ] **Step 7: 运行 Engine 测试**

Run: `env GOCACHE=/tmp/go-tiny-claw-gocache go test ./internal/engine -count=1`

Expected: PASS。

### Task 4: Slack 用一条可更新消息展示规划

**Files:**
- Modify: `internal/channel/slack/reporter.go`
- Modify: `internal/channel/slack/reporter_test.go`
- Modify: `internal/channel/slack/client.go`
- Modify: `internal/channel/slack/client_test.go`
- Modify: `internal/channel/slack/bot_test.go`
- Create: `internal/engine/multi_reporter.go`
- Create: `internal/engine/multi_reporter_test.go`
- Modify: `internal/channel/slack/bot.go`
- Modify: `internal/channel/slack/socket.go`
- Modify: `cmd/claw/main.go`

**Interfaces:**
- Produces: `messageSender.SendMessage(...) (string, error)`，返回 Slack 消息时间戳
- Produces: `messageSender.UpdateMessage(ctx, channelID, timestamp, text) error`
- Consumes: `Reporter.OnPlan(context.Context, string)`
- Produces: `engine.NewMultiReporter(...Reporter)`，并由 Slack Bot 同时转发到 Slack 与终端

- [ ] **Step 1: 修改 Reporter 测试表达两条消息语义**

测试调用 `OnThinking`、`OnPlan`、两次工具事件和 `OnMessage`，断言：仅发送占位消息与最终回答；占位消息时间戳被更新为 `🧠 执行规划：\n<plan>`；工具事件没有发送记录。

- [ ] **Step 2: 运行 Slack Reporter 测试确认失败**

Run: `env GOCACHE=/tmp/go-tiny-claw-gocache go test ./internal/channel/slack -run TestReporter -count=1`

Expected: FAIL，错误包含 `OnPlan undefined` 或消息数量不符。

- [ ] **Step 3: 修改 Sender 接口与 Reporter**

```go
type messageSender interface {
    SendMessage(ctx context.Context, channelID, threadTimestamp, text string) (string, error)
    UpdateMessage(ctx context.Context, channelID, timestamp, text string) error
}
```

`OnThinking` 保存返回时间戳；`OnPlan` 优先更新，失败则降级发送；`OnToolCall` 与 `OnToolResult` 保持空实现；`OnMessage` 发送最终文本。

- [ ] **Step 4: 为 Slack Client 写 post 返回时间戳与 update 请求测试**

HTTP fake 分别断言 `/chat.postMessage` 返回的 `ts` 被传回，`/chat.update` 收到正确的 `channel`、`ts` 和 `text`。

- [ ] **Step 5: 实现 Client 的发送返回值与更新能力**

```go
channel, timestamp, err := s.client.PostMessageContext(ctx, channelID, options...)
_ = channel
return timestamp, err

_, _, _, err := s.client.UpdateMessageContext(ctx, channelID, timestamp, goslack.MsgOptionText(text, false))
return err
```

- [ ] **Step 6: 运行 Slack 包测试**

Run: `env GOCACHE=/tmp/go-tiny-claw-gocache go test ./internal/channel/slack -count=1`

Expected: PASS。

- [ ] **Step 7: 测试并实现 Reporter 扇出与主程序接线**

先测试 `NewMultiReporter` 会转发所有 Reporter 事件，再测试 `Bot.handleAppMention` 会同时写入 Slack sender 与附加的 TerminalReporter。`NewBot` 接收可变参数 `additionalReporters ...engine.Reporter`，主程序传入 `engine.NewTerminalReporter(os.Stdout)`。

Run: `env GOCACHE=/tmp/go-tiny-claw-gocache go test ./internal/engine ./internal/channel/slack ./cmd/claw -count=1`

Expected: PASS。

### Task 5: 文档与全量验证

**Files:**
- Modify: `README.md`

**Interfaces:**
- Documents: Slack 正常输出为“规划消息 + 最终回答”，终端额外保留工具细节。

- [ ] **Step 1: 更新 README 的 Slack 运行说明**

明确说明机器人会先更新一条执行规划，再发送最终回答；工具调用详情只写终端。

- [ ] **Step 2: 格式化并运行全部测试**

Run: `gofmt -w internal/provider/interface.go internal/provider/claude.go internal/provider/openai.go internal/engine/reporter.go internal/engine/terminal_reporter.go internal/engine/terminal_reporter_test.go internal/engine/planning.go internal/engine/loop.go internal/engine/loop_test.go internal/channel/slack/reporter.go internal/channel/slack/reporter_test.go internal/channel/slack/client.go internal/channel/slack/client_test.go internal/channel/slack/bot_test.go`

Run: `env GOCACHE=/tmp/go-tiny-claw-gocache go test ./... -count=1`

Expected: PASS。

- [ ] **Step 3: 运行竞态与静态检查**

Run: `env GOCACHE=/tmp/go-tiny-claw-gocache go test -race ./... -count=1`

Run: `env GOCACHE=/tmp/go-tiny-claw-gocache go vet ./...`

Run: `git diff --check`

Expected: 全部退出码为 0。

- [ ] **Step 4: 人工 Slack 验收**

重启机器人并在已邀请机器人的频道中 `@Demo App 请读取 README 并介绍项目`。确认规划占位消息被更新、工具细节不出现在 Slack、最终回答单独出现；终端仍显示规划与工具细节。
