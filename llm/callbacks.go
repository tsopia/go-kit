package llm

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/callbacks/langfuse"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// LangfuseConfig 是 Langfuse 的配置。
type LangfuseConfig struct {
	Host      string
	PublicKey string
	SecretKey string
}

// NewLangfuseHandler 创建一个 Langfuse 回调处理器。
// 返回的 flush 函数应该在程序退出前调用，确保所有 Trace 上报完成。
func NewLangfuseHandler(cfg *LangfuseConfig) (callbacks.Handler, func(), error) {
	// 使用 eino-ext 的 Langfuse 实现
	handler, flush := langfuse.NewLangfuseHandler(&langfuse.Config{
		Host:      cfg.Host,
		PublicKey: cfg.PublicKey,
		SecretKey: cfg.SecretKey,
	})
	return handler, flush, nil
}

// NewLogHandler 创建一个基于 LogClient 的日志回调处理器。
// 它会记录组件的输入、输出和 Token 消耗（如果有），并沿用调用时的 ctx。
func NewLogHandler(client LogClient) callbacks.Handler {
	if isNilLogClient(client) {
		client = &noopLogClient{}
	}

	return callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			fields := []any{
				"component", info.Component,
				"name", info.Name,
				"type", info.Type,
				"input", fmt.Sprintf("%.100v", input), // 简单截断防止日志过大
			}
			fields = appendInvocationIDField(ctx, fields)
			client.Info(ctx, "Component Start", fields...)
			return ctx
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			attrs := []any{
				"component", string(info.Component),
				"name", info.Name,
				"type", info.Type,
			}

			// 尝试提取 Token Usage
			if cbOut, ok := output.(*model.CallbackOutput); ok && cbOut.TokenUsage != nil {
				attrs = append(attrs, "token_usage", cbOut.TokenUsage)
			} else if cbOut, ok := output.(*schema.Message); ok {
				// 有些 Output 直接就是 Message
				attrs = append(attrs, "content", fmt.Sprintf("%.100s", cbOut.Content))
			} else {
				attrs = append(attrs, "output", fmt.Sprintf("%.100v", output))
			}

			attrs = appendInvocationIDField(ctx, attrs)
			client.Info(ctx, "Component End", attrs...)
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			fields := []any{
				"component", info.Component,
				"name", info.Name,
				"error", err,
			}
			fields = appendInvocationIDField(ctx, fields)
			client.Error(ctx, "Component Error", fields...)
			return ctx
		}).
		Build()
}

type noopLogClient struct{}

func (n *noopLogClient) Info(context.Context, string, ...any)  {}
func (n *noopLogClient) Error(context.Context, string, ...any) {}
