

# Eino ChatModel 统一封装实现 Spec（面向实现）

## 1. 目标与总原则

### 1.1 目标

实现一层 **Provider-agnostic** 的 `ChatModel` 封装，向上只暴露 Eino 标准接口：

* `BaseChatModel.Generate/Stream`
* `ToolCallingChatModel.WithTools`（返回新实例，不修改原实例，保证并发安全）([CloudWeGo][1])

### 1.2 统一契约（必须满足）

Eino 规定的接口与 Message 结构（你封装的输入/输出边界）：

* `Generate(ctx, input []*schema.Message, opts ...Option) (*schema.Message, error)`
* `Stream(ctx, input []*schema.Message, opts ...Option) (*schema.StreamReader[*schema.Message], error)`
* `WithTools(tools []*schema.ToolInfo) (ToolCallingChatModel, error)`（不可变）([CloudWeGo][1])

Message 结构关键字段（必须映射/不丢字段）：

* `Role`, `Content`
* `UserInputMultiContent`（用户输入多模态；角色限制为 User）
* `AssistantGenMultiContent`（模型输出多模态；角色限制为 Assistant）
* `ToolCalls`, `ToolCallID`
* `ResponseMeta`, `Extra`([CloudWeGo][1])

公共 Options（请求级覆盖）：

* Temperature / MaxTokens / Model / TopP / Stop
* 以及 WithTools / WithToolChoice（ToolChoice + AllowedToolNames）([CloudWeGo][1])

---

## 2. 总体架构设计

### 2.1 分层

建议分成 3 层（避免每个 Provider 重复造轮子）：

1. **Eino 接口层**（对外）

* `UnifiedChatModel` 实现 `ToolCallingChatModel`
* 负责：合并 Config + RequestOptions、做通用校验、调用 ProviderAdapter

2. **内部统一语义层**（对内通用）

* `InternalMessage`（统一 role/content/parts/toolcalls）
* `InternalToolSpec`（统一 ToolInfo -> 函数/工具声明）
* `InternalDelta`（流式事件统一）

3. **Provider 适配层**（每家一个 Adapter）

* `BuildRequest()`：Internal -> ProviderRequest
* `ParseResponse()`：ProviderResponse -> InternalMessage + meta/extra
* `StreamDecoder()`：Provider SSE/Event -> InternalDelta

### 2.2 必须实现的通用能力模块

* `MessageMapper`：schema.Message <-> InternalMessage
* `ToolingMapper`：schema.ToolInfo -> Provider Tool Schema（各家差异点集中在这里）
* `StreamAccumulator`：把 delta 聚合成可吐给上游的 `schema.Message`

---

## 3. 统一数据模型（Internal）

### 3.1 InternalMessage（建议）

```go
type InternalRole string
const (
  RoleSystem InternalRole = "system"
  RoleUser   InternalRole = "user"
  RoleAssistant InternalRole = "assistant"
  RoleTool   InternalRole = "tool"
)

type InternalPartType string
const (
  PartText  InternalPartType = "text"
  PartImage InternalPartType = "image"
  PartAudio InternalPartType = "audio"
  PartVideo InternalPartType = "video"
  PartFile  InternalPartType = "file"
)

type InternalPart struct {
  Type InternalPartType
  Text string
  // 统一表示资源：url / base64 / bytes + mimetype
  MIME string
  URL  string
  Base64 string
  Bytes []byte
}

type InternalToolCall struct {
  ID string
  Name string
  ArgumentsJSON string // 聚合后的 JSON string
}

type InternalMessage struct {
  Role InternalRole
  Text string
  Parts []InternalPart

  ToolCalls []InternalToolCall
  ToolCallID string // tool 消息回填

  // meta & extra
  Usage *TokenUsage
  FinishReason string
  Extra map[string]any
}
```

### 3.2 schema.Message -> InternalMessage 映射规则（硬规则）

