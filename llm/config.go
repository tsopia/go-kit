package llm

import (
	"context"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type ExecutionMode string

const (
	Conversation ExecutionMode = "conversation"
	Assistant    ExecutionMode = "assistant"
	Extraction   ExecutionMode = "extraction"
)

type AgentModelConfig struct {
	Config   ModelConfig
	Instance model.ToolCallingChatModel
}

type PromptConfig struct {
	System          string
	PrepareMessages MessageModifier
	RewriteHistory  MessageModifier
}

type ToolsConfig struct {
	Standard  []tool.BaseTool
	Invokable []InvokableTool
	MCPServers []MCPConfig
}

type ExecutionConfig struct {
	Mode              ExecutionMode
	ToolChoice        *schema.ToolChoice
	MaxRetries        int
	MaxStep           int
	DirectReturnTools map[string]struct{}
}

type StreamingConfig struct {
	ToolCallChecker func(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error)
}

type ObservabilityConfig struct {
	Callbacks []callbacks.Handler
}

type AgentConfig struct {
	Model         AgentModelConfig
	Prompt        PromptConfig
	Tools         ToolsConfig
	Execution     ExecutionConfig
	Streaming     StreamingConfig
	Observability ObservabilityConfig
}
