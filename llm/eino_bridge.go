package llm

import (
	"context"
	"fmt"
)

// EinoExecutor 抽象出 eino 风格的执行器，方便用户在外部直接使用官方实现。
// 该接口只要求一个 Chat 方法，与官方 schema 解耦，避免在无法联网时阻塞编译。
type EinoExecutor interface {
	Chat(ctx context.Context, messages []Message, options RequestOptions) (*ChatCompletion, error)
}

// EinoAgent 兼容 Eino Agent/编排接口。
type EinoAgent interface {
	Run(ctx context.Context, messages []Message, options RequestOptions) (*ChatCompletion, error)
}

// RegisterEinoProvider 帮助方法：接受 eino executor，并封装为 Adapter。
func RegisterEinoProvider(provider Provider, constructor func(cfg Config) (EinoExecutor, error)) {
	RegisterProvider(provider, func(ctx context.Context, cfg Config) (Adapter, error) {
		exec, err := constructor(cfg)
		if err != nil {
			return nil, fmt.Errorf("构建 eino executor 失败: %w", err)
		}
		return &einoAdapter{executor: exec}, nil
	})
}

type einoAdapter struct {
	executor EinoExecutor
}

func (a *einoAdapter) Chat(ctx context.Context, req ChatRequest) (*ChatCompletion, error) {
	return a.executor.Chat(ctx, req.Messages, req.Options)
}

// RegisterEinoAgentProvider 将 Eino Agent/Flow 扩展适配为统一接口。
func RegisterEinoAgentProvider(provider Provider, constructor func(cfg Config) (EinoAgent, error)) {
	RegisterProvider(provider, func(ctx context.Context, cfg Config) (Adapter, error) {
		agent, err := constructor(cfg)
		if err != nil {
			return nil, fmt.Errorf("构建 eino agent 失败: %w", err)
		}
		return &einoAgentAdapter{agent: agent}, nil
	})
}

type einoAgentAdapter struct {
	agent EinoAgent
}

func (a *einoAgentAdapter) Chat(ctx context.Context, req ChatRequest) (*ChatCompletion, error) {
	return a.agent.Run(ctx, req.Messages, req.Options)
}
