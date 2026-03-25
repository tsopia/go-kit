package llm

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type observedToolCallingModel struct {
	inner      model.ToolCallingChatModel
	logs       *structuredLogger
	mode       ExecutionMode
	toolChoice schema.ToolChoice
}

func newObservedToolCallingModel(inner model.ToolCallingChatModel, logs *structuredLogger, mode ExecutionMode, toolChoice schema.ToolChoice) model.ToolCallingChatModel {
	if logs == nil || !logs.enabled() || inner == nil {
		return inner
	}
	return &observedToolCallingModel{
		inner:      inner,
		logs:       logs,
		mode:       mode,
		toolChoice: toolChoice,
	}
}

func (m *observedToolCallingModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	msg, err := m.inner.Generate(ctx, input, opts...)
	if err != nil {
		m.logs.logError(ctx, "model.decision", "execution_mode", string(m.mode), "configured_tool_choice", string(m.toolChoice), "tool_call_count", 0, "tool_names", []string{}, "finish_reason", "", "error", err.Error())
		return nil, err
	}
	m.logDecision(ctx, msg)
	return msg, nil
}

func (m *observedToolCallingModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	sr, err := m.inner.Stream(ctx, input, opts...)
	if err != nil {
		m.logs.logError(ctx, "model.decision", "execution_mode", string(m.mode), "configured_tool_choice", string(m.toolChoice), "tool_call_count", 0, "tool_names", []string{}, "finish_reason", "", "error", err.Error())
		return nil, err
	}
	return sr, nil
}

func (m *observedToolCallingModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	inner, err := m.inner.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &observedToolCallingModel{
		inner:      inner,
		logs:       m.logs,
		mode:       m.mode,
		toolChoice: m.toolChoice,
	}, nil
}

func (m *observedToolCallingModel) logDecision(ctx context.Context, msg *schema.Message) {
	toolNames, toolCallCount := decisionToolNames(msg)
	finishReason, reasoningTokens := decisionMeta(msg)
	attrs := []any{
		"execution_mode", string(m.mode),
		"configured_tool_choice", string(m.toolChoice),
		"tool_call_count", toolCallCount,
		"tool_names", toolNames,
		"finish_reason", finishReason,
	}
	if reasoningTokens >= 0 {
		attrs = append(attrs, "reasoning_tokens", reasoningTokens)
	}
	m.logs.logInfo(ctx, "model.decision", attrs...)
}

func decisionToolNames(msg *schema.Message) ([]string, int) {
	if msg == nil || len(msg.ToolCalls) == 0 {
		return []string{}, 0
	}
	names := make([]string, 0, len(msg.ToolCalls))
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name != "" {
			names = append(names, tc.Function.Name)
		}
	}
	return names, len(msg.ToolCalls)
}

func decisionMeta(msg *schema.Message) (string, int) {
	if msg == nil || msg.ResponseMeta == nil {
		return "", -1
	}
	finishReason := msg.ResponseMeta.FinishReason
	reasoningTokens := -1
	if msg.ResponseMeta.Usage != nil {
		reasoningTokens = msg.ResponseMeta.Usage.CompletionTokensDetails.ReasoningTokens
	}
	return finishReason, reasoningTokens
}
