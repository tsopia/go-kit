package llm

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

var ErrStreamNotSupported = errors.New("model does not support streaming")

// RunToolCallLoopStream 与 RunToolCallLoop 类似，但在可流式模型上返回流式输出。
// 当前聚焦于「最终答案流式输出」场景；若模型返回 tool call，请使用 RunToolCallLoop。
func RunToolCallLoopStream(
	ctx context.Context,
	m model.ToolCallingChatModel,
	messages []*schema.Message,
	tools []InvokableTool,
	opts RunOptions,
) (*schema.StreamReader[*schema.Message], error) {
	toolInfos := make([]*schema.ToolInfo, len(tools))
	for i, t := range tools {
		toolInfos[i] = t.Info()
	}
	boundModel, err := m.WithTools(toolInfos)
	if err != nil {
		return nil, err
	}

	cfg := ModelConfig{}.normalized()
	if p, ok := m.(configProvider); ok {
		cfg = p.GetModelConfig().normalized()
	}

	maxRetries := opts.normalizedMaxRetries()

	history := make([]*schema.Message, len(messages))
	copy(history, messages)

	for retries := 0; retries <= maxRetries; retries++ {
		stream, streamErr := boundModel.Stream(ctx, history)
		if streamErr != nil {
			return nil, streamErr
		}

		// 收集流内容以判断是否有 tool calls
		collected, collectErr := collectStream(stream)
		if collectErr != nil {
			return nil, collectErr
		}

		last := lastNonEmptyMsg(collected)
		if len(last.ToolCalls) == 0 {
			if cfg.ToolCallPolicy.Mode == TOOL_REQUIRED_ONE {
				history = append(history, &schema.Message{Role: schema.System, Content: requiredOneFeedback(nil)})
				continue
			}
			if cfg.ToolCallPolicy.Mode == TOOL_REQUIRED_EXACT {
				history = append(history, &schema.Message{Role: schema.System, Content: requiredExactFeedback(cfg.ToolCallPolicy.RequiredToolName)})
				continue
			}
			// 返回收集到的消息作为流
			return newSliceStream(collected), nil
		}

		return nil, errors.New("stream loop does not support tool calls; use RunToolCallLoop")
	}

	return nil, errors.New("max retries exceeded in stream loop")
}

// ConcatStreamContent 读取流中的所有片段并拼接 Content。
func ConcatStreamContent(stream *schema.StreamReader[*schema.Message]) (string, error) {
	if stream == nil {
		return "", nil
	}
	defer stream.Close()
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

func collectStream(stream *schema.StreamReader[*schema.Message]) ([]*schema.Message, error) {
	defer stream.Close()
	msgs := make([]*schema.Message, 0, 8)
	for {
		msg, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	if len(msgs) == 0 {
		msgs = append(msgs, &schema.Message{})
	}
	return msgs, nil
}

func lastNonEmptyMsg(msgs []*schema.Message) *schema.Message {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Content != "" || len(msgs[i].ToolCalls) > 0 {
			return msgs[i]
		}
	}
	return msgs[len(msgs)-1]
}

// newSliceStream 将消息切片包装为 StreamReader。
func newSliceStream(msgs []*schema.Message) *schema.StreamReader[*schema.Message] {
	idx := 0
	sr, sw := schema.Pipe[*schema.Message](len(msgs))
	go func() {
		defer sw.Close()
		for idx < len(msgs) {
			sw.Send(msgs[idx], nil)
			idx++
		}
	}()
	return sr
}
