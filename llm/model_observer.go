package llm

import (
	"context"
	"io"

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
		m.logs.logError("model.decision", "execution_mode", string(m.mode), "tool_choice", string(m.toolChoice), "tool_call_count", 0, "tool_names", []string{}, "finish_reason", "", "error", err.Error())
		return nil, err
	}
	m.logDecision(msg)
	return msg, nil
}

func (m *observedToolCallingModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	sr, err := m.inner.Stream(ctx, input, opts...)
	if err != nil {
		m.logs.logError("model.decision", "execution_mode", string(m.mode), "tool_choice", string(m.toolChoice), "tool_call_count", 0, "tool_names", []string{}, "finish_reason", "", "error", err.Error())
		return nil, err
	}

	copies := sr.Copy(2)
	observer := copies[1]
	sr = copies[0]

	go m.observeStream(observer)
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

func (m *observedToolCallingModel) logDecision(msg *schema.Message) {
	toolNames, toolCallCount := decisionToolNames(msg)
	finishReason, reasoningTokens := decisionMeta(msg)
	attrs := []any{
		"execution_mode", string(m.mode),
		"tool_choice", string(m.toolChoice),
		"tool_call_count", toolCallCount,
		"tool_names", toolNames,
		"finish_reason", finishReason,
	}
	if reasoningTokens >= 0 {
		attrs = append(attrs, "reasoning_tokens", reasoningTokens)
	}
	m.logs.logInfo("model.decision", attrs...)
}

func (m *observedToolCallingModel) observeStream(sr *schema.StreamReader[*schema.Message]) {
	defer sr.Close()

	var chunks []*schema.Message
	for {
		msg, err := sr.Recv()
		if err == io.EOF {
			merged, mergeErr := schema.ConcatMessages(chunks)
			if mergeErr != nil {
				m.logs.logError("model.decision", "execution_mode", string(m.mode), "tool_choice", string(m.toolChoice), "tool_call_count", len(chunks), "tool_names", []string{}, "finish_reason", "", "error", mergeErr.Error())
				return
			}
			m.logDecision(merged)
			return
		}
		if err != nil {
			m.logs.logError("model.decision", "execution_mode", string(m.mode), "tool_choice", string(m.toolChoice), "tool_call_count", len(chunks), "tool_names", []string{}, "finish_reason", "", "error", err.Error())
			return
		}
		chunks = append(chunks, msg)
	}
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