* `Content` -> `Text`
* `UserInputMultiContent`（User only）-> `Parts[]`([CloudWeGo][1])
* `AssistantGenMultiContent`（Assistant only）-> `Parts[]`（输出场景）([CloudWeGo][1])
* `ToolCalls`（assistant）-> `ToolCalls[]`([CloudWeGo][1])
* `ToolCallID`（tool）-> `ToolCallID`([CloudWeGo][1])
* `ResponseMeta/Extra`：保持不丢（Extra 允许挂 provider request_id、cache 命中信息等）

### 3.3 Tool calling 统一语义

Eino 侧的约束是：assistant 产生 `ToolCalls`，tool 消息用 `ToolCallID` 回填结果([CloudWeGo][1])
因此你的封装必须能支持如下链路：

1. 输入：system/user/history
2. assistant 输出：ToolCalls（name + arguments JSON）
3. 你外部执行工具得到 result JSON
4. 再喂回：`Role=tool` 且带 `ToolCallID` 的 message
5. 再次调用模型，得到最终自然语言

---

## 4. 通用 Options 合并策略

### 4.1 两级配置

* **InitConfig（实例级默认）**：Provider client/鉴权/baseURL/timeout/默认 model/默认采样参数
* **RequestOptions（请求级）**：来自 `opts ...Option`，覆盖 InitConfig([CloudWeGo][1])

### 4.2 ToolChoice（重要）

Eino Option 支持：

* `WithTools(tools)`：请求级绑定工具
* `WithToolChoice(toolChoice, allowedToolNames...)`：强制/允许子集([CloudWeGo][1])
  你的封装要把这两项**合并成 Provider request 的 tool 配置**（差异在 ProviderAdapter）。

---

## 5. 流式（Stream）实现规范

### 5.1 输出语义

`Stream()` 返回 `*schema.StreamReader[*schema.Message]`([CloudWeGo][1])
你需要在内部维护一个 `Accumulator` 把 Provider delta 聚合为：

* 文本增量（append）
* 工具调用增量（arguments 分片拼接）
* meta/usage（可能在流末尾才给）

### 5.2 Accumulator（建议结构）

```go
type Accumulator struct {
  role InternalRole
  text strings.Builder
  toolCalls map[string]*InternalToolCall // id->call
  finishReason string
  usage *TokenUsage
  extra map[string]any
}

func (a *Accumulator) Apply(d InternalDelta) (emit *InternalMessage, ok bool) {
  // ok==true 表示可以吐一次给上游（每 token / 每 event / 节流策略）
}
```

---

## 6. Provider 配置字段对照表（逐页“新方案点”落地）

> 说明：下表是**你封装层需要暴露/支持**的字段；字段来源均来自各子页的 Config/ChatModelConfig 定义段落。

### 6.1 OpenAI（含 Azure OpenAI）

OpenAI `openai.ChatModelConfig` 关键字段：([CloudWeGo][2])

* 通用：

  * `APIKey string`
  * `Timeout time.Duration`
  * `HTTPClient *http.Client`
* Azure 专用：

  * `ByAzure bool`
  * `AzureModelMapperFunc func(model string) string`
  * `BaseURL string`（Azure endpoint）
  * `APIVersion string`
* 备注：页面还给了多模态、流式、工具调用、音频生成、结构化输出示例入口（封装可分期实现，但要留扩展点）([CloudWeGo][2])

**封装建议**

* 抽一个 `OpenAICompatibleAdapter`，Qwen/DeepSeek（部分）可复用。
* Azure 分支：在 adapter 内根据 `ByAzure` 改 endpoint/path/header/部署名映射。

---

### 6.2 Qwen（DashScope OpenAI-compatible mode）

`qwen.ChatModelConfig`（基本与 OpenAI chat completion 参数一致）([CloudWeGo][3])

* `APIKey`, `Timeout`, `HTTPClient`
* `BaseURL`（示例为兼容模式 URL）
* `Model`
* `MaxTokens *int`, `Temperature *float32`, `TopP *float32`, …（后续字段同 OpenAI 风格）([CloudWeGo][3])

**封装建议**

* 复用 `OpenAICompatibleAdapter`：差异点集中在 BaseURL 与鉴权 header。

---

### 6.3 DeepSeek

DeepSeek 文档强调 `Path` 可配置，默认 `"chat/completions"`([CloudWeGo][4])

