package llm

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

// MessageModifier 在每轮调用模型前修改消息列表。
// 常用于注入 system prompt 或上下文压缩。
type MessageModifier = react.MessageModifier

// AgentConfig 是创建 Agent 的配置。
type AgentConfig struct {
	// ModelConfig 用于通过 NewModel 创建模型实例。
	// 与 Model 二选一，若同时提供则 Model 优先。
	ModelConfig ModelConfig

	// Model 直接传入已创建的模型实例。
	Model model.ToolCallingChatModel

	// Tools 使用 Eino 标准工具接口。
	Tools []tool.BaseTool

	// InvokableTools 使用简化工具接口（内部自动适配为 Eino 标准接口）。
	// 与 Tools 可同时使用，两者合并。
	InvokableTools []InvokableTool

	// SystemPrompt 通过 MessageModifier 在每轮调用前注入。
	// 与 MessageModifier 二选一，若同时提供则 MessageModifier 优先。
	SystemPrompt string

	// MessageModifier 在每轮调用模型前修改消息列表。
	MessageModifier MessageModifier

	// MessageRewriter 持久化修改消息历史（跨轮生效）。
	// 常用于上下文压缩。
	MessageRewriter MessageModifier

	// ToolChoice 控制模型的工具调用行为。
	//   - nil（默认）：模型自行决定是否调用工具
	//   - schema.ToolChoiceForced：强制模型调用工具
	//     首次调用 + 失败重试期间为 Forced，工具成功后自动切换为 Allowed。
	//   - schema.ToolChoiceForbidden：禁止调用工具
	//   - schema.ToolChoiceAllowed：允许但不强制
	ToolChoice *schema.ToolChoice

	// MaxRetries 工具执行失败时的最大重试次数。
	// 仅在 ToolChoice 为 Forced 时生效。
	// 默认 3。
	MaxRetries int

	// MaxStep Agent 最大运行步长。
	// 每次节点转移为一步；一次循环 = ChatModel + Tools = 2 步。
	// 默认 12（最多 5 次工具调用）。
	MaxStep int

	// ToolReturnDirectly 指定某些工具执行后直接返回结果，不再回模型。
	ToolReturnDirectly map[string]struct{}

	// StreamToolCallChecker 流式场景下判断是否包含 tool call。
	// StreamToolCallChecker 流式场景下判断是否包含 tool call。
	// 默认检查第一个 chunk。对于先输出文本再输出 tool call 的模型（如 Claude）需自定义。
	StreamToolCallChecker func(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error)

	// Callbacks 是可选的回调处理器列表，用于监控和日志。
	Callbacks []callbacks.Handler
}

// Agent 封装 Eino ReactAgent，提供简化的高层 API。
type Agent struct {
	inner   *react.Agent
	cfg     AgentConfig
	tracker *toolCallTracker // 用于 ToolChoiceForced + ToolReturnDirectly 场景
}

// NewAgent 创建一个 Agent。
//
// 使用示例：
//
//	// 场景 1: 纯对话
//	agent, _ := llm.NewAgent(ctx, llm.AgentConfig{ModelConfig: cfg})
//	msg, _ := agent.Generate(ctx, messages)
//
//	// 场景 2: 强制调工具 → 结果回模型 → 模型决策
//	forced := schema.ToolChoiceForced
//	agent, _ := llm.NewAgent(ctx, llm.AgentConfig{
//	    ModelConfig:    cfg,
//	    InvokableTools: []llm.InvokableTool{myTool},
//	    ToolChoice:     &forced,
//	})
//
//	// 场景 3: 强制调工具 → 直接拿结果
//	agent, _ := llm.NewAgent(ctx, llm.AgentConfig{
//	    ModelConfig:        cfg,
//	    InvokableTools:     []llm.InvokableTool{myTool},
//	    ToolChoice:         &forced,
//	    ToolReturnDirectly: map[string]struct{}{"my_tool": {}},
//	})
func NewAgent(ctx context.Context, cfg AgentConfig) (*Agent, error) {
	// 1. 创建或使用已有模型
	chatModel := cfg.Model
	if chatModel == nil {
		var err error
		chatModel, err = NewModel(ctx, cfg.ModelConfig)
		if err != nil {
			return nil, fmt.Errorf("create model: %w", err)
		}
	}

	// 2. 合并工具列表：Eino 标准工具 + 适配后的简化工具
	allTools := make([]tool.BaseTool, 0, len(cfg.Tools)+len(cfg.InvokableTools))
	allTools = append(allTools, cfg.Tools...)
	allTools = append(allTools, adaptTools(cfg.InvokableTools)...)

	// 3. 构建 ToolsNodeConfig
	toolsConfig := compose.ToolsNodeConfig{
		Tools: allTools,
	}

	// 4. 处理 ToolChoice + 重试机制
	var tracker *toolCallTracker
	// ReactAgent 的 ToolReturnDirectly 配置
	// 如果开启了 ToolChoiceForced，我们需要接管 ToolReturnDirectly 逻辑，所以传给 ReactAgent 的要清空
	reactToolReturnDirectly := cfg.ToolReturnDirectly

	if cfg.ToolChoice != nil && *cfg.ToolChoice == schema.ToolChoiceForced {
		maxRetries := cfg.MaxRetries
		if maxRetries <= 0 {
			maxRetries = 3
		}
		tracker = &toolCallTracker{maxRetries: maxRetries}
		chatModel = &forcedToolChoiceModel{inner: chatModel, tracker: tracker}
		toolsConfig.ToolCallMiddlewares = append(toolsConfig.ToolCallMiddlewares, newRetryMiddleware(tracker))

		// 如果有 tracker，我们在 Agent 层处理 direct return
		if len(cfg.ToolReturnDirectly) > 0 {
			reactToolReturnDirectly = nil
		}
	}

	// 5. 确定 MessageModifier
	modifier := cfg.MessageModifier
	if modifier == nil && cfg.SystemPrompt != "" {
		modifier = func(_ context.Context, input []*schema.Message) []*schema.Message {
			res := make([]*schema.Message, 0, len(input)+1)
			res = append(res, schema.SystemMessage(cfg.SystemPrompt))
			res = append(res, input...)
			return res
		}
	}

	// 6. 构建 ReactAgent 配置
	reactCfg := &react.AgentConfig{
		ToolCallingModel:      chatModel,
		ToolsConfig:           toolsConfig,
		MessageModifier:       modifier,
		MessageRewriter:       cfg.MessageRewriter,
		MaxStep:               cfg.MaxStep,
		ToolReturnDirectly:    reactToolReturnDirectly,
		StreamToolCallChecker: cfg.StreamToolCallChecker,
	}

	// 7. 创建 ReactAgent
	inner, err := react.NewAgent(ctx, reactCfg)
	if err != nil {
		return nil, fmt.Errorf("create react agent: %w", err)
	}
	return &Agent{inner: inner, cfg: cfg, tracker: tracker}, nil
}

