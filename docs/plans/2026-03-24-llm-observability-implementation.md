# LLM Structured Observability Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 `llm` 包新增面向线上排障的结构化 Agent / Tool 日志，保留现有 `NewLogHandler` 的组件级 callback 日志语义不变。

**Architecture:** 保持当前 `Observability.Callbacks` 和 `NewLogHandler` 作为通用组件日志入口；新增一条 `llm` 自己的结构化日志链路，在 `Agent`、模型包装器和 `compose.ToolMiddleware` 三层输出业务语义日志。日志增强优先服务“为什么没调工具 / 为什么重试 / 为什么 direct return”，不引入 `RuntimeSpec` 缓存，也不先暴露不稳定的 `ParentMessageID` 一类公共字段。

**Tech Stack:** Go 1.24、`log/slog`、CloudWeGo Eino `react.Agent`、`compose.ToolMiddleware`、table-driven tests、focused `go test ./llm`

---

## 前置检查

- [x] 已完整阅读相关代码文件（`llm/agent.go`、`llm/config.go`、`llm/callbacks.go`、`llm/model_force.go`、`llm/README.md`）
- [x] 已完整阅读相关测试文件（`llm/callbacks_test.go`、`llm/agent_test.go`、`llm/runtime_builder_test.go`）
- [x] 已列出代码现状（`compileRuntimeSpec()` 只在 `NewAgent()` 调用，不在热路径；`NewLogHandler` 只打组件级 start/end/error；`Extraction` 的 retry 目前在工具中间件里把非终态错误转成结果字符串）

## 非目标

- 不做 `RuntimeSpec` 缓存。`compileRuntimeSpec()` 当前只在 `NewAgent()` 构造路径执行一次，不值得引入 `sync.Once` 和缓存状态。
- 不修改 `NewLogHandler` 的既有输出语义，避免影响已经依赖组件级 callback 日志的使用方式。
- 不先引入公开 `ToolSpan` / `ParentMessageID` API。上游 `schema.Message` 没有稳定的顶层 message id，现阶段直接公开会制造伪稳定字段。

### Task 1: 锁定现有 `NewLogHandler` 语义并定义新的结构化日志契约

**目标：** 先用测试区分“旧组件日志”和“新增结构化日志”，避免后续实现混淆两条链路。

**Files:**
- Modify: `llm/callbacks_test.go`
- Create: `llm/structured_log_test.go`

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**Step 1: Write the failing tests**

在 `llm/structured_log_test.go` 新增 table-driven tests，锁定 3 类场景：

- `Conversation` 只输出 `agent.start` / `model.decision` / `agent.end`
- `Assistant` 调工具时输出 `tool.start` / `tool.success`
- `Extraction` 重试 + direct return 时输出 `tool.error(retryable=true)`、`tool.success`、`agent.end(direct_return=true)`

示例测试骨架：

```go
func TestStructuredLogs_ExtractionRetryAndDirectReturn(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	agent, err := NewAgent(context.Background(), AgentConfig{
		Model: AgentModelConfig{Instance: model},
		Tools: ToolsConfig{Invokable: []InvokableTool{tool}},
		Execution: ExecutionConfig{
			Mode:              Extraction,
			MaxRetries:        2,
			DirectReturnTools: map[string]struct{}{"extract": {}},
		},
		Observability: ObservabilityConfig{
			StructuredLogs: &StructuredLogConfig{
				Logger:           logger,
				LogToolArguments: true,
				LogToolResults:   true,
			},
		},
	})
	if err != nil {
		t.Fatalf("NewAgent failed: %v", err)
	}

	_, err = agent.Generate(context.Background(), []*schema.Message{schema.UserMessage("提取数据")})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	for _, want := range []string{
		"\"event\":\"agent.start\"",
		"\"event\":\"model.decision\"",
		"\"event\":\"tool.error\"",
		"\"retryable\":true",
		"\"event\":\"tool.success\"",
		"\"direct_return\":true",
		"\"event\":\"agent.end\"",
	} {
		if !strings.Contains(buf.String(), want) {
			t.Fatalf("missing log fragment %q\nlogs=%s", want, buf.String())
		}
	}
}
```

同时在 `llm/callbacks_test.go` 增加断言，确保 `NewLogHandler` 继续只输出 `Component Start` / `Component End` / `Component Error`，不混入新的 `event=agent.start` 字段。

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestLogHandler_Integration|TestStructuredLogs_' -v`

Expected: FAIL，报 `StructuredLogConfig`、结构化事件或字段尚不存在。

**Step 3: Write minimal implementation**

暂不改行为，只补最小结构和测试夹具需要的类型声明：

- `ObservabilityConfig.StructuredLogs`
- `StructuredLogConfig`
- 结构化日志的基础 event 常量

**Step 4: Run test to verify it still fails for behavior**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestStructuredLogs_' -v`

Expected: FAIL，从“缺类型”前进到“缺日志行为”。

**Step 5: Commit**