* `Path string`（可写 `"/c/chat/"` 或 baseURL 后任意路径）([CloudWeGo][4])
* 其余字段对应 DeepSeek chat API 参数（Model/MaxTokens 等）([CloudWeGo][4])

**封装建议**

* 仍可走 OpenAI-compatible 的 message/tool 形态，但要把 path 拼接策略做成可配置。

---

### 6.4 ARK（Volcengine Ark Runtime）

`ark.ChatModelConfig`（服务级 + 请求级混在一个 struct 里）([CloudWeGo][5])

* 服务配置：

  * `Timeout *time.Duration`（默认 10 min）
  * `RetryTimes *int`（默认 2）
  * `BaseURL string`（默认 `https://ark.cn-beijing.volces.com/api/v3`）
  * `Region string`（默认 `cn-beijing`）([CloudWeGo][5])
* 页面还说明该包提供：ChatModel + ImageGenerationModel + ResponseAPI([CloudWeGo][5])

**封装建议**

* Adapter 要支持：OpenAI-like chat completion（ARK API 兼容度较高）+ 扩展 Extra（如有 request_id/trace）
* ImageGeneration/ResponseAPI：建议单独模块，不要污染 ChatModel 接口。

---

### 6.5 ARKBot

`arkbot.Config`（比 ARK 多鉴权方式与返回 extra）([CloudWeGo][6])

* `Timeout *time.Duration`, `HTTPClient *http.Client`, `RetryTimes *int`
* `BaseURL`, `Region`
* 鉴权：`APIKey` 优先；或 `AccessKey/SecretKey`
* 请求参数：`Model string` 等（文档后续继续列）([CloudWeGo][6])

**“新方案点”：Extra 解析 helper**
文档示例明确从 `*schema.Message` 里取：

* `GetArkRequestID(msg)`
* `GetBotUsage(msg)`
* `GetBotChatResultReference(msg)`([CloudWeGo][6])

**封装建议**

* 统一把这些信息塞进 `Message.Extra`：

  * `extra["provider"]="arkbot"`
  * `extra["request_id"]=...`
  * `extra["bot_usage"]=...`
  * `extra["reference"]=...`

---

### 6.6 Qianfan（百度千帆）

`qianfan.ChatModelConfig` 关键字段（节选）([CloudWeGo][7])

* `Model string`
* Retry 相关：

  * `LLMRetryCount *int`
  * `LLMRetryTimeout *float32`
  * `LLMRetryBackoffFactor *float32`
* `Temperature *float32`（默认 0.95，范围 (0,1.0]）([CloudWeGo][7])

**封装建议**

* Qianfan SDK/HTTP 细节放 adapter 里
* 把 Retry/backoff 作为“实例级策略”，请求级允许覆盖（比如通过 ExtraOption 或自定义 Option）。

---

### 6.7 Gemini（Google genai）

`gemini.Config` 字段（节选）([CloudWeGo][8])

* `Client *genai.Client`
* `Model string`
* `MaxTokens *int`, `Temperature *float32`, `TopP *float32`, `TopK *int32`
* `ResponseSchema *openapi3.Schema`（结构化输出）
* `EnableCodeExecution bool`
* `SafetySettings []*genai.SafetySetting`
* `ThinkingConfig *genai.ThinkingConfig`
* **Cache *CacheConfig**（TTL/ExpireTime）([CloudWeGo][8])

**“新方案点”：显式前缀缓存**

* 支持 `CreatePrefixCache` 创建缓存
* 后续请求用 `gemini.WithCachedContentName(cacheInfo.Name)` 复用
* 使用缓存时，请求会省略系统指令和工具，依赖缓存前缀([CloudWeGo][8])

**封装建议**

* 在 Unified 层提供可选扩展接口（不改 ToolCallingChatModel）：

  * `type PrefixCacheCapable interface { CreatePrefixCache(...) }`
* `cached_content_name` 放到 RequestOptions 里（Gemini 专属 option）。

---

### 6.8 Ollama（本地/自部署）

`ollama.ChatModelConfig`（节选）([CloudWeGo][9])

