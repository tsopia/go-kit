package llm

import (
	"context"
	"fmt"
)

// ToolExecutor 统一的工具执行接口。
type ToolExecutor func(ctx context.Context, call ToolCall) (string, error)

// AgentOrchestrator 简易 Agent 编排器：
// 1. 调用模型获取 tool_calls。
// 2. 顺序执行工具，并将结果（含错误信息）写回上下文。
// 3. 将工具输出追加到对话，再次发送给模型，直到没有 tool_calls 或达到迭代上限。
type AgentOrchestrator struct {
	Client        *Client
	Tools         map[string]ToolExecutor
	MaxIterations int
}

// Run 执行带工具的对话流程。
func (a *AgentOrchestrator) Run(ctx context.Context, messages []Message, opts ...Option) (*ChatCompletion, error) {
	if a.Client == nil {
		return nil, fmt.Errorf("AgentOrchestrator: client 不能为空")
	}
	if a.Tools == nil {
		a.Tools = map[string]ToolExecutor{}
	}
	maxIters := a.MaxIterations
	if maxIters <= 0 {
		maxIters = 4
	}

	history := append([]Message{}, messages...)

	for iter := 0; iter < maxIters; iter++ {
		resp, err := a.Client.Chat(ctx, history, opts...)
		if err != nil {
			return nil, err
		}

		if len(resp.Choices) == 0 {
			return resp, nil
		}

		choice := resp.Choices[0]
		history = append(history, choice.Message)

		if len(choice.Message.ToolCalls) == 0 {
			return resp, nil
		}

		// 执行工具并把结果写回上下文
		for _, call := range choice.Message.ToolCalls {
			toolMsg := Message{
				Role:       RoleTool,
				Content:    "",
				ToolCallID: call.ID,
			}

			exec := a.Tools[call.Function.Name]
			if exec == nil {
				toolMsg.Content = fmt.Sprintf("tool_error: 未找到工具 %s", call.Function.Name)
				history = append(history, toolMsg)
				continue
			}

			result, err := exec(ctx, call)
			if err != nil {
				toolMsg.Content = fmt.Sprintf("tool_error: %v", err)
			} else {
				toolMsg.Content = result
			}
			history = append(history, toolMsg)
		}
	}

	return nil, fmt.Errorf("AgentOrchestrator: 达到最大迭代次数 %d", maxIters)
}
