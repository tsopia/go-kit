package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/tsopia/go-kit/llm/model"
	"github.com/tsopia/go-kit/llm/tool"
)

type ModelProtocol string

const (
	OPENAI_COMPAT ModelProtocol = "OPENAI_COMPAT"
	CLAUDE_COMPAT ModelProtocol = "CLAUDE_COMPAT"
	ARK           ModelProtocol = "ARK"
	DEEPSEEK      ModelProtocol = "DEEPSEEK"
	ARKBOT        ModelProtocol = "ARKBOT"
	CLAUDE        ModelProtocol = "CLAUDE"
	GEMINI        ModelProtocol = "GEMINI"
	OLLAMA        ModelProtocol = "OLLAMA"
	OPENAI        ModelProtocol = "OPENAI"
	QIANFAN       ModelProtocol = "QIANFAN"
	QWEN          ModelProtocol = "QWEN"
)

type ToolResultPolicy string

const (
	RETURN_FINAL_ANSWER ToolResultPolicy = "RETURN_FINAL_ANSWER"
	RETURN_TOOL_RESULT  ToolResultPolicy = "RETURN_TOOL_RESULT"
	RETURN_BOTH         ToolResultPolicy = "RETURN_BOTH"
)

type ToolCallMode string

const (
	TOOL_OPTIONAL       ToolCallMode = "TOOL_OPTIONAL"
	TOOL_REQUIRED_ONE   ToolCallMode = "TOOL_REQUIRED_ONE"
	TOOL_REQUIRED_EXACT ToolCallMode = "TOOL_REQUIRED_EXACT"
)

type ToolCallPolicy struct {
	Mode             ToolCallMode
	AllowedTools     []string
	RequiredToolName string
}

type ModelConfig struct {
	Protocol ModelProtocol
	BaseURL  string
	APIKey   string
	Model    string
	Timeout  time.Duration

	MaxTokens   *int
	Temperature *float32
	TopP        *float32
	Stop        []string

	ToolCallPolicy   ToolCallPolicy
	ToolResultPolicy ToolResultPolicy
}

type RunOptions struct {
	MaxRetries int
}

type StopReason string

const (
	STOP_MODEL_FINAL          StopReason = "MODEL_FINAL"
	STOP_TOOL_RESULT_RETURNED StopReason = "TOOL_RESULT_RETURNED"
	STOP_MAX_RETRIES          StopReason = "MAX_RETRIES"
	STOP_ERROR                StopReason = "ERROR"
)

type RunResult struct {
	FinalText string

	ToolName   string
	ToolArgs   json.RawMessage
	ToolResult json.RawMessage

	StopReason StopReason
}

type toolCallingModel struct {
	cfg   ModelConfig
	tools []tool.InvokableTool
}

func NewToolCallingModel(cfg ModelConfig) (model.ToolCallingChatModel, error) {
	cfg = cfg.normalized()
	if isCompatProtocol(cfg.Protocol) && cfg.BaseURL == "" {
		return nil, errors.New("base url is required for compat protocol")
	}
	if cfg.APIKey == "" && cfg.Protocol != OLLAMA {
		return nil, errors.New("api key is required")
	}
	if cfg.Model == "" {
		return nil, errors.New("model is required")
	}
	if isSupportedProtocol(cfg.Protocol) {
		return &toolCallingModel{cfg: cfg}, nil
	}
	return nil, fmt.Errorf("unsupported protocol: %s", cfg.Protocol)
}

func isCompatProtocol(p ModelProtocol) bool {
	return p == OPENAI_COMPAT || p == CLAUDE_COMPAT
}

func isSupportedProtocol(p ModelProtocol) bool {
	switch p {
	case OPENAI_COMPAT, CLAUDE_COMPAT, ARK, DEEPSEEK, ARKBOT, CLAUDE, GEMINI, OLLAMA, OPENAI, QIANFAN, QWEN:
		return true
	default:
		return false
	}
}

func (m *toolCallingModel) WithTools(ts ...tool.InvokableTool) (model.ToolCallingChatModel, error) {
	cloned := *m
	cloned.tools = append([]tool.InvokableTool(nil), ts...)
	return &cloned, nil
}

func (m *toolCallingModel) Generate(_ context.Context, _ []model.ChatMessage) (model.ChatMessage, error) {
	return model.ChatMessage{}, errors.New("model backend is not configured for direct generation")
}

func (m *toolCallingModel) GetModelConfig() ModelConfig {
	return m.cfg
}

func (o RunOptions) normalizedMaxRetries() int {
	if o.MaxRetries <= 0 {
		return 3
	}
	return o.MaxRetries
}

func (c ModelConfig) normalized() ModelConfig {
	if c.ToolCallPolicy.Mode == "" {
		c.ToolCallPolicy.Mode = TOOL_OPTIONAL
	}
	if c.ToolResultPolicy == "" {
		c.ToolResultPolicy = RETURN_FINAL_ANSWER
	}
	return c
}
