# llm 包重构设计

## 背景

当前 `llm` 包定位为基于 Eino 的单 Agent 高层封装，已经实现了多模型统一路由、工具增强对话、强制工具调用、失败后让模型修正参数重试、工具结果直返、结构化输出提取、MCP 工具接入、日志与 Langfuse 观测等能力。

本次重构的前提是：

- `llm` 包尚未被外部项目正式依赖
- 允许较大幅度调整公开 API
- 当前已实现的业务效果必须完整保留

因此目标不是“兼容旧字段名”，而是“重做 API，同时保留当前能力闭环”。

## 当前能力总结

当前 `llm` 包已经具备以下效果：

- 统一 OpenAI / Claude / Gemini / Qwen / ARK / DeepSeek / Ollama / Qianfan / Kimi 等模型入口
- 基于 `react.Agent` 提供单 Agent 高层 API
- 提供 `SystemPrompt`、每轮消息修改、历史消息改写能力
- 提供工具自动调用、强制调用、禁止调用能力
- 在强制工具调用场景下，把工具错误反馈给模型，让模型修正参数后重新调用
- 在指定工具成功后直接返回原始工具结果，避免多一次模型总结
- 通过 `StructTool` 实现结构化输出提取
- 通过 MCP 动态接入外部工具
- 提供 Slog / Langfuse 观测能力
- 支持流式调用和自定义 tool call 检测器

这些能力里，真正有业务价值的不是某个字段或某个包装器，而是以下效果：

- 普通对话
- 工具增强对话
- 必须完成工具任务
- 工具失败自修复
- 工具成功后按策略返回
- 结构化结果提取
- 流式场景下的模型差异适配

## 设计目标

- 保留 `llm.NewAgent` 作为主入口
- 保持 `llm` 面向业务场景的单 Agent 高层 API 定位
- 不再将底层实现细节直接暴露为对外主概念
- 对外以“能力声明”组织配置，而不是平铺一组控制旋钮
- 内部保留当前能力闭环，并为后续扩展留出余量

## 非目标

- 不把 `llm` 直接改造成通用 ADK runtime 封装
- 不把 Middleware / Handler 作为业务方必须理解的主抽象
- 不在本次重构中引入多 Agent 编排作为主路径
- 不为了追求“新 API”而删除当前已有业务效果

## 核心结论

采用“对外高层语义重组，对内执行层重构”的方向：

- 对外继续保留单 Agent 高层 API
- 对外 `AgentConfig` 改为按能力分组
- 对内继续以 `react.Agent` 作为循环执行内核
- 对内新增 `RuntimeSpec` 编译层和执行控制层
- 以三种业务模式统一承载当前 `ToolChoice + MaxRetries + ToolReturnDirectly` 的效果

## 设计原则

### 1. 保留效果，不保留旧字段形态

重构后的目标是保留“现有效果”，而不是兼容当前平铺字段布局。只要业务效果完整保留，旧字段名可以消失或重组。

### 2. 对外暴露业务语义，对内隐藏实现细节

对业务方重要的是“这是普通对话还是结构化提取”，不是“内部用了状态机还是 middleware”。因此新 API 应围绕业务模式和能力分组组织。

### 3. 单 Agent 高层 API 不变

外部入口继续保持：

```go
agent, err := llm.NewAgent(ctx, llm.AgentConfig{...})
```

但 `AgentConfig` 的结构将重做。

### 4. `react.Agent` 继续作为默认底层执行内核

当前最重要的能力闭环仍建立在 `react.Agent` 语义上，且 `v0.8.x` 并没有让这层能力消失。因此本次重构继续基于 `react.Agent`，而不是切到 ADK-first。

## 新的 AgentConfig 分层

新的 `AgentConfig` 按 6 组能力组织：

### 1. Model

负责：

- 模型协议与实例选择
- 默认生成参数
- 运行时模型覆盖

对应当前：

- `ModelConfig`
- `Model`

### 2. Prompt

负责：

- 静态 system prompt
- 每轮临时消息预处理
- 历史消息持久改写

对应当前：

- `SystemPrompt`
- `MessageModifier`
- `MessageRewriter`

### 3. Tools

负责：

- 标准工具
- 简化工具接口
- MCP 工具来源
- 结构化输出工具

对应当前：

- `Tools`
- `InvokableTools`
- `NewMCPTools`
- `StructTool`

### 4. Execution

负责：

- 工具调用模式
- 工具失败修复策略
- 最大执行步数
- 最终返回策略

对应当前：

- `ToolChoice`
- `MaxRetries`
- `MaxStep`
- `ToolReturnDirectly`

### 5. Streaming

负责：

- 流式 tool call 检测
- 流式 direct return 策略
- 模型差异适配

对应当前：

