package model

import (
	"context"
	"encoding/json"

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
