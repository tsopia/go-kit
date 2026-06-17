package llm

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
)

// applyToolDefenses 把 ToolsConfig 的防御配置写入 compose.ToolsNodeConfig。
//
// react（agent.go）与 ADK（adk_agent.go）两条路径都基于 compose.ToolsNodeConfig，
// 因此共用此函数，无需各写一套。仅设置用户显式配置的字段，未配置时零影响。
func applyToolDefenses(tnc *compose.ToolsNodeConfig, cfg ToolsConfig) {
	if len(cfg.Aliases) > 0 {
		aliases := make(map[string]compose.ToolAliasConfig, len(cfg.Aliases))
		for canonical, names := range cfg.Aliases {
			aliases[canonical] = compose.ToolAliasConfig{NameAliases: names}
		}
		tnc.ToolAliases = aliases
	}
	if cfg.UnknownHandler != nil {
		tnc.UnknownToolsHandler = cfg.UnknownHandler
	}
	if cfg.ArgumentsFixer != nil {
		tnc.ToolArgumentsHandler = cfg.ArgumentsFixer
	}
	if cfg.ErrorToText != nil && *cfg.ErrorToText {
		tnc.ToolCallMiddlewares = append(tnc.ToolCallMiddlewares, errorToTextMiddleware())
	}
}

// errorToTextMiddleware 把工具执行错误转换为 ToolResult 文本，避免中断 Agent 流程。
// 若上层（react 结构化日志）在 context 中设置了 toolLogState，会同步标记错误，
// 保证日志仍记录到真实失败。
func errorToTextMiddleware() compose.ToolMiddleware {
	return compose.ToolMiddleware{
		Invokable: func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
			return func(ctx context.Context, input *compose.ToolInput) (out *compose.ToolOutput, retErr error) {
				// 兜 panic：工具实现 panic 时也转为文本，兑现"不中断流程"契约。
				defer func() {
					if r := recover(); r != nil {
						out, retErr = errorToTextResult(ctx, fmt.Errorf("工具 panic: %v", r)), nil
					}
				}()
				output, err := next(ctx, input)
				if err == nil {
					return output, nil
				}
				return errorToTextResult(ctx, err), nil
			}
		},
	}
}

// errorToTextResult 把工具错误转为回传模型的文本结果，并同步标记 toolLogState
// （保证 react 结构化日志仍记录到真实失败）。
func errorToTextResult(ctx context.Context, err error) *compose.ToolOutput {
	if st := toolLogStateFromContext(ctx); st != nil {
		st.markError(err, false, true)
	}
	return &compose.ToolOutput{
		Result: fmt.Sprintf("工具执行失败: %v\n请调整参数或改用其他工具后重试。", err),
	}
}
