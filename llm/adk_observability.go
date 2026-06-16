package llm

import (
	"context"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// adkObservabilityMiddleware 为 ADKAgent 提供可观测性适配，对应 react 路径的
// model_observer + tool_observer + MessageModifier，统一用 ChatModelAgentMiddleware hook 实现：
//
//   - WrapModel: 记录 model.decision（tool_calls、finish_reason、reasoning_tokens）
//   - WrapInvokableToolCall: 记录 tool.start / tool.success / tool.error
//   - BeforeModelRewriteState: 应用 PromptConfig.PrepareMessages / RewriteHistory 改写消息
type adkObservabilityMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	logs              *structuredLogger
	mode              ExecutionMode
	toolChoice        schema.ToolChoice
	directReturnTools map[string]struct{}
	prepareMessages   MessageModifier
	rewriteHistory    MessageModifier
}

func newADKObservabilityMiddleware(
	logs *structuredLogger,
	mode ExecutionMode,
	toolChoice schema.ToolChoice,
	directReturnTools map[string]struct{},
	prepareMessages MessageModifier,
	rewriteHistory MessageModifier,
) *adkObservabilityMiddleware {
	return &adkObservabilityMiddleware{
		logs:              logs,
		mode:              mode,
		toolChoice:        toolChoice,
		directReturnTools: directReturnTools,
		prepareMessages:   prepareMessages,
		rewriteHistory:    rewriteHistory,
	}
}

func (m *adkObservabilityMiddleware) logsEnabled() bool {
	return m != nil && m.logs != nil && m.logs.enabled()
}

// ── WrapModel: 记录 model.decision（对应 model_observer.go） ──────────

func (m *adkObservabilityMiddleware) WrapModel(_ context.Context, mdl model.BaseChatModel, _ *adk.ModelContext) (model.BaseChatModel, error) {
	if !m.logsEnabled() {
		return mdl, nil
	}
	return &adkObservedModel{inner: mdl, logs: m.logs, mode: m.mode, toolChoice: m.toolChoice}, nil
}

type adkObservedModel struct {
	inner      model.BaseChatModel
	logs       *structuredLogger
	mode       ExecutionMode
	toolChoice schema.ToolChoice
}

func (o *adkObservedModel) Generate(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	msg, err := o.inner.Generate(ctx, msgs, opts...)
	if err != nil {
		o.logs.logError(ctx, "model.decision",
			"execution_mode", string(o.mode),
			"configured_tool_choice", string(o.toolChoice),
			"tool_call_count", 0,
			"tool_names", []string{},
			"finish_reason", "",
			"error", err.Error(),
		)
		return nil, err
	}
	o.logDecision(ctx, msg)
	return msg, nil
}

func (o *adkObservedModel) Stream(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	// Stream 模式下无法在中途拿到完整决策，直接透传。
	return o.inner.Stream(ctx, msgs, opts...)
}

func (o *adkObservedModel) logDecision(ctx context.Context, msg *schema.Message) {
	toolNames, toolCallCount := decisionToolNames(msg)
	finishReason, reasoningTokens := decisionMeta(msg)
	attrs := []any{
		"execution_mode", string(o.mode),
		"configured_tool_choice", string(o.toolChoice),
		"tool_call_count", toolCallCount,
		"tool_names", toolNames,
		"finish_reason", finishReason,
	}
	if reasoningTokens >= 0 {
		attrs = append(attrs, "reasoning_tokens", reasoningTokens)
	}
	o.logs.logInfo(ctx, "model.decision", attrs...)
}

// ── BeforeModelRewriteState: 应用 PrepareMessages / RewriteHistory ────
//
// 对应 react 路径的 MessageModifier / MessageRewriter。
// ADK 路径下 system prompt 由 Instruction 单独管理，
// PrepareMessages 只作用于对话消息（state.Messages）。

func (m *adkObservabilityMiddleware) BeforeModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, _ *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	if m.rewriteHistory != nil {
		state.Messages = m.rewriteHistory(ctx, state.Messages)
	}
	if m.prepareMessages != nil {
		state.Messages = m.prepareMessages(ctx, state.Messages)
	}
	return ctx, state, nil
}

// ── WrapInvokableToolCall: 记录工具调用生命周期（对应 tool_observer.go） ──

func (m *adkObservabilityMiddleware) WrapInvokableToolCall(_ context.Context, ep adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	if !m.logsEnabled() {
		return ep, nil
	}
	logs := m.logs
	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		startAttrs := []any{
			"tool_name", tCtx.Name,
			"tool_call_id", tCtx.CallID,
		}
		if logs.cfg.LogToolArguments {
			startAttrs = append(startAttrs, "arguments", truncateField(args, logs.cfg.MaxFieldLength))
		}
		logs.logInfo(ctx, "tool.start", startAttrs...)

		started := time.Now()
		result, err := ep(ctx, args, opts...)
		latencyMS := time.Since(started).Milliseconds()

		if err != nil {
			logs.logError(ctx, "tool.error",
				"tool_name", tCtx.Name,
				"tool_call_id", tCtx.CallID,
				"latency_ms", latencyMS,
				"error", err.Error(),
			)
			return "", err
		}

		successAttrs := []any{
			"tool_name", tCtx.Name,
			"tool_call_id", tCtx.CallID,
			"latency_ms", latencyMS,
		}
		if _, ok := m.directReturnTools[tCtx.Name]; ok {
			successAttrs = append(successAttrs, "direct_return", true)
		}
		if logs.cfg.LogToolResults {
			successAttrs = append(successAttrs, "result", truncateField(result, logs.cfg.MaxFieldLength))
		}
		logs.logInfo(ctx, "tool.success", successAttrs...)
		return result, nil
	}, nil
}