```bash
git add llm/callbacks_test.go llm/structured_log_test.go llm/config.go
git commit -m "test(llm): define structured observability contract"
```

### Task 2: 引入结构化日志配置和内部记录器

**目标：** 在不污染 `NewLogHandler` 的前提下，为 `llm` 提供一条专用结构化日志输出通道。

**Files:**
- Modify: `llm/config.go`
- Create: `llm/structured_log.go`
- Test: `llm/config_test.go`

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**Step 1: Write the failing test**

在 `llm/config_test.go` 增加断言：

```go
func TestAgentConfigDefaults(t *testing.T) {
	cfg := AgentConfig{}
	if cfg.Observability.StructuredLogs != nil {
		t.Fatalf("expected nil structured logs, got %#v", cfg.Observability.StructuredLogs)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestAgentConfigDefaults' -v`

Expected: FAIL，因为 `StructuredLogs` 尚不存在。

**Step 3: Write minimal implementation**

在 `llm/structured_log.go` 实现：

- `type StructuredLogConfig struct { Logger *slog.Logger; LogToolArguments bool; LogToolResults bool; MaxFieldLength int }`
- `type structuredLogger struct { ... }`
- `func newStructuredLogger(cfg *StructuredLogConfig) *structuredLogger`
- `func (l *structuredLogger) enabled() bool`
- `func (l *structuredLogger) log(event string, attrs ...any)`
- 统一截断 helper，避免参数/结果日志过长

在 `llm/config.go` 中给 `ObservabilityConfig` 增加：

