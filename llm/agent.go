package llm

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

// MessageModifier 在每轮调用模型前修改消息列表。
// 常用于注入 system prompt 或上下文压缩。
type MessageModifier = react.MessageModifier

// Agent 封装 Eino ReactAgent，提供简化的高层 API。
type Agent struct {
	inner      *react.Agent
	cfg        AgentConfig
	extraction *extractionRuntime
	cleanup    func() error
}

// NewAgent 创建一个 Agent。
// Mode 是推荐配置入口；如果 Mode 和 ToolChoice 同时设置，以 Mode 为准。
//
// 配置约束：
//   - Conversation 不允许同时配置工具、MaxRetries 或 DirectReturnTools
//   - Assistant 不允许配置 MaxRetries
//   - DirectReturnTools 只能引用已注册的工具名
//
// 使用示例：
//
//	// 场景 1: 纯对话
//	agent, _ := llm.NewAgent(ctx, llm.AgentConfig{Model: llm.AgentModelConfig{Config: cfg}})
//	msg, _ := agent.Generate(ctx, messages)
//
//	// 场景 2: 强制调工具 → 结果回模型 → 模型决策
//	agent, _ := llm.NewAgent(ctx, llm.AgentConfig{
//	    Model: llm.AgentModelConfig{Config: cfg},
//	    Tools: llm.ToolsConfig{Invokable: []llm.InvokableTool{myTool}},
//	    Execution: llm.ExecutionConfig{Mode: llm.Extraction},
//	})
//
//	// 场景 3: 强制调工具 → 直接拿结果
//	agent, _ := llm.NewAgent(ctx, llm.AgentConfig{
//	    Model: llm.AgentModelConfig{Config: cfg},
//	    Tools: llm.ToolsConfig{Invokable: []llm.InvokableTool{myTool}},
//	    Execution: llm.ExecutionConfig{
//	        Mode:              llm.Extraction,
//	        DirectReturnTools: map[string]struct{}{"my_tool": {}},
//	    },
//	})
func NewAgent(ctx context.Context, cfg AgentConfig) (*Agent, error) {
	spec, err := compileRuntimeSpec(cfg)
	if err != nil {
		return nil, fmt.Errorf("compile runtime spec: %w", err)
	}

	built, err := buildPromptAndTools(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("build prompt and tools: %w", err)
	}

	// 1. 创建或使用已有模型
	chatModel := spec.Model.Instance
	if chatModel == nil {
		chatModel, err = NewModel(ctx, spec.Model.Config)
		if err != nil {
			_ = built.Cleanup()
			return nil, fmt.Errorf("create model: %w", err)
		}
	}

	// 2. 合并工具列表：Eino 标准工具 + 适配后的简化工具
	toolsConfig := compose.ToolsNodeConfig{
		Tools: built.Tools,
	}

	// 4. 处理 Extraction 模式下的强制工具调用和修复机制
	var extraction *extractionRuntime
	// ReactAgent 的 ToolReturnDirectly 配置。
	// Extraction 运行时需要接管 direct return，避免和修复链路冲突。
	reactToolReturnDirectly := spec.Execution.DirectReturnTools

	if spec.Execution.ToolChoice == schema.ToolChoiceForced {
		maxRetries := spec.Execution.RepairMaxAttempts
		extraction = newExtractionRuntime(maxRetries)
		chatModel = extraction.wrapModel(chatModel)
		toolsConfig.ToolCallMiddlewares = append(toolsConfig.ToolCallMiddlewares, extraction.middleware())

		// 如果有 extraction runtime，我们在 Agent 层处理 direct return
		if len(spec.Execution.DirectReturnTools) > 0 {
			reactToolReturnDirectly = nil
		}
	}

	// 6. 构建 ReactAgent 配置
	reactCfg := &react.AgentConfig{
		ToolCallingModel:      chatModel,
		ToolsConfig:           toolsConfig,
		MessageModifier:       built.MessageModifier,
		MessageRewriter:       built.MessageRewriter,
		MaxStep:               spec.Execution.MaxStep,
		ToolReturnDirectly:    reactToolReturnDirectly,
		StreamToolCallChecker: spec.Streaming.ToolCallChecker,
	}

	// 7. 创建 ReactAgent
	inner, err := react.NewAgent(ctx, reactCfg)
	if err != nil {
		_ = built.Cleanup()
		return nil, fmt.Errorf("create react agent: %w", err)
	}
	return &Agent{inner: inner, cfg: cfg, extraction: extraction, cleanup: built.Cleanup}, nil
}

