package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Config 定义一个模型的基础配置，可从配置文件直接映射。
type Config struct {
	Provider Provider          `json:"provider" yaml:"provider" mapstructure:"provider"`
	Model    string            `json:"model" yaml:"model" mapstructure:"model"`
	BaseURL  string            `json:"base_url,omitempty" yaml:"base_url,omitempty" mapstructure:"base_url"`
	APIKey   string            `json:"api_key,omitempty" yaml:"api_key,omitempty" mapstructure:"api_key"`
	Version  string            `json:"version,omitempty" yaml:"version,omitempty" mapstructure:"version"`
	Timeout  time.Duration     `json:"timeout,omitempty" yaml:"timeout,omitempty" mapstructure:"timeout"`
	Options  RequestOptions    `json:"options,omitempty" yaml:"options,omitempty" mapstructure:"options"`
	Headers  map[string]string `json:"headers,omitempty" yaml:"headers,omitempty" mapstructure:"headers"`
}

func (c Config) validate() error {
	if c.Provider == "" {
		return errors.New("llm provider 不能为空")
	}
	if strings.TrimSpace(c.Model) == "" && c.Provider == ProviderOpenAI {
		return errors.New("model 不能为空")
	}
	return nil
}

func (c Config) timeoutOrDefault() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return 60 * time.Second
}

func (c Config) description() string {
	var segments []string
	if c.Provider != "" {
		segments = append(segments, string(c.Provider))
	}
	if c.Model != "" {
		segments = append(segments, c.Model)
	}
	return strings.Join(segments, "/")
}

func (c Config) mergeHeaders(overrides map[string]string) map[string]string {
	if len(c.Headers) == 0 && len(overrides) == 0 {
		return nil
	}
	result := make(map[string]string, len(c.Headers)+len(overrides))
	for k, v := range c.Headers {
		result[k] = v
	}
	for k, v := range overrides {
		result[k] = v
	}
	return result
}

// RequestFromMessages 构建请求体。
func (c Config) RequestFromMessages(messages []Message, opts ...Option) ChatRequest {
	return ChatRequest{
		Messages: messages,
		Options:  mergeOptions(c.Options, opts...),
	}
}

// ErrProviderNotRegistered 当 provider 未被注册时返回。
var ErrProviderNotRegistered = errors.New("llm provider not registered")

// ErrStreamNotSupported 当 provider 不支持流式时返回。
var ErrStreamNotSupported = errors.New("streaming not supported")

// Adapter 定义统一的推理接口。
type Adapter interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatCompletion, error)
}

// StreamingAdapter 定义可选的流式接口。
type StreamingAdapter interface {
	Stream(ctx context.Context, req ChatRequest) (<-chan ChatStreamChunk, func(), error)
}

// ChatStreamChunk 流式返回片段。
type ChatStreamChunk struct {
	Index        int
	Delta        string
	FinishReason string
	Raw          interface{}
}

// ChatRequest 内部标准化的请求。
type ChatRequest struct {
	Messages []Message
	Options  RequestOptions
}

// Validator 校验请求输入。
type Validator interface {
	Validate() error
}

// validateMessages 确保消息合法。
func validateMessages(messages []Message) error {
	if len(messages) == 0 {
		return errors.New("至少需要一条消息")
	}
	for i, msg := range messages {
		if msg.Role == "" {
			return fmt.Errorf("第 %d 条消息缺少 role", i)
		}
		if strings.TrimSpace(msg.Content) == "" && len(msg.ToolCalls) == 0 {
			return fmt.Errorf("第 %d 条消息 content 不能为空", i)
		}
	}
	return nil
}