```go
type ObservabilityConfig struct {
	Callbacks      []callbacks.Handler
	StructuredLogs *StructuredLogConfig
}
```

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestAgentConfigDefaults' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add llm/config.go llm/structured_log.go llm/config_test.go
git commit -m "feat(llm): add structured observability config"
```

### Task 3: 为模型调用增加决策级日志

**目标：** 解决“为什么工具没触发”的排障问题，记录每轮模型调用是否产出 `tool_calls`。

**Files:**
- Modify: `llm/agent.go`
- Create: `llm/model_observer.go`
- Test: `llm/structured_log_test.go`

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**Step 1: Write the failing test**

新增测试覆盖：

- `Assistant` 模式模型直接返回文本时，日志包含 `event=model.decision` 和 `tool_call_count=0`
- `Assistant` 模式模型返回工具调用时，日志包含 `tool_call_count=1` 和工具名

示例测试骨架：

```go
func TestStructuredLogs_ModelDecisionWithoutToolCall(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	agent, err := NewAgent(context.Background(), AgentConfig{
		Model: AgentModelConfig{Instance: &fakeToolCallingModel{
			responses: []*schema.Message{{Role: schema.Assistant, Content: "plain answer"}},
		}},
		Execution: ExecutionConfig{Mode: Assistant},
		Observability: ObservabilityConfig{
			StructuredLogs: &StructuredLogConfig{Logger: logger},
		},
	})
	if err != nil {
		t.Fatalf("NewAgent failed: %v", err)
	}

	_, err = agent.Generate(context.Background(), []*schema.Message{schema.UserMessage("hello")})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(buf.String(), "\"tool_call_count\":0") {
		t.Fatalf("expected no-tool decision log, got %s", buf.String())
	}
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestStructuredLogs_ModelDecision' -v`

Expected: FAIL，日志里没有 `model.decision`。

**Step 3: Write minimal implementation**

在 `llm/model_observer.go` 新增模型包装器：

- `type observedToolCallingModel struct { inner model.ToolCallingChatModel; logs *structuredLogger; mode ExecutionMode }`
- 包装 `Generate` / `Stream`
- 在输出里记录：
  - `event=model.decision`
  - `execution_mode`
  - `tool_choice`
  - `tool_call_count`
  - `tool_names`
  - `finish_reason`
  - `reasoning_tokens`（如果可取）

在 `NewAgent()` 中，当 `StructuredLogs` 启用时，先包一层 `observedToolCallingModel`，再交给 `react.Agent`。

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestStructuredLogs_ModelDecision' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add llm/agent.go llm/model_observer.go llm/structured_log_test.go
git commit -m "feat(llm): log model tool-call decisions"
```

### Task 4: 为工具调用增加 start/success/error/retry 日志

**目标：** 在 `Assistant` 和 `Extraction` 下都能看清工具名称、参数摘要、耗时、重试和 direct return 候选状态。

**Files:**
- Modify: `llm/agent.go`
- Modify: `llm/model_force.go`
- Create: `llm/tool_observer.go`
- Test: `llm/structured_log_test.go`

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**Step 1: Write the failing test**

新增 table-driven tests 覆盖：

- `Assistant` 调工具成功，输出 `tool.start` / `tool.success`
- `Extraction` 首次失败但可重试，输出 `tool.error retryable=true`
- `Extraction` 重试耗尽，输出 `tool.error retryable=false terminal=true`
- `DirectReturnTools` 命中时，`tool.success` 或 `agent.end` 带 `direct_return=true`

示例测试骨架：

```go
func TestStructuredLogs_ToolRetryLifecycle(t *testing.T) {
	// 断言日志片段：
	// "event":"tool.start"
	// "tool_name":"extract"
	// "attempt":1
	// "event":"tool.error"
	// "retryable":true
	// "event":"tool.success"
	// "attempt":2
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestStructuredLogs_Tool' -v`

Expected: FAIL

**Step 3: Write minimal implementation**

在 `llm/tool_observer.go` 实现通用工具中间件：

- `event=tool.start`
- `tool_name`
- `tool_call_id`
- `attempt`
- `arguments`（可选、截断）
- `event=tool.success`
- `latency_ms`
- `result`（可选、截断）
- `event=tool.error`
- `error`
- `retryable`
- `terminal`

在 `llm/model_force.go` 中让 `newExtractionRetryMiddleware` 在失败转修复字符串前后暴露 retry 语义：

- 失败但未超限：`retryable=true`
- 失败且超限：`retryable=false, terminal=true`

在 `NewAgent()` 中将工具日志 middleware 放到 `toolsConfig.ToolCallMiddlewares`，保证 Assistant/Extraction 都能覆盖。

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestStructuredLogs_Tool' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add llm/agent.go llm/model_force.go llm/tool_observer.go llm/structured_log_test.go
git commit -m "feat(llm): log tool lifecycle and retry semantics"
```

### Task 5: 为 Agent 入口增加 start/end/outcome 日志

**目标：** 把整个单 Agent 调用闭环串起来，最终能从日志里看出模式、是否 direct return、总耗时和最终状态。

**Files:**
- Modify: `llm/agent.go`
- Test: `llm/structured_log_test.go`

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**Step 1: Write the failing test**

新增测试覆盖：

- `agent.start` 包含 `execution_mode`、`tool_count`、`direct_return_enabled`
- `agent.end` 包含 `status=success|error`、`latency_ms`、`direct_return`

示例测试骨架：

```go
func TestStructuredLogs_AgentOutcome(t *testing.T) {
	// 断言：
	// "event":"agent.start"
	// "execution_mode":"assistant"
	// "tool_count":1
	// "event":"agent.end"
	// "status":"success"
	// "latency_ms":
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestStructuredLogs_AgentOutcome' -v`

Expected: FAIL

**Step 3: Write minimal implementation**

在 `Agent.Generate()` 和 `Agent.Stream()` 中补 Agent 级日志：

- `agent.start`
- `agent.end`
- `agent.error`

记录字段：

- `execution_mode`
- `tool_count`
- `direct_return_enabled`
- `direct_return`
- `latency_ms`
- `message_count`
- `error`

确保 direct return 的成功路径和 `context.Canceled` 兜底路径都能输出统一 `agent.end`。

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestStructuredLogs_AgentOutcome' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add llm/agent.go llm/structured_log_test.go
git commit -m "feat(llm): log agent lifecycle outcomes"
```

### Task 6: 文档化新旧日志差异和排障字段

**目标：** 让使用者明确知道 `NewLogHandler` 和结构化日志各自负责什么，以及看到日志后该怎么排障。

**Files:**
- Modify: `llm/README.md`

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**Step 1: Write the failing test**

无需自动化测试；先列出 README 必须补充的内容清单：

- `NewLogHandler` 只负责组件级 callback 日志
- `StructuredLogs` 负责 Agent/Model/Tool 语义日志
- 一份 `Assistant` 示例日志
- 一份 `Extraction` 重试 + direct return 示例日志
- “为什么没调工具”排障检查路径

**Step 2: Write minimal implementation**

在 README 中补：

- 新配置示例
- 日志字段表
- 真实风格的 JSON 示例
- 排障建议

**Step 3: Manual verification**

检查 README 是否明确说明：

- 不做 `RuntimeSpec` 缓存
- 不修改 `NewLogHandler` 既有语义
- 不承诺公开 `ParentMessageID`

**Step 4: Commit**

```bash
git add llm/README.md
git commit -m "docs(llm): document structured observability logs"
```

### Task 7: 全量验证并整理 PR 说明

**目标：** 在合并前确认结构化日志不影响现有行为，且新增日志覆盖了关键排障场景。

**Files:**
- Modify: `llm/structured_log_test.go`
- Modify: `llm/callbacks_test.go`

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**Step 1: Run focused package tests**

Run: `GOCACHE=/tmp/go-build go test ./llm -v`

Expected: PASS

**Step 2: Run repository-wide tests**

Run: `GOCACHE=/tmp/go-build go test ./...`

Expected: PASS

**Step 3: Prepare PR summary**

在 PR 描述中明确：

- `NewLogHandler` 未改变语义
- 新增 `StructuredLogs` 是面向排障的增强链路
- 修复了哪些“看不见”的问题
- 尚未做哪些字段承诺（如 `ParentMessageID`）

**Step 4: Commit**

```bash
git add llm/structured_log_test.go llm/callbacks_test.go llm/README.md
git commit -m "test(llm): verify structured observability end-to-end"
```

