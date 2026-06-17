package llm

import (
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// GenerateOption 是 ADKAgent.Generate / Stream 的运行时选项。
//
// 仅支持「运行时参数调整」（Temperature / MaxTokens / TopP 等），不支持
// 运行时整体替换模型——后者与 ADKAgent「构造时预建 runner」的设计冲突，
// 如需换模型请新建一个 Agent 实例（见 ROADMAP DR / O-004）。
type GenerateOption func(*generateConfig)

type generateConfig struct {
	modelOpts []model.Option
}

// WithTemperature 单次请求覆盖采样温度。
func WithTemperature(t float32) GenerateOption {
	return func(c *generateConfig) { c.modelOpts = append(c.modelOpts, model.WithTemperature(t)) }
}

// WithMaxTokens 单次请求覆盖最大生成 token 数。
func WithMaxTokens(n int) GenerateOption {
	return func(c *generateConfig) { c.modelOpts = append(c.modelOpts, model.WithMaxTokens(n)) }
}

// WithTopP 单次请求覆盖 top-p。
func WithTopP(p float32) GenerateOption {
	return func(c *generateConfig) { c.modelOpts = append(c.modelOpts, model.WithTopP(p)) }
}

// WithModelOptions 透传任意 eino model.Option（高级用法）。
func WithModelOptions(opts ...model.Option) GenerateOption {
	return func(c *generateConfig) { c.modelOpts = append(c.modelOpts, opts...) }
}

func buildGenerateConfig(opts []GenerateOption) *generateConfig {
	c := &generateConfig{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// runOptions 把运行时模型参数转为 adk.AgentRunOption（经 adk.WithChatModelOptions 透传）。
func (c *generateConfig) runOptions() []adk.AgentRunOption {
	if len(c.modelOpts) == 0 {
		return nil
	}
	return []adk.AgentRunOption{adk.WithChatModelOptions(c.modelOpts)}
}
