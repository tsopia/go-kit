package llm

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/filesystem"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
)

// DeepAgentConfig 是 NewDeepAgent 的配置。
//
// DeepAgent 基于 eino adk.DeepAgent，内置任务规划（write_todos）、
// 子 Agent 委派（task）、文件系统工具和 Shell 执行，
// 适合需要多步规划 + 文件操作的复杂任务。
type DeepAgentConfig struct {
	// Model 是底层模型配置。
	Model AgentModelConfig

	// Name 和 Description 标识 Agent，用作子 Agent 时必填。
	Name        string
	Description string

	// Instruction 是 system prompt。为空时使用 DeepAgent 内置默认 prompt
	//（含通用助手行为、安全策略、编码风格、工具使用指南）。
	Instruction string

	// SubAgents 是可被 DeepAgent 委派任务的子 Agent。
	// 通过 ADKAgent.Agent() 获取底层 adk.Agent 传入。
	SubAgents []adk.Agent

	// Tools 是额外工具（除 DeepAgent 内置工具外）。
	Tools ToolsConfig

	// MaxIteration 限制最大推理迭代次数。
	MaxIteration int

	// Backend 设置后启用文件系统工具（read_file/write_file/edit_file/glob/grep）。
	// 可用 filesystem.NewInMemoryBackend() 创建内存实现。
	Backend filesystem.Backend

	// Shell 设置后启用 Shell 执行工具（非流式）。与 StreamingShell 互斥。
	Shell filesystem.Shell

	// StreamingShell 设置后启用流式 Shell 执行。与 Shell 互斥。
	StreamingShell filesystem.StreamingShell

	// WithoutWriteTodos 禁用内置 write_todos 规划工具。
	WithoutWriteTodos bool

	// WithoutGeneralSubAgent 禁用默认通用子 Agent。
	WithoutGeneralSubAgent bool

	// Concurrency 控制并发调用数（与 ADKAgent 相同语义）。
	Concurrency ConcurrencyConfig
}

// NewDeepAgent 创建基于 adk.DeepAgent 的复杂任务 Agent。
//
// DeepAgent 内置任务规划（write_todos）、子 Agent 委派（task）、
// 文件系统操作和 Shell 执行，适合"分析 CSV 并生成图表"这类
// 需要规划 + 多步 + 文件操作的复杂任务。
//
// 返回 *ADKAgent，与 NewADKAgent 返回类型一致，复用 Generate/Stream/Close/Agent API。
// 并发控制通过 Concurrency.MaxConcurrency 配置。
//
// 与 NewADKAgent（ChatModelAgent）的区别：
//   - NewADKAgent：通用工具调用 Agent，你给什么工具用什么
//   - NewDeepAgent：内置规划/子Agent委派/文件系统/Shell 的全家桶
func NewDeepAgent(ctx context.Context, cfg DeepAgentConfig) (*ADKAgent, error) {
	if cfg.Shell != nil && cfg.StreamingShell != nil {
		return nil, errors.New("deep agent: Shell and StreamingShell are mutually exclusive")
	}

	// 创建模型
	var chatModel model.BaseChatModel
	if cfg.Model.Instance != nil {
		chatModel = cfg.Model.Instance
	} else {
		m, err := NewModel(ctx, cfg.Model.Config)
		if err != nil {
			return nil, fmt.Errorf("create model: %w", err)
		}
		chatModel = m
	}

	// 构建额外工具（复用工具加载逻辑，含 MCP）
	spec := RuntimeSpec{Tools: cfg.Tools}
	built, err := buildPromptAndTools(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("build tools: %w", err)
	}

	// 构建 deep.Config
	deepCfg := &deep.Config{
		Name:                   cfg.Name,
		Description:            cfg.Description,
		ChatModel:              chatModel,
		Instruction:            cfg.Instruction,
		ToolsConfig:            adk.ToolsConfig{ToolsNodeConfig: compose.ToolsNodeConfig{Tools: built.Tools}},
		MaxIteration:           cfg.MaxIteration,
		Backend:                cfg.Backend,
		Shell:                  cfg.Shell,
		StreamingShell:         cfg.StreamingShell,
		WithoutWriteTodos:      cfg.WithoutWriteTodos,
		WithoutGeneralSubAgent: cfg.WithoutGeneralSubAgent,
	}
	deepCfg.SubAgents = cfg.SubAgents

	// 创建 DeepAgent
	deepAgent, err := deep.New(ctx, deepCfg)
	if err != nil {
		_ = built.Cleanup()
		return nil, fmt.Errorf("create deep agent: %w", err)
	}

	// 两个 runner 共享同一个 agent
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: deepAgent})
	streamRunner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: deepAgent, EnableStreaming: true})

	return &ADKAgent{
		runner:       runner,
		streamRunner: streamRunner,
		adkAgent:     deepAgent,
		guard:        newConcurrencyGuard(cfg.Concurrency.MaxConcurrency),
		cleanup:      built.Cleanup,
		mode:         Assistant,
		toolCount:    len(built.Tools),
	}, nil
}
