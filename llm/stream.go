package llm

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/tsopia/go-kit/llm/model"
	"github.com/tsopia/go-kit/llm/tool"
)

var ErrStreamNotSupported = errors.New("model does not support streaming")

// RunToolCallLoopStream 与 RunToolCallLoop 类似，但在可流式模型上返回最终流。
// 该函数聚焦于无工具调用的流式输出场景；当模型产生 tool call 时返回错误。
func RunToolCallLoopStream(ctx context.Context, m model.ToolCallingChatModel, tools []tool.InvokableTool, opts RunOptions) (model.ChatMessageStream, error) {
	boundModel, err := m.WithTools(tools...)
	if err != nil {
		return nil, err
	}

	streamModel, ok := boundModel.(model.StreamableToolCallingChatModel)
	if !ok {
		return nil, ErrStreamNotSupported
	}

	cfg := ModelConfig{}.normalized()
	if p, ok := m.(configProvider); ok {
		cfg = p.GetModelConfig().normalized()
	}

	maxRetries := opts.normalizedMaxRetries()
	feedback := make([]model.ChatMessage, 0, maxRetries)

	for retries := 0; retries <= maxRetries; retries++ {
		stream, streamErr := streamModel.GenerateStream(ctx, feedback)
		if streamErr != nil {
			return nil, streamErr
		}

		messages, collectErr := collectStreamMessages(stream)
		if collectErr != nil {
			return nil, collectErr
		}

		last := lastNonEmptyMessage(messages)
		if len(last.ToolCalls) == 0 {
			if cfg.ToolCallPolicy.Mode == TOOL_REQUIRED_ONE {
				feedback = append(feedback, model.ChatMessage{Role: "system", Content: requiredOneFeedback(nil)})
				continue
			}
			if cfg.ToolCallPolicy.Mode == TOOL_REQUIRED_EXACT {
				feedback = append(feedback, model.ChatMessage{Role: "system", Content: requiredExactFeedback(cfg.ToolCallPolicy.RequiredToolName)})
				continue
			}
			return model.NewSliceMessageStream(messages), nil
		}

		return nil, errors.New("stream loop does not support tool calls; use RunToolCallLoop")
	}

	return nil, errors.New("max retries exceeded in stream loop")
}

// ConcatStreamContent 读取流中的所有片段并拼接 Content。
func ConcatStreamContent(stream model.ChatMessageStream) (string, error) {
	if stream == nil {
		return "", nil
	}
	var b strings.Builder
	for {
		msg, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}
		b.WriteString(msg.Content)
	}
	return b.String(), nil
}

func collectStreamMessages(stream model.ChatMessageStream) ([]model.ChatMessage, error) {
	messages := make([]model.ChatMessage, 0, 8)
	for {
		msg, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		messages = append(messages, msg)
	}
	if len(messages) == 0 {
		messages = append(messages, model.ChatMessage{})
	}
	return messages, nil
}

func lastNonEmptyMessage(messages []model.ChatMessage) model.ChatMessage {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Content != "" || len(messages[i].ToolCalls) > 0 {
			return messages[i]
		}
	}
	return messages[len(messages)-1]
}
