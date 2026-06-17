# NewAgent（react） vs NewADKAgent（adk） 能力对照表

> 本文档是 [`OPTIMIZATION.md`](./OPTIMIZATION.md) 中 **O-007** 的产出。
> 用途：在 DR-001（主推 `NewADKAgent`、`NewAgent` 标 go-kit legacy）落地前，
> 核查 `NewADKAgent` 是否已覆盖 `NewAgent` 的全部能力，识别并处置缺口。
>
> 结论：**能力已对齐（运行时 Option 由 O-004 补齐完成）；ExportGraph 为接受缺口。**
> 因此 DR-001 / O-006（标 Legacy）可以执行。

---

## 一、公开方法对照

| 方法 | NewAgent（react，`agent.go`） | NewADKAgent（adk，`adk_agent.go`） | 状态 |
|------|------------------------------|-----------------------------------|------|
| `Generate(ctx, msgs, ...)` | ✅ 含 `opts ...agent.AgentOption` | ✅ 含 `...GenerateOption`（参数调整） | ✅（O-004 已完成） |
| `Stream(ctx, msgs, ...)` | ✅ 含 `opts ...agent.AgentOption` | ✅ 含 `...GenerateOption`（参数调整） | ✅（O-004 已完成） |
| `Close() error` | ✅ | ✅ | ✅ |
| `ExportGraph()` | ✅ | ❌（adk 无等价能力） | 🚫 接受缺口（见三） |
| `Agent() adk.Agent` | ❌ | ✅（暴露底层、用于子 Agent / DeepAgent） | ADK 独有，非缺口 |

---

## 二、共享配置能力（经 `AgentConfig` + `compileRuntimeSpec`）

两条路径共用同一份 `AgentConfig` 与 `compileRuntimeSpec`，下列能力**天然对齐**：

| 能力 | react | adk | 备注 |
|------|-------|-----|------|
| 三种执行模式（Conversation/Assistant/Extraction） | ✅ | ✅ | 共享 `compileRuntimeSpec` |
| 工具：Standard / Invokable / MCP | ✅ | ✅ | 共享 `buildPromptAndTools` |
| Extraction 强制 toolcall + 修复重试 | ✅（`model_force.go`） | ✅（`adk_extraction.go`，共享 `extractionState`） | 状态机共享 |
| `DirectReturnTools` | ✅（context.Cancel 优化） | ✅（ADK 原生 `ReturnDirectly`） | 实现不同、语义一致 |
| Extraction 自动关闭 Thinking | ✅ | ✅ | 仅 `Instance==nil` 时生效（共同已知限制） |
| 实例级并发控制（`MaxConcurrency`） | ✅ | ✅ | 共享 `concurrencyGuard`；并发复用安全见 DR-004 |
| `PrepareMessages` / `RewriteHistory` | ✅（MessageModifier/Rewriter） | ✅（`BeforeModelRewriteState`） | adk 下 system prompt 由 Instruction 单管 |
| `StreamToolCallChecker` | ✅ | ⚠️ 见缺口说明 | adk 内部自管流式 toolcall 检测 |
| 可观测性：Callbacks | ✅ | ✅ | |
| 可观测性：结构化日志 agent/tool | ✅ | ✅ | |
| 可观测性：`model.decision`（非流式） | ✅ | ✅ | 流式两路均缺，→ O-008 |
| 工具 Middleware 扩展 | ✅（`compose.ToolMiddleware`，内置注入） | ✅（`ChatModelAgentMiddleware`）→ 用户入口 O-005 | |

---

## 三、缺口处置决定

### 缺口 1：运行时 Option（`Generate`/`Stream` 的 `opts`）

- **现状**：react 支持 `agent.WithChatModelOptions` 等运行时注入；ADK 的 `Generate`/`Stream` 暂无 `opts`。
- **处置**：**补齐** → O-004。已核查 ADK 侧存在 `adk.WithChatModelOptions([]model.Option)`，可透传"运行时参数调整"；"运行时换模型"明确不做（与预建 runner 架构冲突）。

### 缺口 2：`ExportGraph()`

- **现状**：react 可导出为 `compose.AnyGraph` 嵌入更大编排图；ADK 无等价能力。
- **处置**：**接受缺口**。包维护者确认现网无消费方依赖 `ExportGraph()`。将来"把 Agent 嵌入编排图"的需求由阶段 2 的 `AgentAsTool` / `GraphAsTool`（Eino 官方推荐方式）覆盖。

### 缺口 3：`StreamToolCallChecker`（轻微）

- **现状**：react 暴露自定义流式 toolcall 检测；ADK 内部自管该逻辑，未透出同名钩子。
- **处置**：**文档化为已知差异**。ADK 的内部检测已覆盖常见场景；如出现具体业务需求再评估透出（暂不立项）。

---

## 四、结论

- 阻断性缺口仅"运行时 Option"，已由 O-004 规划补齐。
- `ExportGraph` / `StreamToolCallChecker` 为可接受差异。
- **DR-001 成立**：`NewADKAgent` 能力实质 ≥ `NewAgent`，可执行 O-006（标 go-kit legacy）。

---

## 相关文档

- [`ROADMAP.md`](./ROADMAP.md) —— DR-001（主推 ADK）、DR-006（不抽统一接口）
- [`OPTIMIZATION.md`](./OPTIMIZATION.md) —— O-004 / O-006 / O-008
</content>
</invoke>
