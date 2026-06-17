package llm

import "errors"

// 包级 sentinel error 定义。
//
// 调用方可通过 errors.Is(err, llm.ErrXxx) 精确判断错误类型，
// 而非依赖脆弱的字符串匹配。所有错误均以 fmt.Errorf("...: %w", ErrXxx)
// 形式包装，保留具体上下文（如 protocol 值）的同时支持 errors.Is。
var (
	// ErrMissingModel 模型名缺失（ModelConfig.Model 为空）。
	ErrMissingModel = errors.New("llm: model is required")

	// ErrMissingBaseURL 兼容协议（OPENAI_COMPAT / CLAUDE_COMPAT）缺少 BaseURL。
	ErrMissingBaseURL = errors.New("llm: base url is required for compat protocol")

	// ErrMissingAPIKey 调用所需 Provider 时缺少 API Key。
	ErrMissingAPIKey = errors.New("llm: api key is required")

	// ErrUnsupportedProtocol 不支持的 ModelProtocol。
	ErrUnsupportedProtocol = errors.New("llm: unsupported model protocol")

	// ErrInvalidConfig AgentConfig 配置无效或冲突（执行模式与其它字段冲突等）。
	ErrInvalidConfig = errors.New("llm: invalid agent config")

	// ErrExtractionRetriesExhausted Extraction 模式重试耗尽，
	// 模型未能产出符合工具 JSON Schema 的参数。
	ErrExtractionRetriesExhausted = errors.New("llm: extraction retries exhausted")

	// ErrUnknownMCPProtocol 未知的 MCP 协议（仅支持 "stdio" / "sse"）。
	ErrUnknownMCPProtocol = errors.New("llm: unknown mcp protocol")
)