- `StreamToolCallChecker`

### 6. Observability

负责：

- callback handlers
- Langfuse
- slog 日志

对应当前：

- `Callbacks`
- `NewLangfuseHandler`
- `NewLogHandler`

## Execution 模式设计

`Execution` 是重构核心，建议引入三种业务模式：

### 1. Conversation

语义：

- 不使用工具
- 直接模型回答

覆盖当前：

- `ToolChoiceForbidden`
- 无工具场景

### 2. Assistant

语义：

- 工具可用
- 模型自行决定是否调用工具
- 工具成功后默认回模型总结
- 可对指定工具开启直返

覆盖当前：

- 默认模式
- `ToolChoiceAllowed`

### 3. Extraction

语义：

- 必须调用目标工具完成任务
- 工具失败后将错误反馈给模型修正参数再试
- 成功后默认直接返回工具结果
- 可选成功后回模型做自然语言包装

覆盖当前：

- `ToolChoiceForced`
- `MaxRetries`
- `ToolReturnDirectly`
- `StructTool` 主场景

## 内部架构

### 1. AgentConfig -> RuntimeSpec

新增内部编译层，将对外配置编译为统一运行规格：

- 模式
- Prompt 策略
- 工具列表
- 失败修复策略
- 结果返回策略
- Streaming 策略
- 观测策略

### 2. RuntimeSpec -> Builder

Builder 负责组装：

- 模型
- 工具
- `react.Agent` 配置
- 工具调用中间逻辑
- 流式检测器
- 观测桥接

### 3. react.Agent

继续承担：

- chat -> tool -> chat 循环
- 消息状态维护
- 原生流式处理
- 原生 direct return / stream checker 能力

### 4. 执行控制层

保留并重构当前真正有价值的自定义逻辑：

- 工具调用模式控制
- 工具失败后反馈模型修正参数
- 指定工具成功后直接返回
- 结构化提取场景优化
- 流式场景特殊模型适配

## Middleware 的定位

Middleware / Handler 可继续使用，但只作为内部实现手段，不作为业务 API 核心概念。

适合放在内部 middleware 的能力：

- 工具调用成功 / 失败记录
- 工具失败转反馈结果
- 工具成功后触发 direct return
- 流式 chunk 检测与改写
- 日志 / trace / token 统计
- provider request / response 定制化适配

不适合作为外部主抽象的能力：

- 强制工具调用
- 结构化输出提取
- 工具失败修复
- 结果直返策略

这些都应保持为高层业务语义。

## 依赖升级策略

本次重构采用“选择性升级”，不建议无脑全升。

### 第一批建议升级

- `github.com/cloudwego/eino` -> `v0.8.4`
- `github.com/cloudwego/eino-ext/components/model/openai` -> `v0.1.10`
- `github.com/cloudwego/eino-ext/components/model/qwen` -> `v0.1.6`
- `github.com/cloudwego/eino-ext/components/model/gemini` -> `v0.1.29`

理由：

- `eino v0.8.4` 作为稳定底座
- `openai v0.1.10` 增加 request / response modifier 扩展点
- `qwen v0.1.6` 跟进底层 openai ACL 版本
- `gemini v0.1.29` 修正 tool call part 顺序问题

### 第二批暂缓

- `ark -> v0.1.65`
- `langfuse`
- `mcp`
- arkbot / claude / deepseek / qianfan / ollama

理由：

- `ark v0.1.65` 改了工具绑定时默认 tool choice 行为，应在新的执行层稳定后再评估
- 其余依赖对本次重构不是关键路径

## 验证要求

重构完成后必须验证以下能力闭环：

- 普通对话
- 自动工具调用
- 强制工具调用
- 工具失败后模型修正再试
- 指定工具成功后直返
- 结构化输出提取
- 流式 tool call 检测
- callback / 日志 / Langfuse 集成
- MCP 工具加载
- 运行时模型覆盖

## 风险与约束

- 如果上游新版本仍不能原生覆盖“强制工具调用后把工具错误回灌模型”的链路，内部仍需保留自定义执行控制逻辑
- 如果流式模型在 tool call 输出顺序上仍存在差异，`Streaming` 配置下的检测器能力不能删除
- 如果未来需要引入 ADK，更合理的方式是把它作为可替换执行后端，而不是本次重构直接切换主抽象

## 最终建议

重构方向采用：

- 对外：业务单 Agent 高层 API
- 对外：能力分组配置
- 对内：`RuntimeSpec + Builder + react.Agent`
- 对内：保留自定义执行控制层
- 升级策略：先升 Eino / OpenAI / Qwen / Gemini，暂缓 Ark

这条路线可以在不丢失当前能力效果的前提下，显著优化 API 结构、内部边界和后续扩展能力。
