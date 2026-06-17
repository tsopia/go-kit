# llm 包优化计划（OPTIMIZATION）

> 本文档是 [`ROADMAP.md`](./ROADMAP.md) 阶段 0 的详细执行计划。  
> 每个优化项独立编号，**可并行、可独立提交**。  
> **维护规则**：完成一项后，把状态从 📋 改为 ✅，并在 `ROADMAP.md` 对应能力行同步更新。

---

## 优化项总览

| ID | 标题 | 优先级 | 状态 | 预估工作量 | 依赖 |
|----|------|--------|------|-----------|------|
| [O-001](#o-001导出-sentinel-errors) | 导出 sentinel errors | P0 | ✅ | 0.5 天 | 无 |
| [O-002](#o-002多模态输入-api) | 多模态输入 API | P0 | ✅ | 1 天 | 无 |
| [O-003](#o-003工具层防御机制) | 工具层防御机制 | P0 | ✅ | 1.5 天 | 无 |
| [O-004](#o-004adkagent-运行时-option) | ADKAgent 运行时 Option | P1 | ✅ | 1 天 | 无 |
| [O-005](#o-005暴露-adk-middleware-注册入口) | 暴露 ADK Middleware 注册入口 | P1 | ✅ | 0.5 天 | 无 |
| [O-006](#o-006文档标记-newagent-为-legacy) | 文档标记 NewAgent 为 Legacy | P1 | ✅ | 0.5 天 | O-007 |
| [O-007](#o-007newadkagent-能力对齐核查) | NewADKAgent 能力对齐核查 | **P0（门禁）** | ✅ | 1 天 | 无 |
| [O-008](#o-008流式-modeldecision-补记) | 流式 model.decision 补记 | P1 | ✅ | 1 天 | 无 |
| [O-009](#o-009tokenusage-用量聚合) | Token/Usage 用量聚合 | P1 | 🟡 | 1.5 天 | 无 |

**并行性**：O-001 / O-002 / O-003 / O-005 / O-007 / O-008 / O-009 完全独立，可并行执行。  
**串行性**：O-006 依赖 O-007（核查完成后才能确定文档措辞）。  
**门禁**：O-007 是 DR-001（把 react 标 Legacy）成立的前提——**能力对齐核查必须先于"标 Legacy"的决策与文档（O-006）**，否则属于先下结论再找证据。因此 O-007 提为 P0。

---

## 通用原则

1. **TDD**：先写失败测试，再写实现，最后跑通
2. **单次提交 ≤ 50 行实现代码**（不含测试）；测试:实现 ≥ 1.5:1
3. **不破坏现有 API**：所有改动为纯增量或向后兼容扩展
4. **错误处理**：所有 error 显式返回，禁止 `_ =`
5. **context 传播**：新增公开 API 第一参必为 `ctx context.Context`
6. **验证命令**：每个 Task 完成后跑 `go build ./... && go test ./llm -count=1`

---

## O-001：导出 sentinel errors

### 背景

当前 `llm` 包内**零导出错误常量**。所有错误通过 `errors.New` / `fmt.Errorf` 即时返回：

| 位置 | 错误场景 | 当前实现 |
|------|---------|---------|
| `llm.go:72-80` | `NewModel` 校验失败（空 model / 缺 APIKey） | `errors.New("...")` |
| `llm.go:88-109` | 未知 `ModelProtocol` | `fmt.Errorf("unsupported protocol: %s", ...)` |
| `runtime_spec.go:99-117` | `compileRuntimeSpec` 配置冲突 | `fmt.Errorf("...")` |
| `model_force.go:97` | Extraction 重试耗尽 | `fmt.Errorf("...")` |
| `adk_extraction.go:67` | ADK Extraction 重试耗尽 | `fmt.Errorf("...")` |
| `mcp.go:90-91` | 未知 MCP 协议 | `fmt.Errorf("unknown mcp protocol: %s", ...)` |

调用方无法 `errors.Is(err, llm.ErrXxx)`，只能字符串匹配 —— 违反 Go 错误处理最佳实践。

### 目标

- 定义一组导出的 sentinel error 变量
- 关键错误路径使用 `errors.Wrap` 模式，保留 `%w` 链
- 调用方可以通过 `errors.Is` 精确判断错误类型

### 文件变更

- **新增**：`llm/errors.go`（约 40 行）
- **修改**：`llm.go` / `runtime_spec.go` / `model_force.go` / `adk_extraction.go` / `mcp.go` 中的错误构造（每处 1-3 行）

### 实现方案

**Step 1**：创建 `llm/errors.go`

```go
package llm

import "errors"

// Package llm 错误定义。
//
// 调用方可通过 errors.Is(err, llm.ErrXxx) 精确判断错误类型。
var (
    // ErrMissingModel 模型配置缺失或无效。
    ErrMissingModel = errors.New("llm: model is required")

    // ErrMissingAPIKey 调用所需 Provider 时缺少 API Key。
    ErrMissingAPIKey = errors.New("llm: api key is required")

    // ErrUnsupportedProtocol 不支持的 ModelProtocol。
    ErrUnsupportedProtocol = errors.New("llm: unsupported model protocol")

    // ErrInvalidConfig AgentConfig 配置无效或冲突。
    ErrInvalidConfig = errors.New("llm: invalid agent config")

    // ErrExtractionRetriesExhausted Extraction 模式重试耗尽，
    // 模型未能产出符合 StructTool JSON Schema 的参数。
    ErrExtractionRetriesExhausted = errors.New("llm: extraction retries exhausted")

    // ErrUnknownMCPProtocol 未知的 MCP 协议（仅支持 "stdio" / "sse"）。
    ErrUnknownMCPProtocol = errors.New("llm: unknown mcp protocol")

    // ErrUnsupportedContentURLScheme 多模态内容 URL 使用了非白名单协议
    //（仅允许 http / https / data）。F-4 新增。
    ErrUnsupportedContentURLScheme = errors.New("llm: unsupported content url scheme")

    // ErrMCPConnectFailed MCP 服务器连接失败。
    ErrMCPConnectFailed = errors.New("llm: mcp connect failed")
)
```

**Step 2**：替换现有点位的错误构造（示例）

```go
// llm.go: NewModel
if cfg.Model == "" {
    return nil, fmt.Errorf("new model: %w", ErrMissingModel)
}

switch cfg.Protocol {
case OPENAI, KIMI: // ...
default:
    return nil, fmt.Errorf("new model: %w (protocol=%s)", ErrUnsupportedProtocol, cfg.Protocol)
}
```

```go
// mcp.go
if cfg.Protocol != "stdio" && cfg.Protocol != "sse" {
    return nil, nil, fmt.Errorf("load mcp tools: %w (protocol=%s)", ErrUnknownMCPProtocol, cfg.Protocol)
}
```

### 测试方案

新增 `llm/errors_test.go`，使用 table-driven 测试：

```go
func TestSentinelErrors_ProgrammaticCheck(t *testing.T) {
    tests := []struct {
        name     string
        buildErr func() error
        target   error
    }{
        {
            name:     "missing model",
            buildErr: func() error { _, err := NewModel(ctx, ModelConfig{}); return err },
            target:   ErrMissingModel,
        },
        {
            name:     "unsupported protocol",
            buildErr: func() error { _, err := NewModel(ctx, ModelConfig{Protocol: 999}); return err },
            target:   ErrUnsupportedProtocol,
        },
        // ... 其他 case
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.buildErr()
            if !errors.Is(err, tt.target) {
                t.Errorf("errors.Is(err, %v) = false, want true; err=%v", tt.target, err)
            }
        })
    }
}
```

### 验收标准

- [ ] `errors.Is(err, llm.ErrMissingModel)` 等全部可正确判断
- [ ] 原有错误信息仍包含具体上下文（如 protocol 值）
- [ ] `go test ./llm -count=1 -run Errors` 全部通过
- [ ] `golangci-lint run ./llm` 无新增告警
- [ ] 无破坏性变更（错误信息文本可微调）

### 风险

**低**。仅错误构造方式变化，错误文本可能微调，影响极少数字符串匹配的调用方。

---

## O-002：多模态输入 API

### 背景

主流模型（GPT-4o、Claude 3+、Gemini）默认支持图片/音频/文件输入，`schema.Message.MultiContent` 字段也已存在，但：

- `ModelConfig` 没有多模态相关字段
- `llm` 包内**没有便利的消息构造函数**
- README 中无多模态示例
- 用户需要自己拼 `schema.Message{MultiContent: ...}`，心智负担大

参考 Eino 文档：[ChatModel 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/chat_model_guide/) 中多模态用法。

### 目标

提供便利函数，**不修改 `ModelConfig`**（多模态是消息级而非模型级属性）：

```go
msg := llm.UserImageMessage(ctx, "https://example.com/cat.png", "这是什么动物？")
resp, _ := agent.Generate(ctx, []*schema.Message{msg})
```

### 文件变更

- **新增**：`llm/message.go`（约 80 行）
- **新增**：`llm/message_test.go`（约 100 行）

### 实现方案

实际落地代码见 `llm/message.go`。关键差异：

- 函数返回 `(*schema.Message, error)`，URL 校验失败时返回 `ErrUnsupportedContentURLScheme`。
- 使用 eino v0.9 的 `schema.MessageInputPart` + `schema.UserInputMultiContent`（旧 `MultiContent` 已 Deprecated）。
- 仅允许 `http` / `https` / `data` 三种 URL scheme，防止 `file://` 等引发 SSRF。

```go
package llm

import (
    "fmt"
    "net/url"
    "strings"

    "github.com/cloudwego/eino/schema"
)

// allowedContentURLSchemes 是多模态内容 URL 允许的协议白名单。
var allowedContentURLSchemes = map[string]bool{"http": true, "https": true, "data": true}

// UserImageMessage 构造一条「图片 + 文本」用户消息。text 为空时仅含图片。
func UserImageMessage(imageURL, text string) (*schema.Message, error) {
    if err := validateContentURL(imageURL); err != nil {
        return nil, err
    }
    parts := appendTextPart([]schema.MessageInputPart{imageInputPart(imageURL)}, text)
    return &schema.Message{Role: schema.User, UserInputMultiContent: parts}, nil
}

// UserImageMessages 构造一条「多图 + 文本」用户消息，图片顺序保留。
func UserImageMessages(imageURLs []string, text string) (*schema.Message, error) {
    parts := make([]schema.MessageInputPart, 0, len(imageURLs)+1)
    for _, u := range imageURLs {
        if err := validateContentURL(u); err != nil {
            return nil, err
        }
        parts = append(parts, imageInputPart(u))
    }
    return &schema.Message{Role: schema.User, UserInputMultiContent: appendTextPart(parts, text)}, nil
}

// UserAudioMessage 构造一条「音频 + 文本」用户消息（如 Gemini / GPT-4o-audio）。
func UserAudioMessage(audioURL, text string) (*schema.Message, error) {
    if err := validateContentURL(audioURL); err != nil {
        return nil, err
    }
    part := schema.MessageInputPart{
        Type:  schema.ChatMessagePartTypeAudioURL,
        Audio: &schema.MessageInputAudio{MessagePartCommon: schema.MessagePartCommon{URL: &audioURL}},
    }
    return &schema.Message{Role: schema.User, UserInputMultiContent: appendTextPart([]schema.MessageInputPart{part}, text)}, nil
}

// validateContentURL 校验内容 URL 协议在白名单内（http/https/data）。
func validateContentURL(raw string) error {
    u, err := url.Parse(raw)
    if err != nil {
        return fmt.Errorf("%w: %q: %v", ErrUnsupportedContentURLScheme, raw, err)
    }
    if !allowedContentURLSchemes[strings.ToLower(u.Scheme)] {
        return fmt.Errorf("%w: %q (allowed: http/https/data)", ErrUnsupportedContentURLScheme, u.Scheme)
    }
    return nil
}

func imageInputPart(u string) schema.MessageInputPart {
    return schema.MessageInputPart{
        Type:  schema.ChatMessagePartTypeImageURL,
        Image: &schema.MessageInputImage{MessagePartCommon: schema.MessagePartCommon{URL: &u}},
    }
}

func appendTextPart(parts []schema.MessageInputPart, text string) []schema.MessageInputPart {
    if text == "" {
        return parts
    }
    return append(parts, schema.MessageInputPart{Type: schema.ChatMessagePartTypeText, Text: text})
}
```

**注意**：`UserInputMultiContent` 是 eino v0.9 引入的新字段，旧的 `schema.Message.MultiContent` 已被标记 Deprecated，本包跟随上游使用新字段。

### 测试方案

- 构造函数：验证 Role、MultiContent 长度、各 part 的 Type 和内容
- 多图：验证顺序正确
- 空 text：验证不追加空 TextContent
- 集成测试（默认 skip）：真实调用 OpenAI GPT-4o，验证能识别图片

### 验收标准

- [ ] `UserImageMessage` / `UserImageMessages` 可用
- [ ] `go test ./llm -run UserImage -v` 通过
- [ ] README 增加「多模态输入」示例章节
- [ ] doc.go 提及多模态支持
- [ ] 集成测试 `t.Skip` 默认跳过，但可通过环境变量启用

### 风险

**中**。Eino `schema` 包 API 可能与本提案示例不完全一致，实现前必须先确认实际类型签名。

### 前置研究（Step 0）

```
阅读 github.com/cloudwego/eino@v0.9.8/schema/message.go，确认：
- ImageURL 的实际类型（结构体 / 指针）
- TextContent 的字段名
- MultiContent 字段的元素类型（schema.UserContent 接口）
```

---

## O-003：工具层防御机制

### 背景

Eino `compose.ToolsNodeConfig` 已经提供了完整的工具防御能力（见 [ToolsNode 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/tools_node_guide/)）：

| 能力 | Eino 已提供 | go-kit 当前暴露 |
|------|-----------|---------------|
| 工具名别名（`ToolAliases`） | ✅ | ❌ |
| 未知工具兜底（`UnknownToolsHandler`） | ✅ | ❌ |
| 参数修复（`ToolArgumentsHandler`） | ✅ | ❌ |
| 工具错误转文本 Middleware | ✅ | ❌ |

当前 go-kit `ToolsConfig` 完全没有暴露这些能力。生产环境遇到模型幻觉工具名 / 返回坏 JSON 时，Agent 流程会中断。

### 目标

在 `ToolsConfig` 增加配置字段，转发给底层 Eino `ToolsNodeConfig`；并提供一个默认的"错误转文本" Middleware 选项。

### 文件变更

- **修改**：`llm/config.go`（新增约 15 行）
- **修改**：`llm/runtime_builder.go`（约 20 行）
- **新增**：`llm/tool_defense.go`（约 60 行，默认 Middleware）
- **新增**：`llm/tool_defense_test.go`（约 120 行）

### 实现方案

**Step 1**：扩展 `ToolsConfig`

```go
// config.go
type ToolsConfig struct {
    Standard   []tool.BaseTool
    Invokable  []InvokableTool
    MCPServers []MCPConfig

    // Aliases 工具名别名映射，主名 -> 别名列表。
    // 当模型输出已被废弃的工具名时，仍能路由到正确工具。
    Aliases map[string][]string

    // UnknownHandler 处理模型调用了未注册工具的情况。
    // 返回的字符串将作为 ToolResult 回传给模型，让 Agent 自行纠错。
    // 若为 nil，默认行为是返回 "unknown tool: <name>"。
    UnknownHandler func(ctx context.Context, name, args string) string

    // ArgumentsFixer 在工具调用前修复非法 JSON 参数。
    // 例如模型输出了 trailing comma、单引号等，可在此修复。
    // 若为 nil，原样透传。
    ArgumentsFixer func(ctx context.Context, toolName, args string) (string, error)

    // ErrorToText 若为 true，工具执行错误将被转换为 ToolResult 文本，
    // 而不是中断 Agent 流程。默认 true（生产推荐）。
    ErrorToText *bool
}
```

**Step 2**：在 `runtime_builder.go` 转发到 Eino

```go
func buildToolsNodeOpts(cfg ToolsConfig) []compose.ToolsNodeOption {
    var opts []compose.ToolsNodeOption
    if len(cfg.Aliases) > 0 {
        opts = append(opts, compose.WithToolAliases(cfg.Aliases))
    }
    if cfg.UnknownHandler != nil {
        opts = append(opts, compose.WithUnknownToolsHandler(func(...) string {...}))
    }
    if cfg.ArgumentsFixer != nil {
        opts = append(opts, compose.WithToolArgumentsHandler(func(...) (string, error) {...}))
    }
    return opts
}
```

**Step 3**：提供默认 Middleware（可选开关）

```go
// tool_defense.go
func defaultErrorToTextMiddleware() compose.ToolMiddleware {
    return func(next tool.InvokableTool) tool.InvokableTool {
        return &errorToTextTool{inner: next}
    }
}

type errorToTextTool struct{ inner tool.InvokableTool }

func (t *errorToTextTool) Invoke(ctx context.Context, args string) (string, error) {
    out, err := t.inner.Invoke(ctx, args)
    if err != nil {
        return fmt.Sprintf("tool execution failed: %v\nplease try different arguments or another tool.", err), nil
    }
    return out, nil
}
```

### 测试方案

- Aliases：模型输出别名 → 实际工具被调用
- UnknownHandler：调用未注册工具 → handler 返回值作为 ToolResult
- ArgumentsFixer：输入非法 JSON → 修复后正确解析
- ErrorToText：工具返回 error → Agent 收到文本而非中断
- 配置组合测试：同时启用多项防御

### 验收标准

- [x] 四项配置字段全部生效（`tool_defense.go`）
- [x] ~~`ErrorToText` 默认开启~~ → **决策变更：默认关闭（nil）**，见下方「实现说明」
- [x] 现有测试不受影响（不传新字段时行为完全一致）
- [ ] README 增加「工具层防御」章节（O-006 文档轮一并补）
- [x] 测试覆盖所有新增分支（`tool_defense_test.go`）

### 实现说明（与原方案的偏离）

1. **ErrorToText 默认改为关闭（nil）**：原方案"默认开启"与"现有行为不变"冲突——默认开启会把 Assistant 模式下"工具报错中断"变成"返回文本"，属行为破坏。依据"不破坏现有 API / 最小变更"原则，改为 `*bool` 且 nil=关闭，生产需显式 `ErrorToText: &true` 开启。
2. **两路实现统一**：react 与 ADK **都基于 `compose.ToolsNodeConfig`**，故四项防御由单一 `applyToolDefenses(*compose.ToolsNodeConfig, ToolsConfig)` 同时服务两路，无需两套 API（优于原方案预估）。
3. **eino 实际 API**：防御能力是 `ToolsNodeConfig` 的结构体字段（`ToolAliases`/`UnknownToolsHandler`/`ToolArgumentsHandler`/`ToolCallMiddlewares`），非 `WithXxx` Option。别名用 `ToolAliasConfig.NameAliases`（go-kit 暴露简化的 `map[string][]string`）。
4. **日志一致性**：ErrorToText 吞错时同步 `markError` 到 react 的 toolLogState，保证结构化日志仍记录失败。
5. **ErrorToText 同时 recover panic**：`errorToTextMiddleware` 内部用 `defer recover()` 捕获工具调用 panic（包括用户工具自身 panic 和 Eino 内部 panic），将其转为文本结果回传模型，避免单工具 panic 拖垮整个 Agent 调用。

### 风险

**已落地**。两路均基于 `compose.ToolsNodeConfig`，单函数接入；ErrorToText 默认关闭，无回归。

### 依赖

无。可独立于其他项开发。

---

## O-004：ADKAgent 运行时 Option

### 背景

`NewAgent`（react 路径）支持运行时注入：

```go
// agent.go
func (a *Agent) Generate(ctx context.Context, msgs []*schema.Message, opts ...agent.AgentOption) (*schema.Message, error)
```

调用方可以使用 `agent.WithChatModel(otherModel)` / `agent.WithChatModelOptions(...)` 在单次请求内切换模型或参数。

但 `ADKAgent` 不支持：

```go
// adk_agent.go
func (a *ADKAgent) Generate(ctx context.Context, msgs []*schema.Message) (*schema.Message, error)
```

这导致主推路径反而能力弱于 legacy 路径。

### 目标

为 `ADKAgent.Generate` / `ADKAgent.Stream` 增加可选的运行时参数，至少覆盖：

- 单次请求切换模型
- 单次请求调整模型参数（Temperature / MaxTokens / Thinking）

### 文件变更

- **修改**：`llm/adk_agent.go`（约 30 行）
- **新增**：`llm/options.go`（约 50 行，定义 Option 类型）
- **新增**：`llm/options_test.go`（约 80 行）

### 实现方案

**调研前置**：阅读 `eino/adk` 包，确认 `Runner.Run` / `AgentRunOption` 是否支持运行时模型替换。如不支持，需评估替代方案（如 clone agent 内部状态）。

**Step 1**：定义 Option 类型

```go
// options.go
type GenerateOption func(*generateConfig)

type generateConfig struct {
    model        model.ToolCallingChatModel
    temperature  *float32
    maxTokens    *int
    thinking     *ThinkingConfig
}

func WithRuntimeModel(m model.ToolCallingChatModel) GenerateOption {
    return func(c *generateConfig) { c.model = m }
}

func WithRuntimeTemperature(t float32) GenerateOption {
    return func(c *generateConfig) { c.temperature = &t }
}

func WithRuntimeMaxTokens(n int) GenerateOption {
    return func(c *generateConfig) { c.maxTokens = &n }
}
```

**Step 2**：扩展 ADKAgent 签名

```go
func (a *ADKAgent) Generate(
    ctx context.Context,
    msgs []*schema.Message,
    opts ...GenerateOption,
) (*schema.Message, error) {
    gc := &generateConfig{}
    for _, opt := range opts { opt(gc) }

    // 如果运行时指定了模型，构造临时 agent（或通过 adk.AgentRunOption 注入）
    if gc.model != nil || gc.temperature != nil || gc.maxTokens != nil {
        return a.generateWithOverride(ctx, msgs, gc)
    }
    // ... 原有逻辑
}
```

### 测试方案

- 不传 opts：行为与原版完全一致（回归测试）
- WithRuntimeModel：使用 mock 模型验证被调用
- WithRuntimeTemperature：验证 model option 正确传递
- 并发：多个 goroutine 同时调用 Generate（不同 opts），互不影响

### 验收标准

- [ ] `ADKAgent.Generate` / `Stream` 支持 `...GenerateOption`
- [ ] 不传 opts 时 100% 向后兼容
- [ ] 至少支持模型替换 + 温度调整
- [ ] README 增加「运行时参数」示例
- [ ] 同步给 `NewAgent` 路径增加一致签名的 alias（统一入口）

### 风险

**高**。需要确认 Eino ADK 是否支持运行时模型替换；如不支持，可能需要 clone agent 或放弃模型替换能力（仅保留参数调整）。

### 前置研究（Step 0）

```
阅读 github.com/cloudwego/eino@v0.9.8/adk/runner.go 和 chatmodel.go，确认：
- AgentRunOption 是否支持 WithChatModel
- 若不支持，Runner 是否可重用、是否每次 Run 都会重新构建内部状态
```

**已核查结论（eino v0.9.8）**：
- ✅ `adk.WithChatModelOptions([]model.Option)` 存在（`adk/chatmodel.go`），是 `AgentRunOption`，可透传模型参数（Temperature/MaxTokens/ToolChoice 等）→ **"运行时参数调整"可实现**。
- ⚠️ **未发现** `WithChatModel`（运行时整体换模型）的原生 option。`ChatModelAgent` 的 `model` 是构造时固定字段、首次 Run 后 `frozen`。运行时换模型只能"重建 agent+runner"，但这会重连 MCP 工具、违背预建设计。
- **决策**：O-004 仅实现"运行时参数调整"（透传 `adk.WithChatModelOptions`）；"运行时换模型"标记为「不做 / 用户改为新建 Agent 实例」，写入 ROADMAP 边界表。
- Runner 可并发复用已由 DR-004 核查确认，O-004 无需担心并发安全。

---

## O-005：暴露 ADK Middleware 注册入口

### 背景

Eino ADK 提供完整的 `ChatModelAgentMiddleware` 钩子体系（`BeforeAgent` / `BeforeModelRewriteState` / `WrapModel` / `WrapToolCall` / `AfterAgent`），已有成熟的内置 Middleware：

| Middleware | 作用 | go-kit 当前可用 |
|-----------|------|---------------|
| `ModelRetry` | 模型调用失败时自动重试 | ❌ |
| `ModelFailover` | 模型调用失败时切换备用模型 | ❌ |
| `SafeTool` | 工具错误拦截 | ❌ |
| `ContextReduction` | 上下文压缩 | ❌ |
| 自定义 Middleware | 任意扩展 | ❌ |

当前 go-kit 把 Middleware 全部写死在包内（extraction / observability / concurrency），用户无法接入 Eino 生态。

### 目标

在 `AgentConfig` 增加 `Middlewares` 字段，让用户能注册任意 `adk.ChatModelAgentMiddleware`。

### 文件变更

- **修改**：`llm/config.go`（新增 3 行）
- **修改**：`llm/adk_agent.go`（约 10 行，把 user-provided middleware 拼到链上）
- **新增**：`llm/middleware_test.go`（约 60 行）

### 实现方案

```go
// config.go
type AgentConfig struct {
    // ... 原有字段
    Middlewares []adk.ChatModelAgentMiddleware
}
```

```go
// adk_agent.go: NewADKAgent
middlewares := append([]adk.ChatModelAgentMiddleware{}, cfg.Middlewares...)

// 包内已有的 extraction / observability middleware 仍按原顺序追加，
// 但放在用户 middleware 之后，便于用户 middleware 拦截到包内行为
if spec.Execution.ToolChoice == schema.ToolChoiceForced {
    middlewares = append(middlewares, newADKExtractionMiddleware(...))
}
if obs != nil {
    middlewares = append(middlewares, newADKObservabilityMiddleware(...))
}

// 构造 ChatModelAgent 时传入
chatModelAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    // ...
    Middlewares: middlewares,
})
```

### 测试方案

- 不传 Middlewares：行为与原版一致（回归）
- 传 1 个自定义 middleware：钩子被触发
- 传多个 middleware：按顺序执行
- 与包内 middleware 共存：顺序正确（用户先，包内后）

### 验收标准

- [ ] `AgentConfig.Middlewares` 可用
- [ ] 现有测试全部通过（不传时无影响）
- [ ] README 增加「自定义 Middleware」示例，演示接入 `ModelRetry` 等
- [ ] middleware 顺序文档化（用户 vs 包内）

### 风险

**低**。ADK 已支持 `ChatModelAgentConfig.Middlewares` 字段（已用于包内 middleware），只是没暴露给用户。

---

## O-006：文档标记 NewAgent 为 Legacy

### 背景

当前 `doc.go` 已隐式提到 `NewAgent` "保留向后兼容"，但 README 没有明确说明。新用户可能误用 `NewAgent`。

### 目标

让所有文档明确传达「主推 `NewADKAgent`」信号，引导新用户用对路径。

### 文件变更

- **修改**：`llm/doc.go`（5-10 行）
- **修改**：`llm/README.md`（顶部增加明显的 Legacy 提示框 + 示例改为优先用 ADK）

### 实现方案

**Step 1**：`doc.go` 顶部加 Legacy 标记

```go
// Package llm 提供大模型 Agent 的统一封装。
//
// ⚠️ 路径选择：
//   - 新代码请使用 [NewADKAgent]（基于 eino adk.ChatModelAgent，主推路径）
//   - [NewAgent]（基于 eino react.Agent）保留向后兼容，不再演进
//   - [NewDeepAgent] 用于复杂任务的预置应用
//
// 三者共享 AgentConfig / NewModel / 三种执行模式语义。
//
// 核心能力：
//   - 多供应商模型路由（OpenAI/Claude/DeepSeek/Gemini/Ark/Ollama/Qwen 等）
//   - 三种执行模式：Conversation / Assistant / Extraction
//   - 并发控制、思考模式统一映射、MCP 工具集成、可观测性
package llm
```

**Step 2**：README 顶部增加提示框

```markdown
> ℹ️ **路径选择**
>
> - 新项目请使用 `NewADKAgent`（主推）
> - `NewAgent` 仍可用，但不再演进
> - 切换成本：仅改一个函数名，配置完全兼容
```

**Step 3**：所有示例改为 `NewADKAgent` 优先，`NewAgent` 仅出现在「Legacy 参考」章节。

### 验收标准

- [ ] `doc.go` 顶部明确标记
- [ ] README 顶部提示框 + 示例切换
- [ ] 现有 NewAgent 示例移到「Legacy」章节
- [ ] ROADMAP.md 决策记录 DR-001 被引用

### 依赖

依赖 O-007：必须先核查 `NewADKAgent` 是否已完全覆盖 `NewAgent` 能力。若有缺口，文档中需说明缺口。

---

## O-007：NewADKAgent 能力对齐核查

### 背景

既然要主推 `NewADKAgent` 并把 `NewAgent` 标 Legacy，必须确认前者**已覆盖**后者全部能力，否则会引导用户走向不归路。

### 目标

产出一份能力对照表，识别缺口（若有），并为每个缺口创建后续优化项。

### 文件变更

- **新增**：`llm/CAPABILITY_DIFF.md`（约 80 行，仅文档，无代码）

### 实施方案

**Step 1**：列出 `NewAgent`（react 路径）所有公开能力

参考 `agent.go` 全部导出方法与可配置项：
- `Generate(ctx, msgs, opts ...agent.AgentOption)` —— 含运行时 Option
- `Stream(ctx, msgs, opts ...agent.AgentOption)` —— 含运行时 Option
- `Close()`
- `ExportGraph()`
- 通过 `agent.AgentOption` 支持的运行时注入（WithChatModel / WithChatModelOptions 等）
- react 路径的中间件（compose.ToolMiddleware）

**Step 2**：列出 `NewADKAgent`（adk 路径）所有公开能力

参考 `adk_agent.go`：
- `Generate(ctx, msgs)` —— **不支持 opts**（缺口）
- `Stream(ctx, msgs)` —— **不支持 opts**（缺口）
- `Close()`
- `Agent()` —— 暴露底层 adk.Agent

**Step 3**：构造对照表

| 能力 | NewAgent（react） | NewADKAgent（adk） | 缺口处理 |
|------|------------------|-------------------|---------|
| 基础 Generate/Stream | ✅ | ✅ | — |
| 运行时模型替换 | ✅（agent.WithChatModel） | ❌ | → O-004 |
| 运行时参数调整 | ✅（agent.WithChatModelOptions） | ❌ | → O-004（ADK 侧 `adk.WithChatModelOptions` 已存在，可透传，见下） |
| ExportGraph | ✅ | ❌（adk 无等价能力） | **缺口决策见下方「ExportGraph 缺口处置」** |
| 工具 Middleware | ✅（compose.ToolMiddleware） | ✅（adk.ChatModelAgentMiddleware） | — |
| 并发控制 | ✅ | ✅ | — |
| Extraction 重试 | ✅ | ✅ | — |

**Step 4**：对每个缺口决定处理方式

- 接受缺口（如 ExportGraph，见下）
- 补齐（如运行时 Option → O-004）
- 文档化为"已知差异"

### ExportGraph 缺口处置（必须显式决策）

`Agent.ExportGraph()`（`agent.go:292`）把 react Agent 导出为 `compose.AnyGraph`，可嵌入更大的 compose 编排图。**ADK 路径无等价能力**。这与"主推 ADK、react 标 Legacy"直接冲突——一旦阶段 2 做 `llm/compose`，react 反而是唯一可 `ExportGraph` 的路径。

**已决策（2026-06）：接受缺口（候选 3）。** 包维护者确认现网无消费方依赖 `Agent.ExportGraph()`。因此：

- 不为 ADK 补 ExportGraph 等价能力。
- 将来若出现"把 Agent 嵌入 compose 编排图"的需求，用阶段 2 的 `AgentAsTool` / `GraphAsTool`（Eino 官方推荐方式）覆盖，比 ExportGraph 更干净。
- `CAPABILITY_DIFF.md` 中将 ExportGraph 记为"已知差异（接受）"，并注明替代路径。

被否决的候选（留档）：

1. ~~保留 react 作为"可嵌入编排"的特例~~ —— 会打折扣 DR-001 的"主推唯一入口"。
2. ~~为 ADK 补等价导出~~ —— 无现网需求，YAGNI。

### 运行时 Option 缺口（与 O-004 联动）

核查已确认：ADK 侧存在 `adk.WithChatModelOptions([]model.Option)`（eino v0.9.8 `adk/chatmodel.go`）作为 `AgentRunOption`，可透传到底层模型。因此 O-004 的"运行时**参数**调整"可行；但"运行时**换模型**"与 ADKAgent"构造时一次性预建 runner"的设计存在架构张力（换模型需重建 agent/runner 或重连 MCP）——该取舍在 O-004 中决策。

### 验收标准

- [ ] 产出 `CAPABILITY_DIFF.md`
- [ ] 每个缺口有明确的处理决定（ExportGraph 缺口必须三选一）
- [ ] ROADMAP.md 引用该文档

### 风险

**低**（纯文档）。但 O-007 是 P0 门禁：未完成前不得执行 O-006（标 Legacy）。

---

## O-008：流式 model.decision 补记

### 背景

设计原则写"可观测性一等公民"，但 `model.decision` 结构化日志在**两条路径的 Stream 下都不记录**：

- ADK：`adkObservedModel.Stream` 直接透传，不解析（`adk_observability.go:84-87`）
- react：observed model 的 Stream 同样不汇总决策

生产环境流式是主力场景，这是个实质监控盲区，应从 🟡 缺口提升为正式优化项。

### 目标

在流结束时聚合 StreamReader 的 chunk（拼接 tool_calls / finish_reason / usage），补记一条 `model.decision`，与非流式语义对齐。

### 实现方案

- 包装 `Stream` 返回的 `*schema.StreamReader`，在读取侧（类似 `wrapStreamWithGuard`）累积 chunk；EOF 时用 `schema.ConcatMessages`（或 eino 等价拼接）还原完整决策再记录。
- 注意：不能在 `Generate` 路径重复记录；仅在 streaming wrapper 内补。
- 两条路径分别接入（react observed model / ADK observed model）。

### 验收标准

- [x] 流式调用结束后产生与非流式同字段的 `model.decision`（含 `streaming=true`）
- [x] 不影响流的实时性（`observeStreamDecision` 边转发边累积，不缓冲整流）
- [x] react / ADK 两路均覆盖
- [x] ROADMAP 4.4 对应行 🟡 → ✅

### 实现说明

- 实现于 `stream_decision.go`：`observeStreamDecision` 包装最终输出流，EOF 时
  `schema.ConcatMessageStream` 等价（`ConcatMessages`）还原完整消息并补记 `model.decision`。
- logs 关闭时原样返回原 stream（零开销）。已过 `-race`。
- 收尾覆盖：EOF / error / 读端提前 Close（defer 中尽力记录已累积内容）。

---

## O-009：Token/Usage 用量聚合

### 背景

当前仅在 `model.decision` 记录了 `reasoning_tokens`，没有按 invocation 聚合的整体 usage 出口。生产 LLM 系统的成本核算、配额、计费几乎必然需要 prompt/completion/total/reasoning tokens 的汇总。

### 目标

提供按 invocation 聚合的用量回调或在 `agent.end` 日志补充 usage 字段。

```go
type Usage struct {
    PromptTokens     int
    CompletionTokens int
    ReasoningTokens  int
    TotalTokens      int
}
// 方案 A：ObservabilityConfig 增加 OnUsage func(ctx, Usage)
// 方案 B：在 agent.end 结构化日志补 usage.* 字段
```

### 实现方案

- 从 `schema.Message.ResponseMeta.Usage`（确认 eino 字段名）累加每轮模型调用的 usage。
- 多轮工具循环需累加，而非只取最后一轮。
- 与 O-008 共用流式 chunk 聚合逻辑（usage 通常在流末 chunk）。

### 验收标准

- [x] 每条 `model.decision`（流式 + 非流式）携带 `prompt/completion/total_tokens`（`appendUsageAttrs`）
- [ ] **未做**：跨工具轮求和到单条 `agent.end` 总计 / `OnUsage` 回调
- [x] eino 字段已确认：`schema.Message.ResponseMeta.Usage{PromptTokens,CompletionTokens,TotalTokens}`

### 实现状态（🟡 部分完成）

**已落地**：`appendUsageAttrs` 给每条 `model.decision`（两路、流式 + 非流式）附加 token 用量，
配合 `invocation_id` 可在日志侧按调用聚合。

**未落地（需后续）**：单一 `agent.end` 总计 / `ObservabilityConfig.OnUsage` 回调。
跨工具轮求和需在模型层（每次 model 调用）累加并在 `AfterAgent` 输出，属更大改动，
拆为后续优化项；当前以"每轮 usage + invocation_id"满足成本核算的基本需求。

### 风险

**低**（已落地部分）。剩余的跨轮总计为增量增强，不阻断现有能力。

---

## 执行顺序建议

```
Week 1（并行）：
  ├─ O-007（能力核查，纯文档）       [P0 门禁，优先启动]
  ├─ O-001（sentinel errors）       [独立]
  ├─ O-002（多模态 API）            [独立，需先做 Step 0 调研]
  ├─ O-003（工具防御）              [独立]
  ├─ O-005（Middleware 入口）       [独立]
  ├─ O-008（流式 model.decision）   [独立]
  └─ O-009（Usage 聚合）            [独立，建议与 O-008 合做共用流聚合]

Week 2：
  ├─ O-004（运行时 Option）         [已完成 Step 0 调研，仅做参数调整]
  └─ O-006（文档标记 Legacy）       [依赖 O-007 完成]
```

**关键路径**：O-007（门禁）→ O-006

---

## 完成定义（Definition of Done）

每个优化项必须满足：

- [ ] 实现代码 ≤ 50 行/次提交
- [ ] 测试代码 ≥ 实现代码 × 1.5
- [ ] `go build ./...` 通过
- [ ] `go test ./llm -count=1` 通过
- [ ] `golangci-lint run ./llm` 无新增告警
- [ ] 相关文档（doc.go / README.md / ROADMAP.md）同步更新
- [ ] 状态从 📋 更新为 ✅
- [ ] 单独 commit，message 格式：`llm: implement O-NNN <title>`

---

## 附录：Review 修复轮（F-1~F-7）

stage-0 O 轮代码合并后，review 发现若干实现与文档、观测或安全预期不一致的问题，本轮集中修复。

| ID | Commit | 修复内容 | 严重度 |
|----|--------|---------|--------|
| F-1 | `4336b50` | 流式 `model.decision` 日志在 EOF 之前记录，避免 EOF 后仍写日志导致顺序混乱或丢失 | 中 |
| F-2 | `4336b50` | `schema.ConcatMessageStream` 拼接错误显式返回，不再吞掉 concat 失败 | 中 |
| F-3 | `40b784b` | README quick-start 改用 `NewADKAgent`；`CAPABILITY_DIFF.md` 同步 O-004 运行时 Option 状态 | 低 |
| F-4 | `088b1cb` | 多模态内容 URL 增加协议白名单（仅 http/https/data），非法协议返回 `ErrUnsupportedContentURLScheme` | 高 |
| F-5 | `3a375c1` | `ErrorToText` 中间件增加 `recover()`，把工具调用 panic 转为文本回传模型 | 高 |
| F-6 | `8f03228` | 补齐 react 路径流式决策回归测试；新增运行时 Option 并发 `-race` 测试 | 高 |
| F-7 | `3a375c1` | 在文档中警示 `ErrorToText` 默认关闭、开启后可能泄露工具内部错误给模型的风险 | 中 |

本轮引入或加固的能力包括：URL 协议白名单与 SSRF 纵深防御、工具调用 panic recover、flaky 流式决策测试修复、README 与能力差异文档同步、react 路径流式决策回归测试、并发运行时选项的 `-race` 覆盖，以及对 `ErrorToText` 开启后可能向模型泄露内部错误的安全提示。

---

## 附录：风险登记册

| 风险 | 等级 | 缓解措施 |
|------|------|---------|
| O-002 多模态：Eino schema API 与提案不符 | 中 | Step 0 先读 eino schema 包源码 |
| O-004 运行时 Option：ADK Runner 不支持模型替换 | ~~高~~ 已澄清 | 已核查：参数调整可做（`adk.WithChatModelOptions`）；换模型不做 |
| O-003 工具防御：react 与 ADK 两套配置 API 差异 | 中 | 分别测试两条路径 |
| O-008/O-009 流聚合：各 provider usage 字段不一致 | 中 | Step 0 核查 eino schema 与 eino-ext 实现 |
| 并发复用底层执行器是否安全 | ~~高~~ 已排除 | 已核查 eino v0.9.8 源码，react/ADK 均安全（见 DR-004），无需改动 |
| 全局：sonic v1.14.1 编译失败（go-kit 已知基线问题） | 低 | 仅影响全仓 `go test ./...`，本包测试不受影响 |

---

## 附录：参考资料

- Eino 文档：https://www.cloudwego.io/zh/docs/eino/
- Eino ADK 文档：https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_preview/
- ToolsNode 使用说明：https://www.cloudwego.io/zh/docs/eino/core_modules/components/tools_node_guide/
- go-kit 内部分析报告（如有）：联系包维护者
