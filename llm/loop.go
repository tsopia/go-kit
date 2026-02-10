package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// RunToolCallLoop 执行工具调用循环。
//
// messages 是用户上下文（system/user/history），tools 是可执行工具集合。
// loop 会根据 ToolCallPolicy 驱动模型调用工具，根据 ToolResultPolicy 决定返回策略。
// 支持多个 tool calls（parallel function calling）和多轮工具调用链。
func RunToolCallLoop(
	ctx context.Context,
	m model.ToolCallingChatModel,
	messages []*schema.Message,
	tools []InvokableTool,
	opts RunOptions,
) (RunResult, error) {
	// 提取 ToolInfo 并绑定到模型
	toolInfos := make([]*schema.ToolInfo, len(tools))
	for i, t := range tools {
		toolInfos[i] = t.Info()
	}
	boundModel, err := m.WithTools(toolInfos)
	if err != nil {
		return RunResult{StopReason: STOP_ERROR}, err
	}

	cfg := ModelConfig{}.normalized()
	if p, ok := m.(configProvider); ok {
		cfg = p.GetModelConfig().normalized()
	}

	toolByName := map[string]InvokableTool{}
	allNames := make([]string, 0, len(tools))
	for _, t := range tools {
		name := t.Info().Name
		toolByName[name] = t
		allNames = append(allNames, name)
	}
	allowedSet, allowedNames := computeAllowedTools(allNames, cfg.ToolCallPolicy.AllowedTools)

	maxRetries := opts.normalizedMaxRetries()

	// 构建完整的对话历史：用户原始 messages + loop 过程中的 feedback
	history := make([]*schema.Message, len(messages))
	copy(history, messages)

	var lastFailure error

	for retries := 0; retries <= maxRetries; retries++ {
		resp, runErr := boundModel.Generate(ctx, history)
		if runErr != nil {
			return RunResult{StopReason: STOP_ERROR}, runErr
		}

		// ── 无 tool call：按策略处理 ──
		if len(resp.ToolCalls) == 0 {
			switch cfg.ToolCallPolicy.Mode {
			case TOOL_REQUIRED_ONE:
				lastFailure = errors.New("model did not call a tool")
				history = append(history, &schema.Message{Role: schema.System, Content: requiredOneFeedback(allowedNames)})
				continue
			case TOOL_REQUIRED_EXACT:
				lastFailure = errors.New("model did not call required tool")
				history = append(history, &schema.Message{Role: schema.System, Content: requiredExactFeedback(cfg.ToolCallPolicy.RequiredToolName)})
				continue
			default: // TOOL_OPTIONAL
				return RunResult{FinalText: resp.Content, StopReason: STOP_MODEL_FINAL}, nil
			}
		}

		// ── 有 tool calls：校验 + 执行 ──
		allValid := true
		var toolResults []ToolCallResult
		var feedbackMsgs []*schema.Message

		// 先把 assistant 的 tool call 消息加入历史
		feedbackMsgs = append(feedbackMsgs, resp)

		for _, tc := range resp.ToolCalls {
			callName := tc.Function.Name
			callArgs := tc.Function.Arguments
			callID := tc.ID

			// 检查 TOOL_REQUIRED_EXACT
			if cfg.ToolCallPolicy.Mode == TOOL_REQUIRED_EXACT && callName != cfg.ToolCallPolicy.RequiredToolName {
				lastFailure = fmt.Errorf("wrong tool called: %s", callName)
				history = append(history, &schema.Message{Role: schema.System, Content: requiredExactFeedback(cfg.ToolCallPolicy.RequiredToolName)})
				allValid = false
				break
			}

			// 检查是否在允许列表中
			if _, ok := allowedSet[callName]; !ok {
				lastFailure = fmt.Errorf("tool not allowed: %s", callName)
				history = append(history, &schema.Message{Role: schema.System, Content: toolNotAllowedFeedback(callName, allowedNames)})
				allValid = false
				break
			}

			// 查找工具
			t, ok := toolByName[callName]
			if !ok {
				lastFailure = fmt.Errorf("tool not found: %s", callName)
				history = append(history, &schema.Message{Role: schema.System, Content: toolNotAllowedFeedback(callName, allowedNames)})
				allValid = false
				break
			}

			// 参数校验
			issues := validateToolArgs(callArgs, t.Info())
			if len(issues) > 0 {
				lastFailure = fmt.Errorf("validation failed for tool %s", callName)
				history = append(history, &schema.Message{Role: schema.System, Content: buildValidationFeedback(callName, issues)})
				allValid = false
				break
			}

			// 执行工具
			result, invokeErr := t.Invoke(ctx, callArgs)
			if invokeErr != nil {
				return RunResult{StopReason: STOP_ERROR}, invokeErr
			}

			toolResults = append(toolResults, ToolCallResult{
				ID: callID, Name: callName, Args: callArgs, Result: result,
			})

			// 构建 tool result 消息
			feedbackMsgs = append(feedbackMsgs, &schema.Message{
				Role:       schema.Tool,
				Content:    result,
				ToolCallID: callID,
			})
		}

		if !allValid {
			continue
		}

		// ── 根据 ToolResultPolicy 决定返回 ──
		result := RunResult{ToolCalls: toolResults}

		switch cfg.ToolResultPolicy {
		case RETURN_TOOL_RESULT:
			result.StopReason = STOP_TOOL_RESULT_RETURNED
			return result, nil

		case RETURN_BOTH, RETURN_FINAL_ANSWER:
			// 把 assistant tool call + tool results 加入历史，继续 loop
			history = append(history, feedbackMsgs...)

			// 让模型基于 tool results 产生最终答案（或再次调用工具 — 继续循环）
			continue

		default:
			result.StopReason = STOP_TOOL_RESULT_RETURNED
			return result, nil
		}
	}

	if lastFailure == nil {
		lastFailure = errors.New("tool call retry limit reached")
	}
	return RunResult{StopReason: STOP_MAX_RETRIES}, fmt.Errorf("max retries exceeded: %w", lastFailure)
}

