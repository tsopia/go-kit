package llm

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type runtimeBuilderResult struct {
	MessageModifier MessageModifier
	MessageRewriter MessageModifier
	Tools           []tool.BaseTool
	Cleanup         func() error
}

func buildPromptAndTools(ctx context.Context, spec RuntimeSpec) (runtimeBuilderResult, error) {
	tools := make([]tool.BaseTool, 0, len(spec.Tools.Standard)+len(spec.Tools.Invokable))
	if !spec.Execution.DisableTools {
		tools = append(tools, spec.Tools.Standard...)
		tools = appendInvokableTools(tools, spec.Tools.Invokable)
	}

	cleanups := make([]func() error, 0, len(spec.Tools.MCPServers))
	for _, server := range spec.Tools.MCPServers {
		if spec.Execution.DisableTools {
			break
		}
		loaded, cleanup, err := mcpToolLoader(ctx, server)
		if err != nil {
			if cleanupErr := runCleanups(cleanups); cleanupErr != nil {
				return runtimeBuilderResult{}, fmt.Errorf("load MCP tools: %w (cleanup: %v)", err, cleanupErr)
			}
			return runtimeBuilderResult{}, fmt.Errorf("load MCP tools: %w", err)
		}
		tools = append(tools, loaded...)
		cleanups = append(cleanups, cleanup)
	}
	if err := validateDirectReturnTools(ctx, spec.Execution.DirectReturnTools, tools); err != nil {
		if cleanupErr := runCleanups(cleanups); cleanupErr != nil {
			return runtimeBuilderResult{}, fmt.Errorf("validate direct return tools: %w (cleanup: %v)", err, cleanupErr)
		}
		return runtimeBuilderResult{}, fmt.Errorf("validate direct return tools: %w", err)
	}

	return runtimeBuilderResult{
		MessageModifier: buildMessageModifier(spec.Prompt),
		MessageRewriter: spec.Prompt.RewriteHistory,
		Tools:           tools,
		Cleanup:         onceCleanup(cleanups),
	}, nil
}

func validateDirectReturnTools(ctx context.Context, configured map[string]struct{}, tools []tool.BaseTool) error {
	if len(configured) == 0 {
		return nil
	}
	known := make(map[string]struct{}, len(tools))
	for _, candidate := range tools {
		info, err := candidate.Info(ctx)
		if err != nil {
			return fmt.Errorf("inspect tool info: %w", err)
		}
		known[info.Name] = struct{}{}
	}
	for name := range configured {
		if _, ok := known[name]; !ok {
			return fmt.Errorf("direct return tool not found: %s", name)
		}
	}
	return nil
}

func buildMessageModifier(cfg PromptConfig) MessageModifier {
	if cfg.System == "" {
		return cfg.PrepareMessages
	}
	return func(ctx context.Context, input []*schema.Message) []*schema.Message {
		res := make([]*schema.Message, 0, len(input)+1)
		res = append(res, schema.SystemMessage(cfg.System))
		res = append(res, input...)
		if cfg.PrepareMessages != nil {
			return cfg.PrepareMessages(ctx, res)
		}
		return res
	}
}

func onceCleanup(cleanups []func() error) func() error {
	var once sync.Once
	var err error
	return func() error {
		once.Do(func() {
			err = runCleanups(cleanups)
		})
		return err
	}
}

func runCleanups(cleanups []func() error) error {
	for _, cleanup := range cleanups {
		if cleanup == nil {
			continue
		}
		if err := cleanup(); err != nil {
			return err
		}
	}
	return nil
}
