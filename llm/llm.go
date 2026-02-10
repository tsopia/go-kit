package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	einoark "github.com/cloudwego/eino-ext/components/model/ark"
	einoarkbot "github.com/cloudwego/eino-ext/components/model/arkbot"
	einoclaude "github.com/cloudwego/eino-ext/components/model/claude"
	einodeepseek "github.com/cloudwego/eino-ext/components/model/deepseek"
	einogemini "github.com/cloudwego/eino-ext/components/model/gemini"
	einoollama "github.com/cloudwego/eino-ext/components/model/ollama"
	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	einoqianfan "github.com/cloudwego/eino-ext/components/model/qianfan"
	einoqwen "github.com/cloudwego/eino-ext/components/model/qwen"
)

// ModelProtocol 定义模型厂商协议。
type ModelProtocol string

const (
	OPENAI        ModelProtocol = "OPENAI"
	OPENAI_COMPAT ModelProtocol = "OPENAI_COMPAT"
	CLAUDE        ModelProtocol = "CLAUDE"
	CLAUDE_COMPAT ModelProtocol = "CLAUDE_COMPAT"
	ARK           ModelProtocol = "ARK"
	ARKBOT        ModelProtocol = "ARKBOT"
	DEEPSEEK      ModelProtocol = "DEEPSEEK"
	GEMINI        ModelProtocol = "GEMINI"
	OLLAMA        ModelProtocol = "OLLAMA"
	QIANFAN       ModelProtocol = "QIANFAN"
	QWEN          ModelProtocol = "QWEN"
)

// ToolResultPolicy 决定执行工具后如何返回结果。
type ToolResultPolicy string

const (
	RETURN_FINAL_ANSWER ToolResultPolicy = "RETURN_FINAL_ANSWER"
	RETURN_TOOL_RESULT  ToolResultPolicy = "RETURN_TOOL_RESULT"
	RETURN_BOTH         ToolResultPolicy = "RETURN_BOTH"
)

// ToolCallMode 决定模型是否必须调用工具。
type ToolCallMode string

const (
	TOOL_OPTIONAL       ToolCallMode = "TOOL_OPTIONAL"
	TOOL_REQUIRED_ONE   ToolCallMode = "TOOL_REQUIRED_ONE"
	TOOL_REQUIRED_EXACT ToolCallMode = "TOOL_REQUIRED_EXACT"
)

// ToolCallPolicy 控制工具调用策略。
type ToolCallPolicy struct {
	Mode             ToolCallMode
	AllowedTools     []string
	RequiredToolName string
}

// ModelConfig 是创建模型的统一配置。
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

// RunOptions 控制 tool loop 执行参数。
type RunOptions struct {
	MaxRetries int
}

// StopReason 描述 loop 停止的原因。
type StopReason string

const (
	STOP_MODEL_FINAL          StopReason = "MODEL_FINAL"
	STOP_TOOL_RESULT_RETURNED StopReason = "TOOL_RESULT_RETURNED"
	STOP_MAX_RETRIES          StopReason = "MAX_RETRIES"
	STOP_ERROR                StopReason = "ERROR"
)

// ToolCallResult 记录单次工具调用的结果。
type ToolCallResult struct {
	ID     string
	Name   string
	Args   string
	Result string
}

// RunResult 是 RunToolCallLoop 的返回值。
type RunResult struct {
	FinalText  string
	ToolCalls  []ToolCallResult
	StopReason StopReason
}

// InvokableTool 代表一个可执行工具（定义 + 执行能力）。
type InvokableTool interface {
	Info() *schema.ToolInfo
	Invoke(ctx context.Context, args string) (string, error)
}

// configProvider 用于从 model 实例获取 ModelConfig（仅限本包创建的 model）。
type configProvider interface {
	GetModelConfig() ModelConfig
}

// modelWithConfig 包装 eino ToolCallingChatModel 并携带 ModelConfig。
type modelWithConfig struct {
	model.ToolCallingChatModel
	cfg ModelConfig
}

func (m *modelWithConfig) GetModelConfig() ModelConfig { return m.cfg }

func (m *modelWithConfig) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	inner, err := m.ToolCallingChatModel.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &modelWithConfig{ToolCallingChatModel: inner, cfg: m.cfg}, nil
}