func computeAllowedTools(allTools []string, configured []string) (map[string]struct{}, []string) {
	set := map[string]struct{}{}
	if len(configured) == 0 {
		for _, name := range allTools {
			set[name] = struct{}{}
		}
		return set, append([]string(nil), allTools...)
	}
	configuredSet := map[string]struct{}{}
	for _, name := range configured {
		configuredSet[name] = struct{}{}
	}
	allowed := make([]string, 0, len(allTools))
	for _, name := range allTools {
		if _, ok := configuredSet[name]; ok {
			set[name] = struct{}{}
			allowed = append(allowed, name)
		}
	}
	return set, allowed
}

func requiredOneFeedback(allowed []string) string {
	return fmt.Sprintf("You must call a tool to complete the task. Choose one from: %s. Do not output natural language; output only the tool call.", strings.Join(allowed, ", "))
}

func toolNotAllowedFeedback(toolName string, allowed []string) string {
	return fmt.Sprintf("Tool %s is not allowed. Choose one from: %s and call it.", toolName, strings.Join(allowed, ", "))
}

func requiredExactFeedback(requiredTool string) string {
	return fmt.Sprintf("You must call tool %s. Do not output natural language; output only the tool call.", requiredTool)
}

type validationIssue struct {
	Path     string `json:"path"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Hint     string `json:"hint"`
}

type validationFailure struct {
	ToolName    string            `json:"tool_name"`
	ErrorType   string            `json:"error_type"`
	Message     string            `json:"message"`
	Issues      []validationIssue `json:"issues"`
	Instruction string            `json:"instruction"`
}

// validateToolArgs 对工具参数进行轻量校验。
func validateToolArgs(argsStr string, info *schema.ToolInfo) []validationIssue {
	if info == nil || info.ParamsOneOf == nil {
		return nil
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(argsStr), &payload); err != nil {
		return []validationIssue{{Path: "$", Expected: "object", Actual: "invalid_json", Hint: "arguments must be a JSON object"}}
	}

	// 通过 ToJSONSchema 获取 schema 定义
	jsSchema, err := info.ParamsOneOf.ToJSONSchema()
	if err != nil || jsSchema == nil {
		return nil
	}

	issues := make([]validationIssue, 0)

	// 检查 required 字段
	for _, fieldName := range jsSchema.Required {
		if _, exists := payload[fieldName]; !exists {
			issues = append(issues, validationIssue{
				Path: fmt.Sprintf("$.%s", fieldName), Expected: "present", Actual: "missing",
				Hint: fmt.Sprintf("field %s is required", fieldName),
			})
		}
	}

	// 检查 properties 类型
	if jsSchema.Properties != nil {
		for pair := jsSchema.Properties.Oldest(); pair != nil; pair = pair.Next() {
			key := pair.Key
			propSchema := pair.Value
			val, ok := payload[key]
			if !ok || propSchema == nil {
				continue
			}
			expectedType := propSchema.Type
			if expectedType == "" {
				continue
			}
			actualType := detectJSONType(val)
			if expectedType != actualType {
				issues = append(issues, validationIssue{
					Path: fmt.Sprintf("$.%s", key), Expected: expectedType, Actual: actualType,
					Hint: fmt.Sprintf("field must be a %s", expectedType),
				})
			}
		}
	}

	return issues
}

func detectJSONType(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return "unknown"
	}
}

func buildValidationFeedback(toolName string, issues []validationIssue) string {
	payload := validationFailure{
		ToolName:    toolName,
		ErrorType:   "SCHEMA_VALIDATION_ERROR",
		Message:     "arguments are invalid",
		Issues:      issues,
		Instruction: "Please call the same tool again with a complete corrected arguments object. Do not output natural language.",
	}
	b, _ := json.Marshal(payload)
	return string(b)
}
