package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// ADKAgent 基于 eino adk.ChatModelAgent 的 Agent 实现（v0.9 ADK 后端）。
//
// 与现有 Agent（react-based）并存：
//   - NewAgent    → react.Agent（v0.8 路径，保留向后兼容）
//   - NewADKAgent → adk.ChatModelAgent（v0.9 路径，享受 ADK 新能力）
//
// 两者共享 AgentConfig / compileRuntimeSpec / NewModel / extractionState，
// 只在执行引擎层分叉。
type ADKAgent struct {
	runner       *adk.Runner // 非流式 runner（EnableStreaming=false）
	streamRunner *adk.Runner // 流式 runner（EnableStreaming=true）
	adkAgent     adk.Agent
	guard        *concurrencyGuard
	cleanup      func() error
	logs         *structuredLogger
	cfg          AgentConfig
	mode         ExecutionMode
	toolCount    int
}

// NewADKAgent 创建基于 adk.ChatModelAgent 的 Agent。
//
// 配置复用 AgentConfig，执行引擎为 ADK（v0.9）。行为语义与 NewAgent 一致：
//   - Conversation: 纯对话
//   - Assistant: 工具可选
//   - Extraction: 强制工具调用 + 失败修复重试（通过 ChatModelAgentMiddleware 实现）
//
// 并发控制通过 Concurrency.MaxConcurrency 配置，与 Agent 相同。
func NewADKAgent(ctx context.Context, cfg AgentConfig) (*ADKAgent, error) {
	spec, err := compileRuntimeSpec(cfg)
	if err != nil {
		return nil, fmt.Errorf("compile runtime spec: %w", err)
	}

	// Extraction 模式关闭 thinking（复用 agent.go 同款逻辑）。
	// 思考模式是模型创建时的配置，与用 react 还是 adk 无关，
	// 只要 NewModel 前把 Thinking.Enable 置 false 即可。
	if spec.Execution.ToolChoice == schema.ToolChoiceForced {
		if spec.Model.Instance == nil {
			if spec.Model.Config.Thinking == nil {
				spec.Model.Config.Thinking = &ThinkingConfig{Enable: false}
			} else if spec.Model.Config.Thinking.Enable {
				cp := *spec.Model.Config.Thinking
				cp.Enable = false
				spec.Model.Config.Thinking = &cp
			}
		}
	}

	built, err := buildPromptAndTools(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("build prompt and tools: %w", err)
	}

	// 创建模型（BaseChatModel 即可，工具绑定由 ADK 框架处理）
	var chatModel model.BaseChatModel
	if spec.Model.Instance != nil {
		chatModel = spec.Model.Instance
	} else {
		m, err := NewModel(ctx, spec.Model.Config)
		if err != nil {
			_ = built.Cleanup()
			return nil, fmt.Errorf("create model: %w", err)
		}
		chatModel = m
	}

	// 构建 ToolsConfig
	toolsConfig := adk.ToolsConfig{
		ToolsNodeConfig: compose.ToolsNodeConfig{Tools: built.Tools},
	}
	// Extraction 的 DirectReturn：ADK 原生 ReturnDirectly（替代 react 的 context.Cancel）
	if len(spec.Execution.DirectReturnTools) > 0 {
		toolsConfig.ReturnDirectly = make(map[string]bool, len(spec.Execution.DirectReturnTools))
		for name := range spec.Execution.DirectReturnTools {
			toolsConfig.ReturnDirectly[name] = true
		}
	}

	agentCfg := &adk.ChatModelAgentConfig{
		Model:       chatModel,
		Instruction: spec.Prompt.System,
		ToolsConfig: toolsConfig,
	}
	if spec.Execution.MaxStep > 0 {
		agentCfg.MaxIterations = spec.Execution.MaxStep
	}
	// Extraction 模式：注入强制 toolcall + 修复重试 middleware（最外层）
	if spec.Execution.ToolChoice == schema.ToolChoiceForced {
		agentCfg.Handlers = append(agentCfg.Handlers, newADKExtractionMiddleware(spec.Execution.RepairMaxAttempts))
	}

	// 可观测性 + MessageModifier 适配（内层，记录真实工具调用结果）
	structuredLogs := newStructuredLogger(spec.Observability.StructuredLogs)
	if structuredLogs.enabled() || spec.Prompt.PrepareMessages != nil || spec.Prompt.RewriteHistory != nil {
		agentCfg.Handlers = append(agentCfg.Handlers, newADKObservabilityMiddleware(
			structuredLogs, spec.Execution.Mode, spec.Execution.ToolChoice,
			spec.Execution.DirectReturnTools, spec.Prompt.PrepareMessages, spec.Prompt.RewriteHistory,
		))
	}

	adkAgent, err := adk.NewChatModelAgent(ctx, agentCfg)
	if err != nil {
		_ = built.Cleanup()
		return nil, fmt.Errorf("create adk chat model agent: %w", err)
	}

	// 两个 runner 共享同一个 agent：非流式用于 Generate，流式用于 Stream
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: adkAgent})
	streamRunner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: adkAgent, EnableStreaming: true})

	return &ADKAgent{
		runner:       runner,
		streamRunner: streamRunner,
		adkAgent:     adkAgent,
		guard:        newConcurrencyGuard(spec.Concurrency.MaxConcurrency),
		cleanup:      built.Cleanup,
		logs:         structuredLogs,
		cfg:          cfg,
		mode:         spec.Execution.Mode,
		toolCount:    len(built.Tools),
	}, nil
}

