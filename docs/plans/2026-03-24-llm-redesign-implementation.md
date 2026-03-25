# LLM Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 重做 `llm` 包为面向业务场景的单 Agent 高层 API，保留当前能力效果，并收敛为按能力分组的配置与三种执行模式。

**Architecture:** 继续以 `react.Agent` 作为底层执行内核，在 `llm` 内部引入 `RuntimeSpec` 编译层与执行控制层，把新的 `AgentConfig` 分组配置编译成 `Conversation`、`Assistant`、`Extraction` 三种模式。工具失败修复、结构化提取、直返策略和流式 tool call 检测继续保留，但不再直接暴露为分散的实现细节字段。

**Tech Stack:** Go 1.24、CloudWeGo Eino `react.Agent`、table-driven tests、Langfuse callbacks、MCP、focused `go test ./llm`

---

## 前置检查

- [x] 已完整阅读相关代码文件
- [x] 已完整阅读相关测试文件
- [x] 已列出代码现状（能力、行为、约束已在设计文档中确认）

### Task 1: 先锁定现有业务效果的回归测试

**目标：** 在不依赖旧字段形态的前提下，先用测试锁住现有效果，避免重构时丢能力。

**Files:**
- Modify: `llm/agent_test.go`
- Modify: `llm/optimization_test.go`
- Modify: `llm/callbacks_test.go`
- Test: `llm/integration_test_extra_test.go`

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**Step 1: Write the failing tests**

补新的 table-driven tests，直接表达这些效果：

- `Conversation` 模式不应调用工具
- `Assistant` 模式允许模型自行决定是否调工具
- `Extraction` 模式必须先调工具
- 工具失败后模型能收到错误反馈并再次调工具
- 指定工具成功后可直接返回原始结果
- `StructTool` 仍可完成结构化提取闭环

```go
func TestExecutionModeContracts(t *testing.T) {
	tests := []struct {
		name string
		// TODO: 填入新 AgentConfig 和断言
	}{
		{name: "conversation_no_tools"},
		{name: "assistant_optional_tools"},
		{name: "extraction_forced_tool_repair_direct_return"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Fatal("TODO: implement new mode contract test")
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestExecutionModeContracts|TestOptimization_RealAgent|TestLogHandler_Integration|TestStructTool_FullScenario' -v`

Expected: FAIL，至少暴露新的模式 API 尚未落地。

**Step 3: Commit**

```bash
git add llm/agent_test.go llm/optimization_test.go llm/callbacks_test.go llm/integration_test_extra_test.go
git commit -m "test(llm): lock agent behavior contracts"
```

### Task 2: 引入新的 AgentConfig 分组结构

**目标：** 建立新的公开配置模型，先完成类型定义和零值语义。

**Files:**
- Create: `llm/config.go`
- Modify: `llm/agent.go`
- Test: `llm/config_test.go`

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**Step 1: Write the failing test**

新增测试，锁定：

- `AgentConfig` 具备 `Model` / `Prompt` / `Tools` / `Execution` / `Streaming` / `Observability`
- 零值配置可表达普通对话模式
- `Execution.Mode` 的默认值明确

```go
func TestAgentConfigDefaults(t *testing.T) {
	cfg := AgentConfig{}
	if cfg.Execution.Mode != "" {
		t.Fatalf("expected zero value mode before normalization, got %q", cfg.Execution.Mode)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestAgentConfigDefaults' -v`

Expected: FAIL，当前还不存在新的分组配置。

**Step 3: Write minimal implementation**

实现最小公开类型：