* `BaseURL string`
* `Timeout time.Duration`
* `HTTPClient *http.Client`
* `Model string`
* `Format json.RawMessage`
* `KeepAlive *time.Duration`
* `Options *Options`（含 TopK/TopP/NumPredict/Seed/…）
* `Thinking *ThinkValue`（Thinking 模式）([CloudWeGo][9])

**封装建议**

* Options 透传：不要强行“统一成 OpenAI 参数名”，保持 Ollama 原生参数可用（通过 ExtraOptions 或 provider-specific options）。

---

### 6.9 Claude（Anthropic / AWS Bedrock）

Claude `claude.Config`（节选）([CloudWeGo][10])

* Bedrock 分支：

  * `ByBedrock bool`
  * `AccessKey`, `SecretAccessKey`, `SessionToken`, `Profile`, `Region`
* Anthropic 分支：

  * `BaseURL *string`
  * `APIKey string`
* 请求参数：

  * `Model string`
  * `MaxTokens int`
  * `Temperature *float32`
  * `StopSequences []string`
  * `Thinking *Thinking`
  * `DisableParallelToolUse *bool`
  * `HTTPClient *http.Client`([CloudWeGo][10])

**封装建议**

* Tool calling：Claude 的 tool schema/事件与 OpenAI 不同，必须独立 mapper/stream decoder。
* Thinking：提供 provider-specific Option（示例里有 `claude.WithThinking(...)` 用法）([CloudWeGo][10])

---

## 7. 统一请求/响应映射（按 Eino 语义）

> 这里不强行给每家贴具体 JSON（因为各家会变），而是定义你封装必须表达的“语义字段”。ProviderAdapter 负责把语义落到真实 JSON。

### 7.1 请求语义字段（InternalRequest）

* `model`
* `messages[]`（role + content/parts）
* sampling：

  * `temperature`, `top_p`, `top_k`
  * `max_tokens`
  * `stop[]`
* tools：

  * `tools[]`（函数声明）
  * `tool_choice`（auto/none/forced + allowed names）
* extra：

  * `timeout/retry`
  * `thinking`
  * `cache_name`（Gemini）
  * `keepalive/options`（Ollama）

### 7.2 响应语义字段（InternalResponse）

* `assistant_message`（text + parts）
* `tool_calls[]`（id/name/arguments_json）
* `usage`（prompt/completion/total）
* `finish_reason`
* `extra`（request_id、bot_usage、reference、cache_hit 等）

---

## 8. Go 伪代码骨架（可以直接当实现蓝图）

### 8.1 ProviderAdapter 接口

```go
type ProviderAdapter interface {
  Name() string

  // BuildRequest: internal -> provider req
  BuildRequest(ctx context.Context, req InternalRequest) (any, error)

  // Do: 非流式调用
  Do(ctx context.Context, providerReq any) (providerResp any, err error)

  // ParseResponse: provider resp -> internal message
  ParseResponse(ctx context.Context, providerResp any) (InternalMessage, error)

  // Stream: 流式调用，返回 provider event reader
  Stream(ctx context.Context, providerReq any) (ProviderStream, error)

  // DecodeEvent: provider event -> internal delta
  DecodeEvent(evt any) (InternalDelta, error)
}

type ProviderStream interface {
  Recv() (any, error) // provider event
  Close() error
}
```

### 8.2 UnifiedChatModel（实现 ToolCallingChatModel）

```go
type UnifiedChatModel struct {
  adapter ProviderAdapter
  initCfg InitConfig

  boundTools []*schema.ToolInfo
}

func (m *UnifiedChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
  // 必须返回新实例，不修改原实例
  cp := *m
  cp.boundTools = cloneTools(tools)
  return &cp, nil
}
```

### 8.3 Generate（非流式）

