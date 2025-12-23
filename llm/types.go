package llm

import "time"

// Provider 代表内置或自定义的大模型供应商。
type Provider string

const (
	// ProviderOpenAI 使用 OpenAI Chat Completions 协议。
	ProviderOpenAI Provider = "openai"
	// ProviderEinoAgent 预留给 Eino Agent 端到端编排（自定义注册时使用）。
	ProviderEinoAgent Provider = "eino-agent"
)

// MessageRole 统一的消息角色定义。
type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

// Message 聊天消息定义，兼容主流 Chat Completions 协议。
type Message struct {
	Role       MessageRole `json:"role" yaml:"role"`
	Content    string      `json:"content" yaml:"content"`
	Name       string      `json:"name,omitempty" yaml:"name,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty" yaml:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty" yaml:"tool_call_id,omitempty"`
}

// CompletionChoice 单次回复的选择。
type CompletionChoice struct {
	Index        int         `json:"index"`
	Message      Message     `json:"message"`
	FinishReason string      `json:"finish_reason"`
	ProviderMeta interface{} `json:"provider_meta,omitempty"`
}

// CompletionUsage 记录 tokens 用量。
type CompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatCompletion 标准化的回复结构。
type ChatCompletion struct {
	ID       string             `json:"id"`
	Model    string             `json:"model"`
	Provider Provider           `json:"provider"`
	Created  time.Time          `json:"created"`
	Choices  []CompletionChoice `json:"choices"`
	Usage    CompletionUsage    `json:"usage"`
	Raw      interface{}        `json:"raw,omitempty"`
}

// ToolFunctionCall 代表一次函数调用。
type ToolFunctionCall struct {
	Name      string `json:"name" yaml:"name"`
	Arguments string `json:"arguments" yaml:"arguments"` // 原始 JSON 字符串
}

// ToolCall 表示一次工具调用（assistant 侧），或工具响应（tool 侧）。
type ToolCall struct {
	ID       string           `json:"id" yaml:"id"`
	Type     string           `json:"type" yaml:"type"` // 目前使用 "function"
	Function ToolFunctionCall `json:"function" yaml:"function"`
}

// ToolDefinition 用于向模型暴露可用工具。
type ToolDefinition struct {
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Parameters  map[string]any         `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty" yaml:"meta,omitempty"`
}
