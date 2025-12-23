package llm

import "time"

// RequestOptions 运行时请求参数（可覆盖默认配置）。
type RequestOptions struct {
	Temperature *float64          `json:"temperature,omitempty" yaml:"temperature,omitempty"`
	TopP        *float64          `json:"top_p,omitempty" yaml:"top_p,omitempty"`
	MaxTokens   *int              `json:"max_tokens,omitempty" yaml:"max_tokens,omitempty"`
	Stop        []string          `json:"stop,omitempty" yaml:"stop,omitempty"`
	User        string            `json:"user,omitempty" yaml:"user,omitempty"`
	Stream      bool              `json:"stream,omitempty" yaml:"stream,omitempty"`
	Metadata    map[string]any    `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Timeout     time.Duration     `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Headers     map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	ProviderRaw map[string]any    `json:"provider_raw,omitempty" yaml:"provider_raw,omitempty"`
	Tools       []ToolDefinition  `json:"tools,omitempty" yaml:"tools,omitempty"`
	ToolChoice  string            `json:"tool_choice,omitempty" yaml:"tool_choice,omitempty"` // auto/none/required/具体工具名（OpenAI 风格）
}

// Option 函数式可选项。
type Option func(*RequestOptions)

// WithTemperature 覆盖采样温度。
func WithTemperature(value float64) Option {
	return func(opts *RequestOptions) {
		opts.Temperature = &value
	}
}

// WithTopP 覆盖 top_p。
func WithTopP(value float64) Option {
	return func(opts *RequestOptions) {
		opts.TopP = &value
	}
}

// WithMaxTokens 覆盖最大 tokens。
func WithMaxTokens(value int) Option {
	return func(opts *RequestOptions) {
		opts.MaxTokens = &value
	}
}

// WithStop 覆盖停止词。
func WithStop(stop ...string) Option {
	return func(opts *RequestOptions) {
		opts.Stop = append([]string{}, stop...)
	}
}

// WithUser 传递 user 字段。
func WithUser(user string) Option {
	return func(opts *RequestOptions) {
		opts.User = user
	}
}

// WithStream 开启或关闭流式。
func WithStream(stream bool) Option {
	return func(opts *RequestOptions) {
		opts.Stream = stream
	}
}

// WithTimeout 覆盖单次请求超时。
func WithTimeout(timeout time.Duration) Option {
	return func(opts *RequestOptions) {
		opts.Timeout = timeout
	}
}

// WithHeaders 附加请求头（与默认合并）。
func WithHeaders(headers map[string]string) Option {
	return func(opts *RequestOptions) {
		if len(headers) == 0 {
			return
		}
		if opts.Headers == nil {
			opts.Headers = make(map[string]string, len(headers))
		}
		for k, v := range headers {
			opts.Headers[k] = v
		}
	}
}

// WithProviderRaw 透传给具体 provider 的参数。
func WithProviderRaw(values map[string]any) Option {
	return func(opts *RequestOptions) {
		if len(values) == 0 {
			return
		}
		if opts.ProviderRaw == nil {
			opts.ProviderRaw = make(map[string]any, len(values))
		}
		for k, v := range values {
			opts.ProviderRaw[k] = v
		}
	}
}

// WithTools 设置可用工具列表。
func WithTools(tools []ToolDefinition) Option {
	return func(opts *RequestOptions) {
		opts.Tools = cloneTools(tools)
	}
}

// WithToolChoice 指定工具调用策略（auto/none/required/具体工具名）。
func WithToolChoice(choice string) Option {
	return func(opts *RequestOptions) {
		opts.ToolChoice = choice
	}
}

// mergeOptions 合并默认配置与运行时 Option。
func mergeOptions(base RequestOptions, opts ...Option) RequestOptions {
	merged := RequestOptions{
		Temperature: base.Temperature,
		TopP:        base.TopP,
		MaxTokens:   base.MaxTokens,
		User:        base.User,
		Stop:        append([]string{}, base.Stop...),
		Stream:      base.Stream,
		Metadata:    cloneMap(base.Metadata),
		Headers:     cloneStringMap(base.Headers),
		ProviderRaw: cloneMap(base.ProviderRaw),
		Timeout:     base.Timeout,
		Tools:       cloneTools(base.Tools),
		ToolChoice:  base.ToolChoice,
	}

	for _, opt := range opts {
		opt(&merged)
	}

	return merged
}

func cloneMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneTools(src []ToolDefinition) []ToolDefinition {
	if len(src) == 0 {
		return nil
	}
	dst := make([]ToolDefinition, len(src))
	copy(dst, src)
	return dst
}
