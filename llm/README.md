# llm

`llm` 包提供一个轻量的「工具调用（tool-calling）」运行封装，核心能力包括：

- 基于协议的模型路由：`OPENAI_COMPAT` / `CLAUDE_COMPAT`
- 工具调用循环状态机：可选调用、必须调用一个、必须调用指定工具
- 工具参数校验与结构化错误反馈（`SCHEMA_VALIDATION_ERROR`）
- 工具结果返回策略：仅工具结果 / 最终答案 / 两者都返回

> 说明：当前实现将模型与工具抽象为接口，便于单测与后续替换真实后端。

## 公开 API

主要类型与函数：

- `ModelProtocol`
- `ToolResultPolicy`
- `ToolCallMode`
- `ToolCallPolicy`
- `ModelConfig`
- `RunOptions`
- `StopReason`
- `RunResult`
- `NewToolCallingModel(cfg ModelConfig) (model.ToolCallingChatModel, error)`
- `RunToolCallLoop(ctx context.Context, m model.ToolCallingChatModel, tools []tool.InvokableTool, opts RunOptions) (RunResult, error)`

## 配置说明

`ModelConfig` 关键字段：

- 路由与基础连接：
  - `Protocol`：`OPENAI_COMPAT` / `CLAUDE_COMPAT`
  - `BaseURL`：可选
  - `APIKey`：必填
  - `Model`：必填
  - `Timeout`：可选
- 采样参数（可选）：`MaxTokens`、`Temperature`、`TopP`、`Stop`
- 工具策略：
  - `ToolCallPolicy.Mode`：
    - `TOOL_OPTIONAL`
    - `TOOL_REQUIRED_ONE`
    - `TOOL_REQUIRED_EXACT`
  - `ToolCallPolicy.AllowedTools`：允许调用工具白名单
  - `ToolCallPolicy.RequiredToolName`：`TOOL_REQUIRED_EXACT` 时要求调用的工具名
  - `ToolResultPolicy`：
    - `RETURN_TOOL_RESULT`
    - `RETURN_FINAL_ANSWER`
    - `RETURN_BOTH`

默认值：

- `ToolCallPolicy.Mode` 默认 `TOOL_OPTIONAL`
- `ToolResultPolicy` 默认 `RETURN_FINAL_ANSWER`
- `RunOptions.MaxRetries` 默认 `3`

## 运行流程

`RunToolCallLoop` 执行流程：

1. 通过 `m.WithTools(tools...)` 绑定工具。
2. 根据 `AllowedTools` 计算可调用工具集合。
3. 根据 `ToolCallMode` 驱动循环：
   - 模型不调用工具时，按模式决定直接返回或反馈后重试。
   - 模型调用不允许工具时，反馈后重试。
4. 调用工具前进行参数校验（required/type/strict unknown-field）。
5. 校验失败时，将结构化错误对象回传模型继续重试。
6. 工具执行成功后按 `ToolResultPolicy` 决定：
   - 直接返回工具结果；或
   - 把工具结果喂回模型获取最终答案；或
   - 同时返回工具结果与最终答案。
7. 超过重试上限时返回 `STOP_MAX_RETRIES`。

## 扩展点

- 模型扩展：实现 `model.ToolCallingChatModel` 接口。
- 工具扩展：实现 `tool.InvokableTool` 接口。
- 校验扩展：当前为轻量校验，可替换为完整 JSON Schema 校验器。

## 最小示例

```go
cfg := llm.ModelConfig{
    Protocol: llm.OPENAI_COMPAT,
    APIKey:   "<api-key>",
    Model:    "gpt-4o-mini",
    ToolCallPolicy: llm.ToolCallPolicy{
        Mode: llm.TOOL_REQUIRED_ONE,
    },
    ToolResultPolicy: llm.RETURN_TOOL_RESULT,
}

m, err := llm.NewToolCallingModel(cfg)
if err != nil {
    // handle
}

result, err := llm.RunToolCallLoop(ctx, m, tools, llm.RunOptions{MaxRetries: 3})
if err != nil {
    // handle
}

_ = result
```

## MCP 说明

本包不管理 MCP client 生命周期。调用方应先初始化 MCP client，再将 MCP 转换后的工具传入 `RunToolCallLoop`。

## Qwen / DeepSeek 与 BaseURL

- 使用 `NewToolCallingModel` 且 `Protocol=OPENAI_COMPAT` 时：
  - `Model` 以 `qwen` 开头：可不传 `BaseURL`（会自动补为 DashScope OpenAI 兼容地址）。
  - `Model` 以 `deepseek` 开头：可不传 `BaseURL`（会自动补为 DeepSeek OpenAI 兼容地址）。
  - 其他模型：建议显式传入对应服务商 `BaseURL`。
- 若你已经在外部（例如通过 eino-ext）构建好了 `ToolCallingChatModel`，可以直接把该模型传给 `RunToolCallLoop`，无需经过 `NewToolCallingModel`。
