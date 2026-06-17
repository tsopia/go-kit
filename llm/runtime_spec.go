package llm

import (
	"fmt"

	"github.com/cloudwego/eino/schema"
)

type RuntimeSpec struct {
	Model         AgentModelConfig
	Prompt        PromptConfig
	Tools         ToolsConfig
	Execution     RuntimeExecutionSpec
	Streaming     StreamingConfig
	Observability ObservabilityConfig
	Concurrency   ConcurrencyConfig
}

type RuntimeExecutionSpec struct {
	Mode              ExecutionMode
	DisableTools      bool
	ToolChoice        schema.ToolChoice
	RepairMaxAttempts int
	MaxStep           int
	DirectReturnTools map[string]struct{}
}

func compileRuntimeSpec(cfg AgentConfig) (RuntimeSpec, error) {
	mode, err := normalizeExecutionMode(cfg)
	if err != nil {
		return RuntimeSpec{}, err
	}
	if err := validateExecutionConfig(cfg, mode); err != nil {
		return RuntimeSpec{}, err
	}

	spec := RuntimeSpec{
		Model:         cfg.Model,
		Prompt:        cfg.Prompt,
		Tools:         cfg.Tools,
		Streaming:     cfg.Streaming,
		Observability: cfg.Observability,
		Concurrency:   cfg.Concurrency,
		Execution: RuntimeExecutionSpec{
			Mode:              mode,
			MaxStep:           cfg.Execution.MaxStep,
			DirectReturnTools: cloneDirectReturnTools(cfg.Execution.DirectReturnTools),
		},
	}

	switch mode {
	case Conversation:
		spec.Execution.DisableTools = true
		spec.Execution.ToolChoice = schema.ToolChoiceForbidden
	case Assistant:
		spec.Execution.ToolChoice = schema.ToolChoiceAllowed
	case Extraction:
		spec.Execution.ToolChoice = schema.ToolChoiceForced
		spec.Execution.RepairMaxAttempts = cfg.Execution.MaxRetries
		if spec.Execution.RepairMaxAttempts <= 0 {
			spec.Execution.RepairMaxAttempts = 3
		}
	default:
		return RuntimeSpec{}, fmt.Errorf("%w: unsupported execution mode %q", ErrInvalidConfig, mode)
	}

	return spec, nil
}

func normalizeExecutionMode(cfg AgentConfig) (ExecutionMode, error) {
	switch cfg.Execution.Mode {
	case "":
		if cfg.Execution.ToolChoice != nil {
			switch *cfg.Execution.ToolChoice {
			case schema.ToolChoiceForbidden:
				return Conversation, nil
			case schema.ToolChoiceAllowed:
				if !hasConfiguredTools(cfg.Tools) {
					return Conversation, nil
				}
				return Assistant, nil
			case schema.ToolChoiceForced:
				return Extraction, nil
			default:
				return "", fmt.Errorf("%w: unknown tool choice %q", ErrInvalidConfig, *cfg.Execution.ToolChoice)
			}
		}
		if !hasConfiguredTools(cfg.Tools) {
			return Conversation, nil
		}
		return Assistant, nil
	case Conversation, Assistant, Extraction:
		return cfg.Execution.Mode, nil
	default:
		return "", fmt.Errorf("%w: unknown execution mode %q", ErrInvalidConfig, cfg.Execution.Mode)
	}
}

func validateExecutionConfig(cfg AgentConfig, mode ExecutionMode) error {
	switch mode {
	case Conversation:
		if cfg.Execution.Mode == Conversation && hasConfiguredTools(cfg.Tools) {
			return fmt.Errorf("%w: conversation mode does not allow tools", ErrInvalidConfig)
		}
		if cfg.Execution.MaxRetries > 0 {
			return fmt.Errorf("%w: conversation mode does not allow max retries", ErrInvalidConfig)
		}
		if len(cfg.Execution.DirectReturnTools) > 0 {
			return fmt.Errorf("%w: conversation mode does not allow direct return tools", ErrInvalidConfig)
		}
	case Assistant:
		if cfg.Execution.MaxRetries > 0 {
			return fmt.Errorf("%w: assistant mode does not allow max retries", ErrInvalidConfig)
		}
	}
	return nil
}

func hasConfiguredTools(cfg ToolsConfig) bool {
	return len(cfg.Standard) > 0 || len(cfg.Invokable) > 0 || len(cfg.MCPServers) > 0
}

func cloneDirectReturnTools(src map[string]struct{}) map[string]struct{} {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]struct{}, len(src))
	for name := range src {
		dst[name] = struct{}{}
	}
	return dst
}
