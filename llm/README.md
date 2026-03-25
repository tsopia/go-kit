# llm — Eino LLM 高级封装库

基于 [Eino](https://github.com/cloudwego/eino) 框架的生产级 LLM 工具调用与 Agent 封装库。本 README 旨在为 AI 编程助手提供完整的上下文索引。

## 🌟 核心特性

- **多协议统一路由**：一键切换 OpenAI / Claude / DeepSeek / Gemini / Ollama / Moonshot(Kimi) 等模型协议。
- **React Agent 增强**：内置工具调用循环、自动重试、死循环检测。
- **StructTool**：基于 Go 结构体标签（tag）自动生成 JSON Schema，支持嵌套结构、枚举值。
- **可观测性**：开箱即用的 Langfuse 和 Slog 集成。
- **运行时动态控制**：支持请求级别的模型替换、工具替换、参数调整。
- **流式完整支持**：从推理到工具调用再到最终回答的全链路流式。

## 🚀 快速开始

### 1. 初始化模型配置

```go
cfg := llm.ModelConfig{
    Protocol: llm.OPENAI, // 或 llm.CLAUDE, llm.OLLAMA ...
    BaseURL:  "https://api.openai.com/v1", // 可选
    APIKey:   "sk-xxx",
    Model:    "gpt-4o",
}
```

### 2. 定义工具 (推荐 StructTool)

使用 `struct` 定义目标结构，自动生成 Schema：

```go
type WeatherResult struct {
    City        string `json:"city" desc:"城市" required:"true"`
    Condition   string `json:"condition" desc:"天气情况" required:"true"`
    Temperature string `json:"temperature" desc:"温度" required:"true"`
}

weatherTool := llm.NewStructTool[WeatherResult]("extract_weather", "提取天气结果")
```

### 3. 创建 Agent 并运行

```go
agent, _ := llm.NewAgent(ctx, llm.AgentConfig{
    Model: llm.AgentModelConfig{Config: cfg},
    Tools: llm.ToolsConfig{Invokable: []llm.InvokableTool{weatherTool}},
    Prompt: llm.PromptConfig{
        System: "从用户请求中提取天气结果，输出必须是合法 JSON。",
    },
    Execution: llm.ExecutionConfig{
        Mode:              llm.Extraction,
        DirectReturnTools: map[string]struct{}{"extract_weather": {}},
    },
})

msg, _ := agent.Generate(ctx, []*schema.Message{
    schema.UserMessage("北京天气晴，25摄氏度。请提取结构化结果。"),
})
fmt.Println(msg.Content)
```

## 🛠 高级功能

### 0. 执行模式与配置约束

推荐优先使用 `Execution.Mode` 描述 Agent 行为：

- `Conversation`：纯对话，不启用工具
- `Assistant`：工具可用，由模型自行决定是否调用
- `Extraction`：先完成工具任务，再决定是否总结

配置约束：

- `Mode` 和 `ToolChoice` 同时传入时，以 `Mode` 为准；`ToolChoice` 仅保留兼容路径
- `Conversation` 不允许同时配置工具、`MaxRetries` 或 `DirectReturnTools`
- `Assistant` 不允许配置 `MaxRetries`
- `DirectReturnTools` 中的工具名必须真实存在，否则 `NewAgent` 会直接返回错误

### 1. 可观测性 (Observability)

`llm` 现在有两条互补的观测链路：

- `Callbacks` / `NewLogHandler`：保留现有组件级日志语义，适合看底层 `ChatModel` / `Tool` 组件有没有被调用
- `StructuredLogs`：新增的 Agent 语义日志，适合排查“为什么没调工具 / 为什么重试 / 为什么 direct return”

两者可以同时开启：

```go
// 初始化 Langfuse
lfHandler, flush, _ := llm.NewLangfuseHandler(&llm.LangfuseConfig{...})
defer flush()

// 初始化日志 (slog)
logHandler := llm.NewLogHandler(slog.Default())

// 注入 Agent
agent, _ := llm.NewAgent(ctx, llm.AgentConfig{
    // ...
    Observability: llm.ObservabilityConfig{
        Callbacks: []callbacks.Handler{lfHandler, logHandler},
        StructuredLogs: &llm.StructuredLogConfig{
            Logger:           slog.Default(),
            LogToolArguments: true,
            LogToolResults:   true,
            MaxFieldLength:   256,
        },
    },
})
```

说明：

- `NewLogHandler` 的既有输出语义没有改，仍然只输出 `Component Start` / `Component End` / `Component Error`
- `StructuredLogs` 会输出 `agent.start` / `model.decision` / `tool.start` / `tool.success` / `tool.error` / `agent.end`
- 当前没有为 `RuntimeSpec` 做缓存；构造成本不在热路径，不值得为此引入额外状态
- 当前不承诺公开 `ParentMessageID` 一类字段；上游 `schema.Message` 没有稳定的顶层 message id

常用字段：

- `agent.start`：`execution_mode`、`tool_count`、`direct_return_enabled`、`message_count`
- `model.decision`：`configured_tool_choice`、`tool_call_count`、`tool_names`、`finish_reason`、`reasoning_tokens`
- `tool.start`：`tool_name`、`tool_call_id`、`attempt`、`arguments`
- `tool.success`：`tool_name`、`attempt`、`latency_ms`、`result`、`direct_return`
- `tool.error`：`tool_name`、`attempt`、`latency_ms`、`retryable`、`terminal`、`error`
- `agent.end`：`status`、`latency_ms`、`direct_return`

`Assistant` 场景日志示例：

```json
{"level":"INFO","msg":"agent.start","event":"agent.start","execution_mode":"assistant","tool_count":1,"direct_return_enabled":false,"message_count":1}
{"level":"INFO","msg":"model.decision","event":"model.decision","execution_mode":"assistant","configured_tool_choice":"allowed","tool_call_count":1,"tool_names":["lookup_user"],"finish_reason":"tool_calls"}
{"level":"INFO","msg":"tool.start","event":"tool.start","tool_name":"lookup_user","tool_call_id":"tc1","attempt":1,"arguments":"{\"name\":\"Alice\"}"}
{"level":"INFO","msg":"tool.success","event":"tool.success","tool_name":"lookup_user","tool_call_id":"tc1","attempt":1,"latency_ms":12,"result":"{\"name\":\"Alice\"}"}
{"level":"INFO","msg":"agent.end","event":"agent.end","execution_mode":"assistant","tool_count":1,"direct_return_enabled":false,"latency_ms":38,"status":"success"}
```

`Extraction` 重试 + direct return 日志示例：

```json
{"level":"INFO","msg":"agent.start","event":"agent.start","execution_mode":"extraction","tool_count":1,"direct_return_enabled":true,"message_count":1}
{"level":"INFO","msg":"model.decision","event":"model.decision","execution_mode":"extraction","configured_tool_choice":"forced","tool_call_count":1,"tool_names":["extract_resume"]}
{"level":"INFO","msg":"tool.start","event":"tool.start","tool_name":"extract_resume","tool_call_id":"tc1","attempt":1,"arguments":"{\"query\":\"bad\"}"}
{"level":"ERROR","msg":"tool.error","event":"tool.error","tool_name":"extract_resume","tool_call_id":"tc1","attempt":1,"latency_ms":4,"retryable":true,"terminal":false,"error":"missing required field"}
{"level":"INFO","msg":"model.decision","event":"model.decision","execution_mode":"extraction","configured_tool_choice":"forced","tool_call_count":1,"tool_names":["extract_resume"]}
{"level":"INFO","msg":"tool.start","event":"tool.start","tool_name":"extract_resume","tool_call_id":"tc2","attempt":2,"arguments":"{\"query\":\"good\"}"}
{"level":"INFO","msg":"tool.success","event":"tool.success","tool_name":"extract_resume","tool_call_id":"tc2","attempt":2,"latency_ms":6,"result":"{\"name\":\"Alice\"}","direct_return":true}
{"level":"INFO","msg":"agent.end","event":"agent.end","execution_mode":"extraction","tool_count":1,"direct_return_enabled":true,"latency_ms":29,"status":"success","direct_return":true}
```

排障顺序建议：

1. 先看 `agent.start`，确认 `execution_mode`、`tool_count`、`direct_return_enabled` 是否符合预期
2. 再看 `model.decision`，判断模型是否真的产出了 `tool_calls`
3. 如果有 `tool.start` 但没有 `tool.success`，继续看 `tool.error` 的 `retryable` / `terminal`
4. 如果工具成功但结果不像最终回答，检查 `direct_return` 是否命中，或者是否仍回到了模型总结

### 2. Extraction 模式与失败修复

```go
agent, _ := llm.NewAgent(ctx, llm.AgentConfig{
    // ...
    Execution: llm.ExecutionConfig{
        Mode:       llm.Extraction, // 强制模型先完成工具任务
        MaxRetries: 3,              // 工具报错后反馈给模型修正再试
    },
})
```

`MaxRetries` 只在 `Extraction` 模式下有效；如果放到 `Conversation` 或 `Assistant`，`NewAgent` 会直接报错。

### 3. 工具结果直接返回 (Direct Return)

某些场景下（如搜索），工具执行后不需要模型再通过 LLM 总结，直接返回工具结果可节省 Token：

```go
agent, _ := llm.NewAgent(ctx, llm.AgentConfig{
    // ...
    Execution: llm.ExecutionConfig{
        Mode: llm.Assistant,
        DirectReturnTools: map[string]struct{}{
            "search_tool": {}, // 执行完 search_tool 后立即结束对话并返回结果
        },
    },
})
```

`DirectReturnTools` 只能填写已经注册到 `Tools.Standard`、`Tools.Invokable` 或 MCP 中的工具名。

### 4. 运行时动态控制 (Runtime Options)

在 `Generate` 或 `Stream` 时动态修改行为，不影响 Agent 实例。

```go
import "github.com/cloudwego/eino/flow/agent"

// 场景 A: 临时更换模型 (例如降级到 3.5)
agent.Generate(ctx, input, agent.WithChatModel(gpt35Model))

// 场景 B: 调整温度
agent.Generate(ctx, input, agent.WithChatModelOptions(model.WithTemperature(0.1)))

// 场景 C: 获取中间状态 (Result Event Stream)
logOpt, future := agent.WithMessageFuture()
go func() {
    defer future.Close()
    stream, _ := future.GetMessageStreams()
    for { msg, _ := stream.Recv(); fmt.Println("中间状态:", msg) }
}()
agent.Generate(ctx, input, logOpt)
```

## 📦 架构说明

`llm` 包是对 CloudWeGo Eino 框架的 Opinionated 封装：

- **Model**: 实现了 `eino/components/model` 接口。
- **Agent**: 封装了 `eino/flow/agent/react`。
- **Tool**: 提供了 `StructTool` 到 `eino/components/tool` 的适配器。

如需更复杂的编排（如多 Agent 协作），可使用 `agent.ExportGraph()` 导出底层 Graph 节点，嵌入到 Eino 的 Graph 中。