- `AgentConfig`
- `ModelConfig` 沿用现有模型路由配置
- `PromptConfig`
- `ToolsConfig`
- `ExecutionConfig`
- `StreamingConfig`
- `ObservabilityConfig`
- `ExecutionMode`

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestAgentConfigDefaults' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add llm/config.go llm/agent.go llm/config_test.go
git commit -m "feat(llm): introduce grouped agent config"
```

### Task 3: 引入 RuntimeSpec 编译层

**目标：** 将公开配置编译成统一内部运行规格，避免 `NewAgent` 继续堆条件分支。

**Files:**
- Create: `llm/runtime_spec.go`
- Test: `llm/runtime_spec_test.go`

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**Step 1: Write the failing test**

锁定以下编译行为：

- `Conversation` 编译后禁用工具
- `Assistant` 编译后允许工具
- `Extraction` 编译后启用强制工具、修复策略和返回策略

```go
func TestCompileRuntimeSpec(t *testing.T) {
	tests := []struct {
		name string
		cfg  AgentConfig
	}{
		{name: "conversation"},
		{name: "assistant"},
		{name: "extraction"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := compileRuntimeSpec(tt.cfg)
			if err != nil {
				t.Fatalf("compileRuntimeSpec failed: %v", err)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestCompileRuntimeSpec' -v`

Expected: FAIL，因为 `RuntimeSpec` 尚不存在。

**Step 3: Write minimal implementation**

实现：

- `RuntimeSpec`
- `compileRuntimeSpec(cfg AgentConfig) (RuntimeSpec, error)`
- 默认值归一化
- 对 `Prompt` / `Execution` / `Streaming` / `Observability` 的基础编译

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestCompileRuntimeSpec' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add llm/runtime_spec.go llm/runtime_spec_test.go
git commit -m "refactor(llm): compile agent config into runtime spec"
```

### Task 4: 重建 Prompt / Tools 装配层

**目标：** 把 `SystemPrompt`、每轮消息改写、历史改写、工具聚合和 MCP 来源收敛为构建器逻辑。

**Files:**
- Create: `llm/runtime_builder.go`
- Modify: `llm/mcp.go`
- Modify: `llm/tool_adapter.go`
- Test: `llm/runtime_builder_test.go`

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**Step 1: Write the failing test**

锁定：

- `Prompt.System` 会映射为 system message 注入
- `Prompt.PrepareMessages` 和 `Prompt.RewriteHistory` 都能生效
- `Tools` 层能同时聚合标准工具、简化工具和 MCP 工具

```go
func TestBuildPromptAndTools(t *testing.T) {
	t.Fatal("TODO: verify prompt and tool assembly from runtime spec")
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestBuildPromptAndTools' -v`

Expected: FAIL

**Step 3: Write minimal implementation**

实现：

- prompt builder
- tool aggregation builder
- MCP source 接入点继续返回 `[]tool.BaseTool`

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestBuildPromptAndTools' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add llm/runtime_builder.go llm/mcp.go llm/tool_adapter.go llm/runtime_builder_test.go
git commit -m "refactor(llm): assemble prompt and tool sources from runtime spec"
```

### Task 5: 落地 Conversation / Assistant 模式

**目标：** 先完成不涉及强制工具修复状态机的两种模式，确保基础路径稳定。

**Files:**
- Modify: `llm/agent.go`
- Test: `llm/agent_test.go`

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**Step 1: Write the failing test**

补专门测试：

- `Conversation` 不绑定工具
- `Assistant` 正常走 `react.Agent`
- `Assistant` 场景下指定工具可 direct return

```go
func TestNewAgent_ConversationAndAssistantModes(t *testing.T) {
	t.Fatal("TODO: cover conversation and assistant modes")
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestNewAgent_ConversationAndAssistantModes' -v`

Expected: FAIL

**Step 3: Write minimal implementation**

在 `NewAgent` 内：

- 使用 `RuntimeSpec`
- 针对 `Conversation` / `Assistant` 组装 `react.AgentConfig`
- 保留现有 callbacks 注入

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestNewAgent_ConversationAndAssistantModes' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add llm/agent.go llm/agent_test.go
git commit -m "feat(llm): support conversation and assistant execution modes"
```

### Task 6: 落地 Extraction 模式的强制工具修复链路

**目标：** 以新的执行模式承载当前最关键的“强制工具调用 -> 失败修复 -> 成功”能力。

**Files:**
- Modify: `llm/agent.go`
- Modify: `llm/model_force.go`
- Test: `llm/agent_test.go`

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**Step 1: Write the failing test**

保留并迁移当前两个核心效果测试：

- 工具失败后错误被反馈给模型，模型再次调用工具
- 成功后不再继续强制工具调用

```go
func TestExtractionMode_RepairLoop(t *testing.T) {
	t.Fatal("TODO: verify extraction mode tool repair loop")
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestExtractionMode_RepairLoop|TestNewAgent_ToolChoiceForced_WithRetry' -v`

Expected: FAIL

**Step 3: Write minimal implementation**

将当前逻辑重构为 Extraction 控制器：

- 维护工具调用状态
- 工具失败时转成模型可消费的反馈
- 成功后退出强制模式

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestExtractionMode_RepairLoop|TestNewAgent_ToolChoiceForced_WithRetry' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add llm/agent.go llm/model_force.go llm/agent_test.go
git commit -m "feat(llm): implement extraction mode repair loop"
```

### Task 7: 落地直返策略与结构化提取场景

**目标：** 把 direct return 和 `StructTool` 提升为新的高层能力闭环。

**Files:**
- Modify: `llm/agent.go`
- Modify: `llm/struct_tool.go`
- Modify: `llm/optimization_test.go`
- Modify: `llm/integration_test_extra_test.go`

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**Step 1: Write the failing test**

锁定：

- 指定工具成功后只调用一次模型
- `StructTool` 在 `Extraction` 模式下默认形成“强制工具 -> 失败修复 -> 成功直返”闭环

```go
func TestExtractionMode_DirectReturnAndStructTool(t *testing.T) {
	t.Fatal("TODO: verify direct return and struct extraction path")
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestExtractionMode_DirectReturnAndStructTool|TestOptimization_RealAgent|TestStructTool_FullScenario' -v`

Expected: FAIL

**Step 3: Write minimal implementation**

实现：

- 新的 response policy 编排
- `StructTool` 的高层使用路径文档化
- 保留工具成功后尽量避免多一次总结模型调用

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestExtractionMode_DirectReturnAndStructTool|TestOptimization_RealAgent|TestStructTool_FullScenario' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add llm/agent.go llm/struct_tool.go llm/optimization_test.go llm/integration_test_extra_test.go
git commit -m "feat(llm): add response policy for extraction results"
```

### Task 8: 收敛 Streaming 与 Observability

**目标：** 保留流式 tool call 检测和 callback / Langfuse / slog 接入能力。

**Files:**
- Modify: `llm/agent.go`
- Modify: `llm/callbacks.go`
- Modify: `llm/callbacks_test.go`

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**Step 1: Write the failing test**

补测试锁定：

- `Streaming.ToolCallDetector` 可透传到底层
- `Observability.Handlers` 仍能注入 callbacks
- 日志 handler 仍能记录组件开始与结束

```go
func TestStreamingAndObservabilityConfig(t *testing.T) {
	t.Fatal("TODO: verify streaming detector and callbacks wiring")
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestStreamingAndObservabilityConfig|TestLogHandler_Integration' -v`

Expected: FAIL

**Step 3: Write minimal implementation**

实现：

- `Streaming` 分组配置到 `react.AgentConfig`
- `Observability` 分组配置到 compose callbacks
- 保留 `NewLangfuseHandler` / `NewLogHandler` 作为 helper

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestStreamingAndObservabilityConfig|TestLogHandler_Integration' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add llm/agent.go llm/callbacks.go llm/callbacks_test.go
git commit -m "refactor(llm): wire streaming and observability config"
```

### Task 9: 升级第一批依赖并重建模型工厂

**目标：** 升级建议中的第一批依赖，并确认模型工厂与新 API 正常协作。

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Modify: `llm/llm.go`
- Modify: `llm/factory_test.go`

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**Step 1: Write the failing test**

补测试锁定：

- 模型工厂仍支持现有协议
- OpenAI / Qwen / Gemini 升级后工厂行为不变

```go
func TestNewModelFactory_AfterUpgrade(t *testing.T) {
	t.Fatal("TODO: verify factory behavior after dependency upgrade")
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestNewModelFactory_AfterUpgrade|TestNewModelValidation' -v`

Expected: FAIL

**Step 3: Write minimal implementation**

执行：

- `go get github.com/cloudwego/eino@v0.8.4`
- `go get github.com/cloudwego/eino-ext/components/model/openai@v0.1.10`
- `go get github.com/cloudwego/eino-ext/components/model/qwen@v0.1.6`
- `go get github.com/cloudwego/eino-ext/components/model/gemini@v0.1.29`
- `go mod tidy`

并根据需要微调 `llm/llm.go`。

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./llm -run 'TestNewModelFactory_AfterUpgrade|TestNewModelValidation' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add go.mod go.sum llm/llm.go llm/factory_test.go
git commit -m "build(llm): upgrade eino core model adapters"
```

### Task 10: 文档收尾与包级验证

**目标：** 让 README 和设计一致，并完成 focused verification。

**Files:**
- Modify: `llm/README.md`
- Modify: `docs/plans/2026-03-24-llm-redesign-design.md`
- Verify only

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**Step 1: Update docs**

更新 README：

- 新 `AgentConfig` 能力分组
- `Conversation / Assistant / Extraction` 三种模式
- 结构化提取推荐路径
- Streaming / Observability 用法

**Step 2: Run focused package verification**

Run: `GOCACHE=/tmp/go-build go test ./llm -v`

Expected: PASS

**Step 3: Run lint**

Run: `GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache GOCACHE=/tmp/go-build golangci-lint run ./llm/...`

Expected: `0 issues.`

**Step 4: Optional full repository verification**

Run: `GOCACHE=/tmp/go-build go test ./...`

Expected: 可能因仓库已知 `sonic` 基线问题失败；若失败，需明确说明失败与本次改动无关。

**Step 5: Commit**

```bash
git add llm/README.md docs/plans
git commit -m "docs(llm): document redesigned single-agent API"
```