// NewModel 根据 Protocol 创建对应的 eino-ext ToolCallingChatModel。
func NewModel(ctx context.Context, cfg ModelConfig) (model.ToolCallingChatModel, error) {
	cfg = cfg.normalized()

	if cfg.Model == "" {
		return nil, errors.New("model is required")
	}
	if isCompatProtocol(cfg.Protocol) && cfg.BaseURL == "" {
		return nil, errors.New("base url is required for compat protocol")
	}
	if cfg.APIKey == "" && cfg.Protocol != OLLAMA {
		return nil, errors.New("api key is required")
	}

	var (
		m   model.ToolCallingChatModel
		err error
	)
	switch cfg.Protocol {
	case OPENAI, OPENAI_COMPAT:
		m, err = newOpenAIModel(ctx, cfg)
	case CLAUDE, CLAUDE_COMPAT:
		m, err = newClaudeModel(ctx, cfg)
	case ARK:
		m, err = newARKModel(ctx, cfg)
	case ARKBOT:
		m, err = newARKBotModel(ctx, cfg)
	case DEEPSEEK:
		m, err = newDeepSeekModel(ctx, cfg)
	case GEMINI:
		m, err = newGeminiModel(ctx, cfg)
	case OLLAMA:
		m, err = newOllamaModel(ctx, cfg)
	case QIANFAN:
		m, err = newQianfanModel(ctx, cfg)
	case QWEN:
		m, err = newQwenModel(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", cfg.Protocol)
	}
	if err != nil {
		return nil, fmt.Errorf("create %s model: %w", cfg.Protocol, err)
	}
	return &modelWithConfig{ToolCallingChatModel: m, cfg: cfg}, nil
}

func isCompatProtocol(p ModelProtocol) bool {
	return p == OPENAI_COMPAT || p == CLAUDE_COMPAT
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

// ── Factory functions ───────────────────────────────────────────────

func newOpenAIModel(ctx context.Context, cfg ModelConfig) (model.ToolCallingChatModel, error) {
	c := &einoopenai.ChatModelConfig{
		Model:       cfg.Model,
		APIKey:      cfg.APIKey,
		Timeout:     cfg.Timeout,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
		TopP:        cfg.TopP,
		Stop:        cfg.Stop,
	}
	if cfg.BaseURL != "" {
		c.BaseURL = cfg.BaseURL
	}
	return einoopenai.NewChatModel(ctx, c)
}

func newClaudeModel(ctx context.Context, cfg ModelConfig) (model.ToolCallingChatModel, error) {
	baseURL := cfg.BaseURL
	c := &einoclaude.Config{
		Model:  cfg.Model,
		APIKey: cfg.APIKey,
	}
	if baseURL != "" {
		c.BaseURL = &baseURL
	}
	if cfg.MaxTokens != nil {
		c.MaxTokens = *cfg.MaxTokens
	}
	if cfg.Temperature != nil {
		c.Temperature = cfg.Temperature
	}
	if cfg.Stop != nil {
		c.StopSequences = cfg.Stop
	}
	return einoclaude.NewChatModel(ctx, c)
}

func newARKModel(ctx context.Context, cfg ModelConfig) (model.ToolCallingChatModel, error) {
	c := &einoark.ChatModelConfig{
		Model:       cfg.Model,
		APIKey:      cfg.APIKey,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
		TopP:        cfg.TopP,
		Stop:        cfg.Stop,
	}
	if cfg.BaseURL != "" {
		c.BaseURL = cfg.BaseURL
	}
	if cfg.Timeout > 0 {
		c.Timeout = &cfg.Timeout
	}
	return einoark.NewChatModel(ctx, c)
}

func newARKBotModel(ctx context.Context, cfg ModelConfig) (model.ToolCallingChatModel, error) {
	c := &einoarkbot.Config{
		Model:  cfg.Model,
		APIKey: cfg.APIKey,
	}
	if cfg.BaseURL != "" {
		c.BaseURL = cfg.BaseURL
	}
	if cfg.Timeout > 0 {
		c.Timeout = &cfg.Timeout
	}
	return einoarkbot.NewChatModel(ctx, c)
}

func newDeepSeekModel(ctx context.Context, cfg ModelConfig) (model.ToolCallingChatModel, error) {
	c := &einodeepseek.ChatModelConfig{
		Model:  cfg.Model,
		APIKey: cfg.APIKey,
		Stop:   cfg.Stop,
	}
	if cfg.MaxTokens != nil {
		c.MaxTokens = *cfg.MaxTokens
	}
	if cfg.Temperature != nil {
		c.Temperature = *cfg.Temperature
	}
	if cfg.TopP != nil {
		c.TopP = *cfg.TopP
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return einodeepseek.NewChatModel(ctx, c)
}

func newGeminiModel(ctx context.Context, cfg ModelConfig) (model.ToolCallingChatModel, error) {
	c := &einogemini.Config{
		Model:       cfg.Model,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
		TopP:        cfg.TopP,
	}
	return einogemini.NewChatModel(ctx, c)
}

func newOllamaModel(ctx context.Context, cfg ModelConfig) (model.ToolCallingChatModel, error) {
	c := &einoollama.ChatModelConfig{
		Model:   cfg.Model,
		Timeout: cfg.Timeout,
	}
	if cfg.BaseURL != "" {
		c.BaseURL = cfg.BaseURL
	}
	return einoollama.NewChatModel(ctx, c)
}

func newQianfanModel(ctx context.Context, cfg ModelConfig) (model.ToolCallingChatModel, error) {
	c := &einoqianfan.ChatModelConfig{
		Model: cfg.Model,
	}
	if cfg.Temperature != nil {
		c.Temperature = cfg.Temperature
	}
	return einoqianfan.NewChatModel(ctx, c)
}

func newQwenModel(ctx context.Context, cfg ModelConfig) (model.ToolCallingChatModel, error) {
	c := &einoqwen.ChatModelConfig{
		Model:       cfg.Model,
		APIKey:      cfg.APIKey,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
		TopP:        cfg.TopP,
		Stop:        cfg.Stop,
	}
	if cfg.BaseURL != "" {
		c.BaseURL = cfg.BaseURL
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return einoqwen.NewChatModel(ctx, c)
}

// mustJSON 将任意值编码为 JSON 字符串。
func mustJSON(v any) string {
	if s, ok := v.(string); ok {
		if json.Valid([]byte(s)) {
			return s
		}
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}
