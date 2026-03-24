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
	// Conversation 表示纯对话，不启用工具。
	Conversation ExecutionMode = "conversation"
	// Assistant 表示工具可用，由模型自行决定是否调用。
	Assistant ExecutionMode = "assistant"
	// Extraction 表示先完成工具任务，再决定是否继续总结。
	Extraction ExecutionMode = "extraction"
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
	Standard   []tool.BaseTool
	Invokable  []InvokableTool
	MCPServers []MCPConfig
}

type ExecutionConfig struct {
	// Mode 是推荐配置入口，用于声明 Agent 的高层执行模式。
	Mode ExecutionMode
	// ToolChoice 仅保留给旧配置的兼容路径；新代码应优先使用 Mode。
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
