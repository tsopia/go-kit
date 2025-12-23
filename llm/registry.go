package llm

import (
	"context"
	"fmt"
	"sync"
)

// Builder 构造具体 provider 的 Adapter。
type Builder func(ctx context.Context, cfg Config) (Adapter, error)

var (
	mu       sync.RWMutex
	builders = make(map[Provider]Builder)
)

// RegisterProvider 注册新的 provider 构造器。
func RegisterProvider(provider Provider, builder Builder) {
	mu.Lock()
	defer mu.Unlock()
	builders[provider] = builder
}

func getBuilder(provider Provider) (Builder, bool) {
	mu.RLock()
	defer mu.RUnlock()
	builder, ok := builders[provider]
	return builder, ok
}

// NewClient 根据配置创建通用 LLM Client。
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	builder, ok := getBuilder(cfg.Provider)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotRegistered, cfg.Provider)
	}

	adapter, err := builder(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("初始化 provider(%s) 失败: %w", cfg.description(), err)
	}

	return &Client{
		provider:       cfg.Provider,
		adapter:        adapter,
		defaultOptions: cfg.Options,
	}, nil
}

// Client 对外暴露的统一入口。
type Client struct {
	provider       Provider
	adapter        Adapter
	defaultOptions RequestOptions
}

// Chat 统一的非流式接口。
func (c *Client) Chat(ctx context.Context, messages []Message, opts ...Option) (*ChatCompletion, error) {
	if err := validateMessages(messages); err != nil {
		return nil, err
	}

	req := ChatRequest{
		Messages: messages,
		Options:  mergeOptions(c.defaultOptions, opts...),
	}

	return c.adapter.Chat(ctx, req)
}

// Stream 尝试流式调用，若 provider 未实现则返回 ErrStreamNotSupported。
func (c *Client) Stream(ctx context.Context, messages []Message, opts ...Option) (<-chan ChatStreamChunk, func(), error) {
	streamable, ok := c.adapter.(StreamingAdapter)
	if !ok {
		return nil, nil, ErrStreamNotSupported
	}

	if err := validateMessages(messages); err != nil {
		return nil, nil, err
	}

	req := ChatRequest{
		Messages: messages,
		Options:  mergeOptions(c.defaultOptions, opts...),
	}

	return streamable.Stream(ctx, req)
}
