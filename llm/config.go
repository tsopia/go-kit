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
	// Deprecated: ToolChoice 仅保留给 legacy-only 兼容路径；新代码应优先使用 Mode。
	ToolChoice *schema.ToolChoice
	// MaxRetries 仅用于 Extraction；Conversation 和 Assistant 会拒绝该配置。
	MaxRetries int
	MaxStep    int
	// DirectReturnTools 仅允许引用已注册的工具名；Conversation 会拒绝该配置。
	DirectReturnTools map[string]struct{}
}

type StreamingConfig struct {
	ToolCallChecker func(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error)
}

type ObservabilityConfig struct {
	Callbacks      []callbacks.Handler
	StructuredLogs *StructuredLogConfig
}

type StructuredLogConfig struct {
	// Client 负责输出 llm 的结构化日志；建议直接传入支持 ctx 的日志客户端。
	Client           LogClient
	LogToolArguments bool
	LogToolResults   bool
	MaxFieldLength   int
}

type AgentConfig struct {
	Model         AgentModelConfig
	Prompt        PromptConfig
	Tools         ToolsConfig
	Execution     ExecutionConfig
	Streaming     StreamingConfig
	Observability ObservabilityConfig
	Concurrency   ConcurrencyConfig
}

// ConcurrencyConfig 控制单个 Agent 实例的最大并发调用数。
type ConcurrencyConfig struct {
	// MaxConcurrency 限制同一 Agent 实例上 Generate/Stream 的并发数。
	// 0 表示不限制（默认）。到达上限后新的调用会阻塞等待已有调用释放名额，
	// 等待中的调用可被 context 取消打断。
	MaxConcurrency int
}
