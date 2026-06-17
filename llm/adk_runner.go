package llm

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"
)

// runADK 驱动非流式 runner，消费事件流，取最终 assistant 消息。
// 适配 adk 的事件迭代器模型为同步 Generate API。
func runADK(runner *adk.Runner, ctx context.Context, messages []*schema.Message, handlers []callbacks.Handler, extra ...adk.AgentRunOption) (*schema.Message, error) {
	iter := runner.Run(ctx, messages, append(toRunOptions(handlers), extra...)...)
	var finalMsg *schema.Message
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return nil, event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		mv := event.Output.MessageOutput
		// 非流式模式下取最后一个非流式 Message 作为最终结果
		if !mv.IsStreaming && mv.Message != nil {
			finalMsg = mv.Message
		}
	}
	if finalMsg == nil {
		return nil, fmt.Errorf("agent produced no message output")
	}
	return finalMsg, nil
}

// streamADK 驱动流式 runner，找到流式 MessageStream 并返回。
// 工具调用等中间事件被跳过，最终模型的流式输出直接返回给调用方。
func streamADK(runner *adk.Runner, ctx context.Context, messages []*schema.Message, handlers []callbacks.Handler, extra ...adk.AgentRunOption) (*schema.StreamReader[*schema.Message], error) {
	iter := runner.Run(ctx, messages, append(toRunOptions(handlers), extra...)...)
	for {
		event, ok := iter.Next()
		if !ok {
			return nil, fmt.Errorf("agent produced no streaming output")
		}
		if event.Err != nil {
			return nil, event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		mv := event.Output.MessageOutput
		if mv.IsStreaming && mv.MessageStream != nil {
			return mv.MessageStream, nil
		}
	}
}

// toRunOptions 将 callback handlers 转为 adk AgentRunOption。
func toRunOptions(handlers []callbacks.Handler) []adk.AgentRunOption {
	if len(handlers) == 0 {
		return nil
	}
	return []adk.AgentRunOption{adk.WithCallbacks(handlers...)}
}
