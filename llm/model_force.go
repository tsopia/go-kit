package llm

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type extractionRuntime struct {
	state *extractionState
}

func newExtractionRuntime(maxRetries int) *extractionRuntime {
	return &extractionRuntime{
		state: &extractionState{maxRetries: maxRetries},
	}
}

func (r *extractionRuntime) wrapModel(inner model.ToolCallingChatModel) model.ToolCallingChatModel {
	return &extractionToolChoiceModel{inner: inner, state: r.state}
}

func (r *extractionRuntime) middleware() compose.ToolMiddleware {
	return newExtractionRetryMiddleware(r.state)
}

func (r *extractionRuntime) directReturnMessage(directReturnTools map[string]struct{}) (*schema.Message, bool) {
	if len(directReturnTools) == 0 {
		return nil, false
	}
	name, result, ok := r.state.lastSuccessfulTool()
	if !ok {
		return nil, false
	}
	if _, ok := directReturnTools[name]; !ok {
		return nil, false
	}
	return &schema.Message{
		Role:    schema.Assistant,
		Content: result,
	}, true
}

// ── defaultOptsModel 把固定 model.Option 绑定到模型 ──────────────────
//
// 用于把 ModelConfig.ExtraFields 在 config 层绑定到只支持 request-level option
// 的供应商（DeepSeek / Qwen）：构造时把 ExtraFields 转为对应的 model.Option，
// 每次 Generate/Stream 自动注入，用户运行时传入的同名 option 会覆盖默认值。
type defaultOptsModel struct {
	inner model.ToolCallingChatModel
	opts  []model.Option
}

func (m *defaultOptsModel) Generate(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return m.inner.Generate(ctx, msgs, append(m.opts, opts...)...)
}

func (m *defaultOptsModel) Stream(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return m.inner.Stream(ctx, msgs, append(m.opts, opts...)...)
}

func (m *defaultOptsModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	inner, err := m.inner.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &defaultOptsModel{inner: inner, opts: m.opts}, nil
}

// ── extractionToolChoiceModel 动态切换 ToolChoice 的模型包装器 ───────
//
// 行为：
//   - 首次调用 + 失败修复期间：ToolChoiceForced（强制调工具）
//   - 首次工具调用成功后：ToolChoiceAllowed（模型自由决策）
type extractionToolChoiceModel struct {
	inner model.ToolCallingChatModel
	state *extractionState
}

func (m *extractionToolChoiceModel) Generate(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	if m.state.shouldForceToolCall() {
		opts = append(opts, model.WithToolChoice(schema.ToolChoiceForced))
	}
	return m.inner.Generate(ctx, msgs, opts...)
}

func (m *extractionToolChoiceModel) Stream(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	if m.state.shouldForceToolCall() {
		opts = append(opts, model.WithToolChoice(schema.ToolChoiceForced))
	}
	return m.inner.Stream(ctx, msgs, opts...)
}

func (m *extractionToolChoiceModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	inner, err := m.inner.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &extractionToolChoiceModel{inner: inner, state: m.state}, nil
}

// ── newExtractionRetryMiddleware 拦截工具错误，转为结果字符串让模型修复 ──
//
// 行为：
//   - 工具成功 → state.recordSuccess()，正常返回
//   - 工具失败且未超限 → state.recordFailure()，错误转为结果字符串
//   - 工具失败且超限 → 传播错误，终止 Agent
func newExtractionRetryMiddleware(state *extractionState) compose.ToolMiddleware {
	return compose.ToolMiddleware{
		Invokable: func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
			return func(ctx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
				output, err := next(ctx, input)
				if err != nil {
					state.recordFailure()
					retriesExhausted := state.retriesExhausted()
					if logState := toolLogStateFromContext(ctx); logState != nil {
						logState.markError(err, !retriesExhausted, retriesExhausted)
					}
					if retriesExhausted {
						return nil, fmt.Errorf("%w: tool %s (%d attempts): %w", ErrExtractionRetriesExhausted, input.Name, state.maxRetries, err)
					}
					return &compose.ToolOutput{
						Result: fmt.Sprintf("工具执行失败: %v\n请分析错误原因，调整参数后重新调用工具。", err),
					}, nil
				}
				state.recordSuccess(input.Name, output.Result)

				// 优化：如果配置了 ToolReturnDirectly，则在工具成功后立即中断（节省 Model Token）
				// 注意：ctxKeyAgentControl 和 agentControl 定义在 agent.go 中
				if ctrl, ok := ctx.Value(ctxKeyAgentControl{}).(*agentControl); ok {
					if ctrl.shouldReturnDirectly(input.Name) {
						ctrl.cancel()
					}
				}

				return output, nil
			}
		},
	}
}
