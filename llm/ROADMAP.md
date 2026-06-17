# llm 包演进路线图（ROADMAP）

> 本文档记录 `llm` 包的定位、能力清单、演进路线与重要决策。  
> **维护规则**：每完成一项能力或优化，必须同步更新对应条目的状态标记（见末尾"图例"）。

---

## 一、包定位

### 1.1 做什么

`llm` 包是 **go-kit 对 LLM Agent 场景的强意见封装**，基于 [CloudWeGo Eino](https://www.cloudwego.io/zh/docs/eino/) 框架。

核心产品化概念：

- **多供应商模型路由**：通过 `ModelConfig.Protocol` 统一 12+ 厂商
- **三种执行模式**：`Conversation` / `Assistant` / `Extraction`
- **Agent 工厂**：`NewADKAgent`（主推）/ `NewAgent`（legacy）/ `NewDeepAgent`（预置应用）

### 1.2 不做什么（明确边界）

| 不做 | 原因 | 替代方案 |
|------|------|---------|
| 编排框架（Chain/Graph/Workflow） | Eino `compose` 已完整实现 | 直接使用 `github.com/cloudwego/eino/compose` |
| RAG 组件链（Embedding/Retriever） | 当前无业务需求 | 阶段 3 评估是否引入 `llm/rag` 子包 |
| 自研模型 SDK | 已有 eino-ext | 通过 `NewModel` 转发 |
| 全局单例（`Configure` / `GetClient`） | Agent 是有状态对象（持工具/并发/cleanup） | 显式实例 + 显式生命周期管理 |
| 自研流式抽象 | Eino `StreamReader[T]` 已成熟 | 直接复用 |

### 1.3 与 Eino 的关系

```
Eino（框架）            llm（产品封装）
─────────────           ──────────────
compose（编排）         ❌ 不暴露
adk（Agent 框架）       ✅ 包装为 NewADKAgent
adk/prebuilt/deep       ✅ 包装为 NewDeepAgent
components/model        ✅ 包装为 NewModel
components/tool         ✅ 通过 ToolsConfig 暴露
flow/agent/react        ⚠️ 仅 legacy（NewAgent）
callbacks               ✅ 通过 ObservabilityConfig 暴露
```

---

## 二、设计原则

1. **产品 > 框架**：暴露最小必要的 API，把 Eino 的复杂性压缩成可 opinionated 的入口
2. **ADK 路径优先**：新能力默认基于 `adk.ChatModelAgent`，`react.Agent` 仅维护向后兼容
3. **配置层共享**：`NewAgent` / `NewADKAgent` / `NewDeepAgent` 共享 `AgentConfig` / `NewModel` / 执行模式语义
4. **失败可恢复**：所有错误显式返回，禁用 `_` 忽略；资源 cleanup 通过 `Close()` 暴露
5. **可观测性一等公民**：结构化日志 + Eino Callback 双链路
6. **不堵死未来**：当前不引入编排/RAG，但保留接口扩展空间

---

## 三、决策记录（Decision Records）

### DR-001：未来主推 `NewADKAgent`，`NewAgent` 标记为 Legacy

- **决策**：`NewAgent`（基于 `eino/flow/agent/react`）保留但不再演进；新能力只在 `NewADKAgent`（基于 `eino/adk`）路径实现
- **措辞边界（重要）**：本决策是 **go-kit 自身的产品取向**，**不是** "eino 已废弃 react"。截至 eino v0.9.8，`flow/agent/react` **未标 `Deprecated`、仍在演进**（例如用 `ToolCallingModel` 取代旧 `Model` 字段），官方也未给迁移指南。文档中只说"go-kit 主推 ADK / NewAgent 为 go-kit legacy"，不得宣称"eino 废弃了 react"。
- **理由（已核查证据，2026-06）**：
  - ADK 的 `ChatModelAgent` **内部自带独立 ReAct 实现**（`adk/react.go`，状态版本 v0.7→v0.8，含 checkpoint/interrupt/resume、AgenticMessage、deferred tools），**完全不复用 `flow/agent/react`** —— 上游把 react 在 ADK 内重写了一遍，是最强投入信号。
  - 官方文档定位 ADK 为"complete, flexible, and powerful agent development framework"，且**仅 ADK 提供多 Agent 编排原语**（Supervisor / Plan-Execute / Group-Chat）；`flow/agent/react` 被归在旧的 "Flow Integration Components"。
  - 两者接口能力高度重叠（共享 `AgentConfig`、相同三种执行模式、相同 Generate/Stream 语义）。
  - ADK 提供 Middleware 钩子体系，扩展性更强。
- **影响**：
  - `doc.go` / `README.md` 顶部需明确标注 `NewAgent` 为 go-kit legacy（措辞按上方边界）。
  - 必须保证 `NewADKAgent` 能力 ≥ `NewAgent`（见 O-007 能力对齐核查）。
  - 现有使用 `NewAgent` 的代码无需改动（向后兼容承诺）。
  - 因上游 react 仍维护，向后兼容风险低。

### DR-002：`NewDeepAgent` 不是 `NewADKAgent` 的"升级版"

- **决策**：将 `NewDeepAgent` 定位为"基于 ADK 能力的**预置应用模板**"，而非通用框架的下一版
- **理由**：
  - `NewADKAgent` 是**通用框架**（任意 Agent 场景）
  - `NewDeepAgent` 是**特定场景产品**（`ChatModelAgent + write_todos + task + FileSystem + Shell + 预置 prompt`）
  - 两者是"基础能力 vs 应用模板"的关系，不是版本演进
- **影响**：
  - 文档与示例中不再用"升级"、"高级版"等表述
  - 复杂任务场景下，用户可在「直接用 `NewDeepAgent`」与「基于 `NewADKAgent` 自组装」之间选择

### DR-003：暂不引入 `llm/compose` 与 `llm/rag` 子包

- **决策**：阶段 0~1 不引入编排和 RAG 子包
- **理由**：
  - 当前业务场景聚焦"单 Agent 与模型交互"
  - Eino 的 `compose` 与 RAG 组件已可直接使用，无需薄封装
  - 避免 YAGNI（You Aren't Gonna Need It）
- **触发条件**（满足任一即重新评估）：
  - 业务出现"需要编排多个 Agent 实例"的场景 → 评估 `llm/compose`
  - 业务出现"知识库问答 / 文档检索增强" → 评估 `llm/rag`
  - 多个使用方重复实现同一种编排模式 → 抽象为 helper

### DR-004：底层执行器可并发复用，`concurrencyGuard` 是限流器而非正确性机制

- **决策**：`Agent` / `ADKAgent` 在构造时各建一份底层执行器（react 是 `compose.Runnable`，ADK 是 `adk.Runner` + 共享的 `adk.Agent`），之后所有并发 `Generate` / `Stream` **共享复用**该执行器，不每次重建。
- **依据**（已核查 eino v0.9.8 源码）：
  - ADK：`Runner.Run` 每次新建 `flowAgent` 与 run context；`ChatModelAgent` 用 `sync.Once` 初始化 `run`/`exeCtx` 后只读（`adk/chatmodel.go:1366`）；per-run 可变状态存活在 context（`typedChatModelAgentExecCtx`）而非结构体；`frozen` 标志仅守卫结构性 setter（设置子/父 Agent，`adk/chatmodel.go:679`），不影响并发 Run。
  - react：`Generate` / `Stream` 直接调用构造时 `Compile` 出的 `compose.Runnable`（`flow/agent/react/react.go:481`），无 per-call 结构体状态。
- **影响**：
  - `ConcurrencyConfig.MaxConcurrency` 的语义是**限流**（控成本 / 控上游 QPS / 控内存），**不是**防数据竞争；`MaxConcurrency=0`（不限并发）在正确性上是安全的。
  - 维护者修改 `Agent` / `ADKAgent` 结构体时，**禁止引入"构造后写、调用期读"的可变字段**，否则会破坏此不变量。该约束应作为代码评审检查点。
  - 升级 eino 大版本时，需重新核查上述源码点位是否仍成立（见「Eino 版本治理」）。

### DR-005：Eino 版本治理 —— 长期锁定 + 选择性升级

> `llm` 包的全部能力建立在 Eino 之上，且 Eino 仍在快速演进。版本策略是本包的头号长期风险。

- **当前锁定版本**：`github.com/cloudwego/eino v0.9.8`（见 `go.mod`）。
- **决策**：**长期锁定 + 选择性升级**，不自动跟随最新版。
- **升级触发条件**（满足任一）：
  - 需要 Eino 新增能力（如 ADK Interrupt/Resume，对应阶段 4）。
  - 上游修复了影响本包的缺陷。
  - 安全更新。
- **升级时必须重新核查的"上游契约点"**（这些是本包正确性所依赖的隐式假设）：
  - DR-004 中列出的并发复用源码点位。
  - Extraction 强制 tool_choice 的 `model.WithToolChoice` 行为（`adk_extraction.go` / `model_force.go`）。
  - `ChatModelAgentMiddleware` 的 hook 签名（`WrapModel` / `WrapInvokableToolCall` / `BeforeModelRewriteState`）。
  - `schema.Message.MultiContent` 多模态字段类型（O-002 依赖）。
- **回归保障**：升级后必须跑通 `go test ./llm -count=1`，并人工复核上述契约点。

### DR-006：不抽 "react + ADK 统一 Agent 接口"

- **决策**：**不**为 `*Agent`（react）与 `*ADKAgent` 抽公共 interface 让两者可互换。
- **背景**：曾考虑用一个 `Agent` interface（`Generate`/`Stream`/`Close`）统一三个工厂的返回类型。复核后否决。
- **理由**：
  - **该统一的已统一**：`NewADKAgent` 与 `NewDeepAgent` **已都返回同一个 `*ADKAgent`**（`deep_agent.go`）。ChatModelAgent 与 DeepAgent 的差异只在 eino 配置层（`deep.Config` 多了 write_todos/子Agent/FileSystem/Shell），go-kit 已在 `*ADKAgent` 层抹平。
  - **不该统一的是 react**：唯一未统一的是 `*Agent`(react) vs `*ADKAgent`，而 DR-001 的方向正是淘汰 react。为让 legacy 类型与主推类型"可互换"而抽接口 = 给正在废弃的东西投资，方向相反。
  - 成本不是问题（该接口零侵入 eino），但"低成本"≠"该做"；这里是"不该做"。
- **影响**：
  - 阶段 1 的 `AgentAsTool` 直接面向 `*ADKAgent`（或其 `.Agent()` 返回的 `adk.Agent`），**不**把 react 纳入。
  - 若未来 react 彻底移除，`*ADKAgent` 自然成为唯一 Agent 类型，届时无需接口抽象。

---

## 四、能力清单（Capability Inventory）

> 状态图例见末尾"图例"。完成新能力时，更新对应行。

### 4.1 模型层（Model）

| 能力 | 状态 | 入口 | 备注 |
|------|------|------|------|
| 多供应商模型路由 | ✅ | `NewModel` | 支持 12 种 `ModelProtocol` |
| OpenAI / OpenAI 兼容 | ✅ | `Protocol: OPENAI` / `OPENAI_COMPAT` | |
| Claude / Claude 兼容 | ✅ | `Protocol: CLAUDE` / `CLAUDE_COMPAT` | |
| Ark / ArkBot | ✅ | `Protocol: ARK` / `ARKBOT` | |
| DeepSeek | ✅ | `Protocol: DEEPSEEK` | 不支持 config 级 `ExtraFields` |
| Gemini | ✅ | `Protocol: GEMINI` | 使用 `genai.ThinkingConfig` |
| Ollama | ✅ | `Protocol: OLLAMA` | 本地模型 |
| Qianfan | ✅ | `Protocol: QIANFAN` | `Thinking` 被忽略（厂商不支持） |
| Qwen | ✅ | `Protocol: QWEN` | |
| Kimi（Moonshot） | ✅ | `Protocol: KIMI` | 复用 OpenAI 组件 |
| 自定义模型实例注入 | ✅ | `AgentModelConfig.Instance` | 跳过 `NewModel` 工厂 |
| 思考模式（Thinking）统一映射 | ✅ | `ModelConfig.Thinking` | Extraction 模式自动关闭 |
| 请求级模型参数调整 | 📋 | `Agent.Generate(..., opts...)` | 仅 `NewAgent` 支持，待对齐（O-004） |
| 多模态输入（Image/Audio） | ✅ | `llm.UserImageMessage(s)` / `llm.UserAudioMessage` | 基于 eino `UserInputMultiContent`（O-002）；Video/File 待需求 |

### 4.2 Agent 层

| 能力 | 状态 | 入口 | 备注 |
|------|------|------|------|
| `NewADKAgent`（主推路径） | ✅ | `NewADKAgent` | 基于 `adk.ChatModelAgent` |
| `NewAgent`（Legacy） | ⚠️ | `NewAgent` | 基于 `react.Agent`，不再演进 |
| `NewDeepAgent`（预置应用） | ✅ | `NewDeepAgent` | 复杂任务开箱即用 |
| Conversation 模式 | ✅ | `Execution.Mode: Conversation` | 纯对话，无工具 |
| Assistant 模式 | ✅ | `Execution.Mode: Assistant` | 工具可选 |
| Extraction 模式 | ✅ | `Execution.Mode: Extraction` | 强制 tool_choice + 失败修复重试 |
| 非流式生成 | ✅ | `Agent.Generate` / `ADKAgent.Generate` | |
| 流式生成 | ✅ | `Agent.Stream` / `ADKAgent.Stream` | 返回 `*schema.StreamReader` |
| `DirectReturnTools` | ✅ | `Execution.DirectReturnTools` | 指定工具结果直接返回 |
| 实例级并发控制 | ✅ | `Concurrency.MaxConcurrency` | channel 信号量，Stream 延迟释放 |
| 自定义 `PrepareMessages` | ✅ | `Prompt.PrepareMessages` | 每轮消息修改 |
| 自定义 `RewriteHistory` | ✅ | `Prompt.RewriteHistory` | 历史消息修改 |
| 自定义 `StreamToolCallChecker` | ✅ | `Streaming.ToolCallChecker` | 流式 tool call 检测 |
| `ExportGraph`（嵌入更大 compose 图） | 🟡 | `Agent.ExportGraph` | **仅 react 路径**；ADK 无等价能力，是 react→ADK 迁移的已知缺口（见 O-007，影响阶段 2 编排） |
| ADK Middleware 注册 | ✅ | `AgentConfig.Middlewares` | 用户 Middleware 先于包内注册（O-005） |
| ADK 运行时 Option | 📋 | `ADKAgent.Generate(ctx, msgs, opts...)` | 待实现（O-004） |
| Interrupt / Resume | 📋 | （远期） | 阶段 4 评估 |
| Checkpoint 持久化 | 📋 | （远期） | 阶段 4 评估 |

### 4.3 工具层（Tools）

| 能力 | 状态 | 入口 | 备注 |
|------|------|------|------|
| Eino 标准工具（`tool.BaseTool`） | ✅ | `ToolsConfig.Standard` | |
| 简化工具（`InvokableTool`） | ✅ | `ToolsConfig.Invokable` | 经 `ToolAdapter` 桥接 |
| 结构化输出工具 | ✅ | `NewStructTool[T]` | JSON Schema 校验 + 触发 Extraction 重试 |
| MCP 工具（stdio） | ✅ | `ToolsConfig.MCPServers` | `MCPConfig.Protocol: "stdio"` |
| MCP 工具（SSE） | ✅ | `ToolsConfig.MCPServers` | `MCPConfig.Protocol: "sse"` |
| 工具别名（Aliases） | 📋 | `ToolsConfig.Aliases` | 待实现（O-003） |
| 未知工具兜底（UnknownHandler） | 📋 | `ToolsConfig.UnknownHandler` | 待实现（O-003） |
| 参数修复（ArgumentsFixer） | 📋 | `ToolsConfig.ArgumentsFixer` | 待实现（O-003） |
| 工具错误转文本 Middleware | 📋 | （内置默认） | 待实现（O-003） |

### 4.4 可观测性（Observability）

| 能力 | 状态 | 入口 | 备注 |
|------|------|------|------|
| Eino Callback 注册 | ✅ | `Observability.Callbacks` | |
| Langfuse 集成 | ✅ | `NewLangfuseHandler` | 配置零校验，已知问题 |
| 结构化日志（LogClient） | ✅ | `Observability.StructuredLogs` | 与 `kit.Logger` 兼容 |
| 结构化日志：`agent.start/end` | ✅ | 自动 | |
| 结构化日志：`tool.start/end` | ✅ | 自动 | react 路径 |
| 结构化日志：`model.decision` | 🟡 | 自动 | **流式下不记录**，已知缺口（react/ADK 两路 Stream 均透传），待实现（O-008） |
| Token / Usage 用量聚合 | 📋 | （按 invocation 聚合 prompt/completion/reasoning tokens） | 待实现（O-009），生产成本核算刚需 |
| 按 Callback 节点精准注入 | 📋 | （远期） | 依赖编排能力 |

### 4.5 错误与防御

| 能力 | 状态 | 入口 | 备注 |
|------|------|------|------|
| `context.Context` 传播 | ✅ | 全部公开 API 首参 | |
| `fmt.Errorf("%w", err)` 错误包装 | ✅ | 内部统一风格 | |
| 导出 sentinel error | ✅ | `llm.ErrMissingModel` / `ErrUnsupportedProtocol` 等（`errors.go`） | 支持 `errors.Is`（O-001） |
| 模型调用重试（内置） | ✅ | Extraction 模式 `MaxRetries` | 仅 Extraction |
| 通用模型重试 Middleware | 📋 | （ADK 透传） | 阶段 1 评估 |
| 模型 Failover | 📋 | （ADK 透传） | 阶段 1 评估 |

---

## 五、演进路线（Roadmap）

> 阶段编号与优先级不严格绑定。每个阶段的能力**触发条件**见对应小节。

### 阶段 0：夯实 NewADKAgent 核心（**进行中**）

**目标**：让 `NewADKAgent` 能力 ≥ `NewAgent`，成为唯一推荐的 Agent 入口。

**范围**：

- ✅ O-001：导出 sentinel errors（P0）
- ✅ O-002：多模态输入 API（P0）
- 📋 O-003：工具层防御机制（P0）
- 📋 O-004：ADKAgent 运行时 Option（P1）
- ✅ O-005：暴露 ADK Middleware 注册入口（P1）
- 📋 O-006：doc.go / README 标记 `NewAgent` 为 Legacy（P1，依赖 O-007）
- ✅ O-007：NewADKAgent 能力对齐核查（**P0 门禁**，产出 [`CAPABILITY_DIFF.md`](./CAPABILITY_DIFF.md)：能力实质对齐，DR-001 成立）
- 📋 O-008：流式 `model.decision` 补记（P1）
- 📋 O-009：Token/Usage 用量聚合（P1）

**触发下一阶段的条件**：阶段 0 全部 ✅，且业务出现"需要多个 Agent 协作"的场景。

详见 [`OPTIMIZATION.md`](./OPTIMIZATION.md)。

---

### 阶段 1：多 Agent 协作（评估中）

**目标**：在不引入完整编排框架的前提下，支持多个 Agent 组合调用。

**候选方案**（按优先级）：

1. **AgentAsTool**：把 `*ADKAgent` 包装为 Tool，给另一个 `*ADKAgent` 使用
   - 示例：`llm.AgentAsTool(translatorAgent, "translate")`
   - 优点：复用现有 Tool 体系，零新概念
   - 缺点：嵌套层数受限（模型上下文窗口）
2. **直接暴露 ADK Runner**：让用户直接使用 `adk.Runner` 的事件流
   - 优点：最大灵活性
   - 缺点：暴露过多 Eino 内部细节

**不做的事**：自研多 Agent 编排 DSL。

**触发条件**：阶段 0 完成 + 业务出现至少 2 个明确的多 Agent 协作需求。

---

### 阶段 2：编排能力（Graph/Workflow）

**目标**：引入 `llm/compose` 子包，将 Eino 编排能力以 go-kit 风格暴露。

**候选能力**：

- `llm.NewChain[I, O](...)` —— 链式编排便利构造
- `llm.GraphAsTool(graph, name, desc)` —— 把 Graph 封装为 Tool（Eino 推荐模式）
- `llm.NewWorkflow(...)` —— 字段级映射编排

**触发条件**：

- 业务出现"确定性多步骤工作流"（与 Agent 自主决策相对）
- 或多个使用方重复实现同一种 Graph 模式

**重要约束**：不重写 Eino 的 `compose` 包，仅做配置层封装和便利函数。

---

### 阶段 3：知识增强（RAG）

**目标**：引入 `llm/rag` 子包，提供知识库问答能力。

**候选能力**：

- `rag.Embedder` —— 文本向量化（基于 eino-ext/embedding）
- `rag.Retriever` —— 相似度检索（基于 eino-ext/retriever）
- `rag.Indexer` —— 向量入库（基于 eino-ext/indexer）
- `rag.LoadDocuments(...)` —— 文档加载与切分

**与 Agent 的集成**：

- 通过 `PromptConfig.PrepareMessages` 在每轮注入检索结果
- 或通过 Tool 形式让 Agent 主动检索

**触发条件**：业务出现明确的"知识库问答"或"长文档检索"需求。

---

### 阶段 4：复杂任务与人机协作（远期）

**目标**：支持人工审批、长任务断点续跑、复杂规划。

**候选能力**：

- ADK Interrupt / Resume —— 人机协作中断恢复
- ADK Checkpoint —— 运行状态持久化
- ADK TurnLoop —— 多轮抢占
- 基于 `NewADKAgent` 自组装"轻量 DeepAgent"（任务规划 + 子 Agent 委派，但不要 DeepAgent 那套重工具）

**触发条件**：业务出现"人工 in-the-loop"或"长任务可恢复"需求。

---

## 六、状态图例

| 标记 | 含义 | 行动 |
|------|------|------|
| ✅ | 已实现，稳定可用 | — |
| 🟡 | 部分实现或有已知缺口 | 可用，但有限制；缺口应记录为优化项 |
| 📋 | 计划中，未开始 | 实现时需补测试 + 文档 |
| ⚠️ | 已废弃（Legacy） | 不再演进，仅维护向后兼容 |
| 🚫 | 明确不做 | 见决策记录 |
| 🔄 | 重新设计中 | 暂时不要扩展，等待设计稳定 |

---

## 七、维护规则

1. **完成一项能力** → 立即更新能力清单中对应行的状态（📋 → ✅）
2. **新增决策** → 在「决策记录」追加 DR-NNN 编号条目
3. **触发新阶段** → 在「演进路线」对应阶段填入"进行中"状态
4. **每季度** → 复核阶段触发条件，重新评估优先级
5. **重大版本变更** → 同步更新 `doc.go` 和 `README.md`

---

## 八、相关文档

- [`OPTIMIZATION.md`](./OPTIMIZATION.md) —— 阶段 0 详细优化计划
- [`README.md`](./README.md) —— 用户使用文档
- [`doc.go`](./doc.go) —— 包级 AI 提示
- [Eino 官方文档](https://www.cloudwego.io/zh/docs/eino/) —— 上游框架
