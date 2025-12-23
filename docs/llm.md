# LLM 客户端

统一封装大模型客户端，屏蔽供应商差异，基于 [Eino](https://www.cloudwego.io/zh/docs/eino/overview/eino_open_source/) 的“连接器 + 配置”理念，实现一套通用接口即可调用不同模型。

## 功能特性

- **通用接口**：统一的 `Chat` / `Stream` 方法，按消息数组驱动。
- **工具调用**：支持 OpenAI 风格的 `tool_calls`，可附带工具列表与调用策略。
- **供应商解耦**：通过 `provider` 注册表加载具体实现，默认内置 OpenAI；可注册自定义/多供应商。
- **配置直连**：`Config` 可直接从 Viper/文件映射；`RequestOptions` 支持运行时覆盖。
- **Eino 兼容**：提供 `RegisterEinoProvider` 与 `RegisterEinoAgentProvider`，可复用 Eino executor / agent / 编排。

## 快速开始

```go
client, err := llm.NewClient(ctx, llm.Config{
    Provider: llm.ProviderOpenAI,
    Model:    "gpt-4o-mini",
    APIKey:   os.Getenv("OPENAI_API_KEY"),
    Options: llm.RequestOptions{
        Temperature: llm.FloatPtr(0.3), // 或使用 WithTemperature 运行时覆盖
    },
})
if err != nil {
    log.Fatal(err)
}

	resp, err := client.Chat(ctx, []llm.Message{
	    {Role: llm.RoleUser, Content: "用一句话介绍 go-kit"},
	}, llm.WithMaxTokens(64),
	   llm.WithTools([]llm.ToolDefinition{{Name: "search"}}),
	   llm.WithToolChoice("auto"),
	)
```

## 配置项

| 字段 | 说明 |
| --- | --- |
| `provider` | 模型供应商，默认注册 `openai` |
| `model` | 模型/部署名 |
| `base_url` | 自定义接口域名，默认 `https://api.openai.com` |
| `api_key` | 认证 token |
| `version` | 供应商版本号（如 Azure OpenAI 的 `api-version`） |
| `timeout` | 默认超时 |
| `options` | `RequestOptions` 默认值：`temperature` / `top_p` / `max_tokens` / `stop` / `stream` / `headers` / `tools` / `tool_choice` 等 |

运行时可用 `WithTemperature`、`WithMaxTokens`、`WithHeaders`、`WithTools`、`WithToolChoice` 等 Option 覆盖。

## 自定义与 Eino 集成

- **注册自定义 provider**：`llm.RegisterProvider("vendor", func(ctx context.Context, cfg llm.Config) (llm.Adapter, error) { ... })`
- **Eino executor 复用**：若已有 Eino 的执行器，实现 `llm.EinoExecutor` 后调用 `llm.RegisterEinoProvider("vendor", builder)`。
- **Eino Agent/编排复用**：已有 Eino Agent（含工具链路）时，实现 `llm.EinoAgent`，使用 `RegisterEinoAgentProvider("vendor", builder)` 即可接入。

## 错误处理

- 未注册 provider：`llm.ErrProviderNotRegistered`
- 当前实现未支持流式：`llm.ErrStreamNotSupported`

## 示例

见 `examples/llm-basic`：通过环境变量 `OPENAI_API_KEY` 创建客户端并完成一次对话。
