package llm

import (
	"context"
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
	KIMI          ModelProtocol = "KIMI"
)

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
}

// InvokableTool 代表一个可执行工具（定义 + 执行能力）。
// 这是一个简化接口，可通过 ToolAdapter 适配到 Eino 标准 tool.InvokableTool。
type InvokableTool interface {
	Info() *schema.ToolInfo
	Invoke(ctx context.Context, args string) (string, error)
}

// NewModel 根据 Protocol 创建对应的 eino-ext ToolCallingChatModel。
func NewModel(ctx context.Context, cfg ModelConfig) (model.ToolCallingChatModel, error) {
	if cfg.Model == "" {
		return nil, errors.New("model is required")
	}
	if isCompatProtocol(cfg.Protocol) && cfg.BaseURL == "" {
		return nil, errors.New("base url is required for compat protocol")
	}
	if cfg.Protocol != OLLAMA && cfg.Protocol != QIANFAN && cfg.APIKey == "" {
		return nil, fmt.Errorf("API Key is required for protocol %s", cfg.Protocol)
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
	case KIMI:
		m, err = newKimiModel(ctx, cfg)
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
	return m, nil
}

func isCompatProtocol(p ModelProtocol) bool {
	return p == OPENAI_COMPAT || p == CLAUDE_COMPAT
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
	} else {
		c.BaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return einoqwen.NewChatModel(ctx, c)
}

func newKimiModel(ctx context.Context, cfg ModelConfig) (model.ToolCallingChatModel, error) {
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
	} else {
		c.BaseURL = "https://api.moonshot.cn/v1"
	}
	return einoopenai.NewChatModel(ctx, c)
}
