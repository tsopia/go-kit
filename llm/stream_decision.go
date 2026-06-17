package llm

import (
	"context"
	"errors"
	"io"

	"github.com/cloudwego/eino/schema"
)

// observeStreamDecision 包装最终输出流：边转发 chunk 给调用方，边累积，
// 在流结束(EOF)时拼接还原完整消息并记录一条 model.decision（含 usage）。
//
// 填补「流式下 model.decision 不记录」的缺口（O-008），并附带 token 用量（O-009）。
// logs 关闭时原样返回，零开销、不引入额外 goroutine。
func observeStreamDecision(
	ctx context.Context,
	sr *schema.StreamReader[*schema.Message],
	logs *structuredLogger,
	mode ExecutionMode,
	toolChoice schema.ToolChoice,
) *schema.StreamReader[*schema.Message] {
	if logs == nil || !logs.enabled() {
		return sr
	}

	newSr, sw := schema.Pipe[*schema.Message](1)
	go func() {
		var chunks []*schema.Message
		defer func() {
			sr.Close()
			sw.Close()
			if msg, err := schema.ConcatMessages(chunks); err == nil && msg != nil {
				logStreamDecision(ctx, logs, mode, toolChoice, msg)
			}
		}()
		for {
			chunk, err := sr.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					sw.Send(chunk, err)
				}
				return
			}
			chunks = append(chunks, chunk)
			// 读端提前 Close 时退出，已累积的 chunk 仍会在 defer 中尽力记录。
			if sw.Send(chunk, nil) {
				return
			}
		}
	}()
	return newSr
}

func logStreamDecision(ctx context.Context, logs *structuredLogger, mode ExecutionMode, toolChoice schema.ToolChoice, msg *schema.Message) {
	toolNames, toolCallCount := decisionToolNames(msg)
	finishReason, reasoningTokens := decisionMeta(msg)
	attrs := []any{
		"execution_mode", string(mode),
		"configured_tool_choice", string(toolChoice),
		"tool_call_count", toolCallCount,
		"tool_names", toolNames,
		"finish_reason", finishReason,
		"streaming", true,
	}
	if reasoningTokens >= 0 {
		attrs = append(attrs, "reasoning_tokens", reasoningTokens)
	}
	attrs = appendUsageAttrs(attrs, msg)
	logs.logInfo(ctx, "model.decision", attrs...)
}

// appendUsageAttrs 追加 token 用量字段（O-009）。无 usage 时不追加。
func appendUsageAttrs(attrs []any, msg *schema.Message) []any {
	if msg == nil || msg.ResponseMeta == nil || msg.ResponseMeta.Usage == nil {
		return attrs
	}
	u := msg.ResponseMeta.Usage
	return append(attrs,
		"prompt_tokens", u.PromptTokens,
		"completion_tokens", u.CompletionTokens,
		"total_tokens", u.TotalTokens,
	)
}

// toolChoiceForMode 根据执行模式推导 configured_tool_choice，
// 与 compileRuntimeSpec 的映射保持一致（流式 decision 日志用）。
func toolChoiceForMode(mode ExecutionMode) schema.ToolChoice {
	switch mode {
	case Conversation:
		return schema.ToolChoiceForbidden
	case Extraction:
		return schema.ToolChoiceForced
	default:
		return schema.ToolChoiceAllowed
	}
}
