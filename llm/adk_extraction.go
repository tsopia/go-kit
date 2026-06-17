package llm

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// adkExtractionMiddleware 实现 adk.ChatModelAgentMiddleware，
// 为 ADKAgent 提供 Extraction 模式的「强制工具调用 + 失败修复重试」。
//
// 对应 react 路径的 extractionToolChoiceModel + extractionRetryMiddleware，
// 状态机 extractionState 被两者共享（见 extraction_state.go）。
type adkExtractionMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	state *extractionState
}

func newADKExtractionMiddleware(maxRetries int) *adkExtractionMiddleware {
	return &adkExtractionMiddleware{
		state: newExtractionState(maxRetries),
	}
}

// WrapModel 包装 model，在工具调用成功前强制 ToolChoiceForced。
// 对应 react 路径的 extractionToolChoiceModel（model_force.go:103）。
func (m *adkExtractionMiddleware) WrapModel(_ context.Context, mdl model.BaseChatModel, _ *adk.ModelContext) (model.BaseChatModel, error) {
	return &forcedToolChoiceModel{inner: mdl, state: m.state}, nil
}

// forcedToolChoiceModel 包装 BaseChatModel（注意：adk 的 WrapModel 给的是
// BaseChatModel 而非 ToolCallingChatModel，工具绑定由框架单独处理）。
type forcedToolChoiceModel struct {
	inner model.BaseChatModel
	state *extractionState
}

func (f *forcedToolChoiceModel) Generate(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	if f.state.shouldForceToolCall() {
		opts = append(opts, model.WithToolChoice(schema.ToolChoiceForced))
	}
	return f.inner.Generate(ctx, msgs, opts...)
}

func (f *forcedToolChoiceModel) Stream(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	if f.state.shouldForceToolCall() {
		opts = append(opts, model.WithToolChoice(schema.ToolChoiceForced))
	}
	return f.inner.Stream(ctx, msgs, opts...)
}

// WrapInvokableToolCall 包装工具执行，失败时转为修复 prompt 让模型重试。
// 对应 react 路径的 extractionRetryMiddleware（model_force.go:136）。
func (m *adkExtractionMiddleware) WrapInvokableToolCall(_ context.Context, ep adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		result, err := ep(ctx, args, opts...)
		if err == nil {
			m.state.recordSuccess(tCtx.Name, result)
			return result, nil
		}
		m.state.recordFailure()
		if m.state.retriesExhausted() {
			return "", fmt.Errorf("%w: tool %s (%d attempts): %w", ErrExtractionRetriesExhausted, tCtx.Name, m.state.maxRetries, err)
		}
		return fmt.Sprintf("工具执行失败: %v\n请分析错误原因，调整参数后重新调用工具。", err), nil
	}, nil
}