```go
func (m *UnifiedChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
  ro := MergeOptions(m.initCfg, opts...) // 合并公共 option + provider 专属 option
  internalMsgs, err := MapSchemaToInternal(input)
  if err != nil { return nil, err }

  // 合并 tools：WithTools 绑定 + opts WithTools
  tools, toolChoice := ResolveTools(m.boundTools, ro)

  ireq := InternalRequest{
    Model: ro.ModelOrDefault(),
    Messages: internalMsgs,
    Temperature: ro.Temperature,
    TopP: ro.TopP,
    MaxTokens: ro.MaxTokens,
    Stop: ro.Stop,
    Tools: MapTools(tools),
    ToolChoice: toolChoice,
    Extra: ro.ProviderExtra,
  }

  preq, err := m.adapter.BuildRequest(ctx, ireq)
  if err != nil { return nil, err }

  presp, err := m.adapter.Do(ctx, preq)
  if err != nil { return nil, err }

  imsg, err := m.adapter.ParseResponse(ctx, presp)
  if err != nil { return nil, err }

  out := MapInternalToSchema(imsg)
  // out.Extra 合并：adapter extra + 你统一填的 provider/name/request_id 等
  return out, nil
}
```

### 8.4 Stream（流式）

```go
func (m *UnifiedChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
  ro := MergeOptions(m.initCfg, opts...)
  internalMsgs, err := MapSchemaToInternal(input)
  if err != nil { return nil, err }

  tools, toolChoice := ResolveTools(m.boundTools, ro)

  ireq := InternalRequest{ /* 同 Generate */ }

  preq, err := m.adapter.BuildRequest(ctx, ireq)
  if err != nil { return nil, err }

  pstream, err := m.adapter.Stream(ctx, preq)
  if err != nil { return nil, err }

  acc := NewAccumulator()
  sr := schema.NewStreamReader[*schema.Message](func() (*schema.Message, error) {
    evt, err := pstream.Recv()
    if err != nil { return nil, err } // io.EOF 由上层处理

    d, err := m.adapter.DecodeEvent(evt)
    if err != nil { return nil, err }

    imsg, ok := acc.Apply(d)
    if !ok { return nil, schema.ErrSkip } // 或者内部循环读到 ok 为止

    smsg := MapInternalToSchema(*imsg)
    return smsg, nil
  }, func() error {
    return pstream.Close()
  })

  return sr, nil
}
```

### 8.5 ToolCall 流式聚合关键点

* arguments 可能分片到达：必须按 tool_call_id 聚合成完整 JSON 字符串再吐给上游。
* 如果 Provider 没有 tool_call_id：你要生成一个稳定 id（例如序号 + hash），并确保 tool 消息回填能对应上。

---

## 9. Provider “专属扩展”如何不污染统一接口

### 9.1 推荐：ProviderOption（放到 ExtraOptions）

* Gemini：

  * `WithCachedContentName(name)`（复用前缀缓存）([CloudWeGo][8])
  * `CreatePrefixCache(messages, WithTools(...))`（能力接口）([CloudWeGo][8])
* Ollama：

  * `Options`（TopK/TopP/NumPredict/Seed/…）与 `Thinking` 透传([CloudWeGo][9])
* Claude：

  * `WithThinking(...)`，以及 Bedrock 分支鉴权字段([CloudWeGo][10])
* ARKBot：

  * 将 request_id / bot_usage / reference 塞入 Extra，并提供 helper 或统一 key([CloudWeGo][6])

### 9.2 推荐的 Extra Key 规范（统一观测）

```text
extra["provider"]           = "openai|qwen|deepseek|ark|arkbot|qianfan|gemini|ollama|claude"
extra["request_id"]         = "..."
extra["usage_raw"]          = any
extra["cache"]              = map[string]any{"name":..., "hit":..., "ttl":...}
extra["vendor"]             = map[string]any{...} // 兜底
```

---

## 10. 验收用例（必须过）

1. 文本 Generate / Stream（所有 provider）([CloudWeGo][1])
2. WithTools + ToolCalls 输出 + tool 回填后继续对话([CloudWeGo][1])
3. 多模态输入（至少 1 家跑通：OpenAI/Qwen/Gemini/Ollama/Claude 任一）：

* OpenAI 有“多模态支持(图片理解)”示例段落([CloudWeGo][2])

4. Gemini 前缀缓存：

* CreatePrefixCache + WithCachedContentName 跑通([CloudWeGo][8])

5. ARKBot Extra：

* request_id / bot_usage / reference 可从 Extra 取到([CloudWeGo][6])