// Generate 非流式调用 Agent。
// 模型会自动处理工具调用循环，直到返回最终答案。
func (a *Agent) Generate(ctx context.Context, messages []*schema.Message, opts ...agent.AgentOption) (*schema.Message, error) {
	// 如果配置了回调，添加到 opts 中
	if len(a.cfg.Observability.Callbacks) > 0 {
		opts = append(opts, agent.WithComposeOptions(compose.WithCallbacks(a.cfg.Observability.Callbacks...)))
	}

	// 优化：注入 Context Cancel 控制器，以便在工具执行成功后立即中断模型生成（节省 Token）
	var cancel context.CancelFunc
	if a.extraction != nil && len(a.cfg.Execution.DirectReturnTools) > 0 {
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()

		ctl := &agentControl{
			cancel: cancel,
			shouldReturnDirectly: func(name string) bool {
				_, ok := a.cfg.Execution.DirectReturnTools[name]
				return ok
			},
		}
		ctx = context.WithValue(ctx, ctxKeyAgentControl{}, ctl)
	}

	msg, err := a.inner.Generate(ctx, messages, opts...)
	if err != nil {
		// 如果是因为我们主动 Cancel 导致的错误，且 Extraction 有成功结果，则视为成功
		if errors.Is(err, context.Canceled) && a.extraction != nil {
			if msg, ok := a.directReturnMessage(); ok {
				return msg, nil
			}
		}
		return nil, err
	}

	// 处理 Extraction 模式下的 ToolReturnDirectly (如果在中间件没能及时 Cancel，这里作为兜底)
	// 因为我们禁用了 ReactAgent 的原生支持（为了支持重试），所以需要在这里手动替换结果
	if msg, ok := a.directReturnMessage(); ok {
		return msg, nil
	}

	return msg, nil
}

func (a *Agent) directReturnMessage() (*schema.Message, bool) {
	if a.extraction == nil {
		return nil, false
	}
	if len(a.cfg.Execution.DirectReturnTools) == 0 {
		return nil, false
	}
	return a.extraction.directReturnMessage(a.cfg.Execution.DirectReturnTools)
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
	if len(a.cfg.Observability.Callbacks) > 0 {
		opts = append(opts, agent.WithComposeOptions(compose.WithCallbacks(a.cfg.Observability.Callbacks...)))
	}

	sr, err := a.inner.Stream(ctx, messages, opts...)
	if err != nil {
		return nil, err
	}

	// 对于 Stream 模式，如果需要 direct return，我们需要包装 StreamReader
	if a.extraction != nil && len(a.cfg.Execution.DirectReturnTools) > 0 {
		// Stream 模式下暂时不介入 ToolReturnDirectly，让模型总结照常输出。
		// 如果用户真的需要 direct return，推荐使用 Generate 方法。
		return sr, nil
	}

	return sr, nil
}

func (a *Agent) Close() error {
	if a.cleanup == nil {
		return nil
	}
	return a.cleanup()
}

// ExportGraph 导出底层 Graph，用于嵌入更大的编排图。
func (a *Agent) ExportGraph() (compose.AnyGraph, []compose.GraphAddNodeOpt) {
	return a.inner.ExportGraph()
}

// Deprecated: 新代码应优先使用 AgentConfig.Execution.Mode。
// ToolChoiceForced 仅保留给旧配置的兼容路径。
var ToolChoiceForced = schema.ToolChoiceForced
