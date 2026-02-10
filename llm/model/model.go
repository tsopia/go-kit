package model

import (
	"context"
	"encoding/json"
	"io"

	"github.com/tsopia/go-kit/llm/tool"
)

type ToolCall struct {
	Name      string
	Arguments json.RawMessage
}

type ChatMessage struct {
	Role      string
	Content   string
	ToolCalls []ToolCall
}

type ToolCallingChatModel interface {
	WithTools(...tool.InvokableTool) (ToolCallingChatModel, error)
	Generate(ctx context.Context, messages []ChatMessage) (ChatMessage, error)
}

// ChatMessageStream 表示模型流式输出。
// 调用方应持续 Recv 直到返回 io.EOF。
type ChatMessageStream interface {
	Recv() (ChatMessage, error)
}

// StreamableToolCallingChatModel 为可选扩展接口：
// 若模型实现该接口，可用于流式交互。
type StreamableToolCallingChatModel interface {
	ToolCallingChatModel
	GenerateStream(ctx context.Context, messages []ChatMessage) (ChatMessageStream, error)
}

// SliceMessageStream 是一个轻量内存流实现，主要用于测试或适配场景。
type SliceMessageStream struct {
	items []ChatMessage
	idx   int
}

func NewSliceMessageStream(items []ChatMessage) *SliceMessageStream {
	cloned := append([]ChatMessage(nil), items...)
	return &SliceMessageStream{items: cloned}
}

func (s *SliceMessageStream) Recv() (ChatMessage, error) {
	if s.idx >= len(s.items) {
		return ChatMessage{}, io.EOF
	}
	msg := s.items[s.idx]
	s.idx++
	return msg, nil
}
