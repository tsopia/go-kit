package llm

import (
	"context"
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
	"google.golang.org/genai"
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

type ThinkingConfig struct {
	Enable       bool
	BudgetTokens int
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

	Thinking    *ThinkingConfig
	ExtraFields map[string]any
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
		return nil, ErrMissingModel
	}
	if isCompatProtocol(cfg.Protocol) && cfg.BaseURL == "" {
		return nil, fmt.Errorf("%w (protocol=%s)", ErrMissingBaseURL, cfg.Protocol)
	}
	if cfg.Protocol != OLLAMA && cfg.Protocol != QIANFAN && cfg.APIKey == "" {
		return nil, fmt.Errorf("%w (protocol=%s)", ErrMissingAPIKey, cfg.Protocol)
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
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedProtocol, cfg.Protocol)
	}
	if err != nil {
		return nil, fmt.Errorf("create %s model: %w", cfg.Protocol, err)
	}
	return m, nil
}

func isCompatProtocol(p ModelProtocol) bool {
	return p == OPENAI_COMPAT || p == CLAUDE_COMPAT
}

func thinkingTypeString(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func mergeOpenAIExtraFields(thinking *ThinkingConfig, userExtra map[string]any) map[string]any {
	result := make(map[string]any, len(userExtra)+1)
	for k, v := range userExtra {
		result[k] = v
	}
	if thinking != nil {
		if _, exists := result["thinking"]; !exists {
			result["thinking"] = map[string]any{"type": thinkingTypeString(thinking.Enable)}
		}
	}
	return result
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
	if extra := mergeOpenAIExtraFields(cfg.Thinking, cfg.ExtraFields); len(extra) > 0 {
		c.ExtraFields = extra
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
	if cfg.Thinking != nil {
		c.Thinking = &einoclaude.Thinking{
			Enable:       cfg.Thinking.Enable,
			BudgetTokens: cfg.Thinking.BudgetTokens,
		}
	}
	if len(cfg.ExtraFields) > 0 {
		c.AdditionalRequestFields = cfg.ExtraFields
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
	if cfg.Thinking != nil {
		c.Thinking = &einoark.Thinking{
			Type: einoark.ThinkingType(thinkingTypeString(cfg.Thinking.Enable)),
		}
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
	if cfg.Thinking != nil {
		c.ThinkingConfig = &einodeepseek.ThinkingConfig{
			Type: thinkingTypeString(cfg.Thinking.Enable),
		}
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
	if cfg.Thinking != nil {
		budget := int32(cfg.Thinking.BudgetTokens)
		c.ThinkingConfig = &genai.ThinkingConfig{
			IncludeThoughts: cfg.Thinking.Enable,
		}
		if budget > 0 {
			c.ThinkingConfig.ThinkingBudget = &budget
		}
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
	if cfg.Thinking != nil {
		c.Thinking = &einoollama.ThinkValue{Value: cfg.Thinking.Enable}
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
	if cfg.Thinking != nil {
		enabled := cfg.Thinking.Enable
		c.EnableThinking = &enabled
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
	if extra := mergeOpenAIExtraFields(cfg.Thinking, cfg.ExtraFields); len(extra) > 0 {
		c.ExtraFields = extra
	}
	return einoopenai.NewChatModel(ctx, c)
}
