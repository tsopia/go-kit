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

使用 `struct` 定义参数，自动生成 Schema：

```go
type WeatherArgs struct {
    City string `json:"city" desc:"查询城市" required:"true"`
    Unit string `json:"unit" desc:"温度单位" enum:"celsius,fahrenheit"`
}

// 创建工具
weatherTool := llm.NewStructTool("get_weather", "查询天气", func(ctx context.Context, args *WeatherArgs) (string, error) {
    return fmt.Sprintf("%s 的天气是 晴, 25%s", args.City, args.Unit), nil
})
```

### 3. 创建 Agent 并运行

```go
agent, _ := llm.NewAgent(ctx, llm.AgentConfig{
    ModelConfig:    cfg,
    InvokableTools: []llm.InvokableTool{weatherTool}, // 自动适配
    SystemPrompt:   "你是一个有用的助手。",
})

// 运行 (Generate)
msg, _ := agent.Generate(ctx, []*schema.Message{
    schema.UserMessage("海淀区天气如何？"),
})
fmt.Println(msg.Content)
```

## 🛠 高级功能

### 1. 可观测性 (Observability)

支持 Langfuse 链路追踪和标准日志记录。

```go
// 初始化 Langfuse
lfHandler, flush, _ := llm.NewLangfuseHandler(&llm.LangfuseConfig{...})
defer flush()

// 初始化日志 (slog)
logHandler := llm.NewLogHandler(slog.Default())

// 注入 Agent
agent, _ := llm.NewAgent(ctx, llm.AgentConfig{
    // ...
    Callbacks: []callbacks.Handler{lfHandler, logHandler},
})
```

### 2. 强制工具调用与重试 (ToolChoice & Retry)

```go
forced := schema.ToolChoiceForced
agent, _ := llm.NewAgent(ctx, llm.AgentConfig{
    // ...
    ToolChoice: &forced, // 强制模型必须调用工具
    MaxRetries: 3,       // 如果工具报错，自动重试 3 次
})
```

### 3. 工具结果直接返回 (Direct Return)

某些场景下（如搜索），工具执行后不需要模型再通过 LLM 总结，直接返回工具结果可节省 Token：

```go
agent, _ := llm.NewAgent(ctx, llm.AgentConfig{
    // ...
    ToolReturnDirectly: map[string]struct{}{
        "search_tool": {}, // 执行完 search_tool 后立即结束对话并返回结果
    },
})
```

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
