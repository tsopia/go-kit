package llm

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// ── toolCallTracker 记录工具调用状态 ──────────────────────────────────

type toolCallTracker struct {
	mu            sync.Mutex
	successCount  int
	failureCount  int
	maxRetries    int
	lastToolName  string
	lastToolTotal string // 完整的工具输出（JSON）
}

func (t *toolCallTracker) recordSuccess(name, result string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.successCount++
	t.lastToolName = name
	t.lastToolTotal = result
}

func (t *toolCallTracker) recordFailure() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.failureCount++
}

// shouldForce 判断是否应该强制模型调用工具。
// 逻辑：尚无成功的工具调用 且 失败次数未超限 → 强制。
func (t *toolCallTracker) shouldForce() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.successCount == 0 && t.failureCount < t.maxRetries
}

// retriesExhausted 判断重试次数是否已耗尽。
func (t *toolCallTracker) retriesExhausted() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.failureCount >= t.maxRetries && t.successCount == 0
}

// getLastSuccess 返回最后一次成功的工具调用结果。
func (t *toolCallTracker) getLastSuccess() (name, result string, ok bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.successCount > 0 {
		return t.lastToolName, t.lastToolTotal, true
	}
	return "", "", false
}

// ── forcedToolChoiceModel 动态切换 ToolChoice 的模型包装器 ─────────────
//
// 行为：
//   - 首次调用 + 失败重试期间：ToolChoiceForced（强制调工具）
//   - 首次工具调用成功后：ToolChoiceAllowed（模型自由决策）
type forcedToolChoiceModel struct {
	inner   model.ToolCallingChatModel
	tracker *toolCallTracker
}

func (m *forcedToolChoiceModel) Generate(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	if m.tracker.shouldForce() {
		opts = append(opts, model.WithToolChoice(schema.ToolChoiceForced))
	}
	return m.inner.Generate(ctx, msgs, opts...)
}

func (m *forcedToolChoiceModel) Stream(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	if m.tracker.shouldForce() {
		opts = append(opts, model.WithToolChoice(schema.ToolChoiceForced))
	}
	return m.inner.Stream(ctx, msgs, opts...)
}

func (m *forcedToolChoiceModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	inner, err := m.inner.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &forcedToolChoiceModel{inner: inner, tracker: m.tracker}, nil
}

// ── retryMiddleware 拦截工具错误，转为结果字符串让模型重试 ──────────────
//
// 行为：
//   - 工具成功 → tracker.recordSuccess()，正常返回
//   - 工具失败且未超限 → tracker.recordFailure()，错误转为结果字符串
//   - 工具失败且超限 → 传播错误，终止 Agent
func newRetryMiddleware(tracker *toolCallTracker) compose.ToolMiddleware {
	return compose.ToolMiddleware{
		Invokable: func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
			return func(ctx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
				output, err := next(ctx, input)
				if err != nil {
					tracker.recordFailure()
					if tracker.retriesExhausted() {
						return nil, fmt.Errorf("tool %s: max retries (%d) exceeded: %w", input.Name, tracker.maxRetries, err)
					}
					return &compose.ToolOutput{
						Result: fmt.Sprintf("工具执行失败: %v\n请分析错误原因，调整参数后重新调用工具。", err),
					}, nil
				}
				tracker.recordSuccess(input.Name, output.Result)

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