// Generate 非流式调用 Agent。
// 模型会自动处理工具调用循环，直到返回最终答案。
func (a *Agent) Generate(ctx context.Context, messages []*schema.Message, opts ...agent.AgentOption) (*schema.Message, error) {
	// 如果配置了回调，添加到 opts 中
	if len(a.cfg.Callbacks) > 0 {
		opts = append(opts, agent.WithComposeOptions(compose.WithCallbacks(a.cfg.Callbacks...)))
	}

	// 优化：注入 Context Cancel 控制器，以便在工具执行成功后立即中断模型生成（节省 Token）
	var cancel context.CancelFunc
	if a.tracker != nil && len(a.cfg.ToolReturnDirectly) > 0 {
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()

		ctl := &agentControl{
			cancel: cancel,
			shouldReturnDirectly: func(name string) bool {
				_, ok := a.cfg.ToolReturnDirectly[name]
				return ok
			},
		}
		ctx = context.WithValue(ctx, ctxKeyAgentControl{}, ctl)
	}

	msg, err := a.inner.Generate(ctx, messages, opts...)
	if err != nil {
		// 如果是因为我们主动 Cancel 导致的错误，且 Tracker 有成功结果，则视为成功
		if errors.Is(err, context.Canceled) && a.tracker != nil {
			if name, result, ok := a.tracker.getLastSuccess(); ok {
				if _, direct := a.cfg.ToolReturnDirectly[name]; direct {
					return &schema.Message{
						Role:    schema.Assistant,
						Content: result,
					}, nil
				}
			}
		}
		return nil, err
	}

	// 处理 ToolChoiceForced 下的 ToolReturnDirectly (如果在中间件没能及时 Cancel，这里作为兜底)
	// 因为我们禁用了 ReactAgent 的原生支持（为了支持重试），所以需要在这里手动替换结果
	if a.tracker != nil && len(a.cfg.ToolReturnDirectly) > 0 {
		if name, result, ok := a.tracker.getLastSuccess(); ok {
			if _, direct := a.cfg.ToolReturnDirectly[name]; direct {
				// 将工具结果作为最终回复返回
				// 注意：这里我们构造一个 Assistant 消息，内容是工具结果
				return &schema.Message{
					Role:    schema.Assistant,
					Content: result,
				}, nil
			}
		}
	}

	return msg, nil
}

type ctxKeyAgentControl struct{}

type agentControl struct {
	cancel               context.CancelFunc
	shouldReturnDirectly func(name string) bool
}

// Stream 流式调用 Agent。
// 完整支持流式 tool call：模型推理 → 工具调用 → 再推理，全程流式。
func (a *Agent) Stream(ctx context.Context, messages []*schema.Message, opts ...agent.AgentOption) (*schema.StreamReader[*schema.Message], error) {
	// 如果配置了回调，添加到 opts 中
	if len(a.cfg.Callbacks) > 0 {
		opts = append(opts, agent.WithComposeOptions(compose.WithCallbacks(a.cfg.Callbacks...)))
	}

	sr, err := a.inner.Stream(ctx, messages, opts...)
	if err != nil {
		return nil, err
	}

	// 对于 Stream 模式，如果需要 direct return，我们需要包装 StreamReader
	if a.tracker != nil && len(a.cfg.ToolReturnDirectly) > 0 {
		// Stream 模式下暂时不介入 ToolReturnDirectly，让模型总结照常输出。
		// 如果用户真的需要 direct return，推荐使用 Generate 方法。
		return sr, nil
	}

	return sr, nil
}

// ExportGraph 导出底层 Graph，用于嵌入更大的编排图。
func (a *Agent) ExportGraph() (compose.AnyGraph, []compose.GraphAddNodeOpt) {
	return a.inner.ExportGraph()
}

// ToolChoiceForced 是一个便捷变量，避免每次都声明指针。
var ToolChoiceForced = schema.ToolChoiceForced