// Generate 非流式调用 Agent。
// 内部消费 ADK 事件流，取最终 assistant 消息。
func (a *ADKAgent) Generate(ctx context.Context, messages []*schema.Message) (msg *schema.Message, err error) {
	if err := a.guard.acquire(ctx); err != nil {
		return nil, err
	}
	defer a.guard.release()
	ctx = withInvocationID(ctx)

	if a.logs != nil && a.logs.enabled() {
		started := time.Now()
		a.logs.logInfo(ctx, "agent.start",
			"execution_mode", string(a.mode),
			"tool_count", a.toolCount,
			"message_count", len(messages),
		)
		defer func() {
			attrs := []any{
				"execution_mode", string(a.mode),
				"tool_count", a.toolCount,
				"latency_ms", time.Since(started).Milliseconds(),
			}
			if err != nil {
				attrs = append(attrs, "status", "error", "error", err.Error())
			} else {
				attrs = append(attrs, "status", "success")
			}
			a.logs.logInfo(ctx, "agent.end", attrs...)
		}()
	}

	return runADK(a.runner, ctx, messages, a.cfg.Observability.Callbacks)
}

// Stream 流式调用 Agent。
// 返回最终模型输出的流式 StreamReader；并发名额在流消费结束时释放。
func (a *ADKAgent) Stream(ctx context.Context, messages []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
	if err := a.guard.acquire(ctx); err != nil {
		return nil, err
	}
	// panic 安全：streamADK panic 或提前返回时释放 guard
	released := false
	defer func() {
		if !released {
			a.guard.release()
		}
	}()

	ctx = withInvocationID(ctx)

	var onEnd func()
	if a.logs != nil && a.logs.enabled() {
		started := time.Now()
		a.logs.logInfo(ctx, "agent.start",
			"execution_mode", string(a.mode),
			"tool_count", a.toolCount,
			"streaming", true,
			"message_count", len(messages),
		)
		onEnd = func() {
			a.logs.logInfo(ctx, "agent.end",
				"execution_mode", string(a.mode),
				"tool_count", a.toolCount,
				"latency_ms", time.Since(started).Milliseconds(),
				"status", "stream_completed",
			)
		}
	}

	sr, err := streamADK(a.streamRunner, ctx, messages, a.cfg.Observability.Callbacks)
	if err != nil {
		return nil, err
	}

	// 成功：release + agent.end 延迟到流消费结束
	released = true
	return wrapStreamWithGuard(sr, a.guard, onEnd), nil
}

// Close 释放 Agent 持有的资源（如 MCP 工具连接）。
func (a *ADKAgent) Close() error {
	if a.cleanup == nil {
		return nil
	}
	return a.cleanup()
}

// Agent 返回底层 adk.Agent，用于将此 Agent 作为子 Agent
// 传给 NewDeepAgent 等需要 adk.Agent 的场景。
func (a *ADKAgent) Agent() adk.Agent {
	return a.adkAgent
}
