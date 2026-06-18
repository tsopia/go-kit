package llm

import (
	"context"

	"github.com/cloudwego/eino/adk"
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

	// Aliases 工具名别名：key=规范工具名，value=别名列表。
	// 当模型输出别名（如旧工具名）时，会被解析回规范工具名。
	Aliases map[string][]string

	// UnknownHandler 处理模型调用了未注册工具（幻觉工具名）的情况。
	// 返回的字符串作为 ToolResult 回传给模型，让 Agent 自行纠错；
	// 若为 nil，调用未知工具会返回错误（现有行为）。
	UnknownHandler func(ctx context.Context, name, input string) (string, error)

	// ArgumentsFixer 在工具执行前修复/改写参数 JSON（如去除 trailing comma）。
	// 若为 nil，参数原样透传。
	ArgumentsFixer func(ctx context.Context, name, arguments string) (string, error)

	// ErrorToText 为 true 时，工具执行错误（含 panic）被转为 ToolResult 文本回传给
	// 模型，而非中断 Agent 流程（生产推荐）。
	// 注意：为保持向后兼容，**默认（nil）为关闭**，行为与现状一致；需显式设置 true 开启。
	//
	// 安全提示：错误文本会原样发给模型（并可能出现在最终回复中），原始 error 若含
	// 内部细节（数据库字段、内网地址、堆栈等）存在泄露风险。如工具错误可能携带敏感
	// 信息，应在工具内部先脱敏，或不开启本选项而自行处理错误。
	ErrorToText *bool
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

	// Middlewares 是用户自定义的 ADK ChatModelAgentMiddleware，仅 NewADKAgent
	// 路径生效（NewAgent legacy 路径忽略）。可用于接入 Eino 生态的内置
	// Middleware（如 ModelRetry / ModelFailover）或任意自定义钩子。
	//
	// 执行顺序：用户 Middleware 先于包内 Middleware（extraction / observability）
	// 注册，因此用户钩子可观察/拦截到包内行为。
	Middlewares []adk.ChatModelAgentMiddleware
}

// ErrorToTextEnabled 返回一个指向 true 的 *bool，便于内联配置 ToolsConfig.ErrorToText：
//
//	Tools: llm.ToolsConfig{ErrorToText: llm.ErrorToTextEnabled()}
func ErrorToTextEnabled() *bool { v := true; return &v }

// ConcurrencyConfig 控制单个 Agent 实例的最大并发调用数。
type ConcurrencyConfig struct {
	// MaxConcurrency 限制同一 Agent 实例上 Generate/Stream 的并发数。
	// 0 表示不限制（默认）。到达上限后新的调用会阻塞等待已有调用释放名额，
	// 等待中的调用可被 context 取消打断。
	MaxConcurrency int
}
