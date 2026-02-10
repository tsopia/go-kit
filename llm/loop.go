package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/tsopia/go-kit/llm/model"
	"github.com/tsopia/go-kit/llm/tool"
)

type configProvider interface {
	GetModelConfig() ModelConfig
}

func RunToolCallLoop(ctx context.Context, m model.ToolCallingChatModel, tools []tool.InvokableTool, opts RunOptions) (RunResult, error) {
	boundModel, err := m.WithTools(tools...)
	if err != nil {
		return RunResult{StopReason: STOP_ERROR}, err
	}

	cfg := ModelConfig{}.normalized()
	if p, ok := m.(configProvider); ok {
		cfg = p.GetModelConfig().normalized()
	}

	toolByName := map[string]tool.InvokableTool{}
	allNames := make([]string, 0, len(tools))
	for _, t := range tools {
		name := t.Name()
		toolByName[name] = t
		allNames = append(allNames, name)
	}
	allowedSet, allowedNames := computeAllowedTools(allNames, cfg.ToolCallPolicy.AllowedTools)

	maxRetries := opts.normalizedMaxRetries()
	feedback := make([]model.ChatMessage, 0, maxRetries)
	var lastFailure error

	for retries := 0; retries <= maxRetries; retries++ {
		resp, runErr := boundModel.Generate(ctx, feedback)
		if runErr != nil {
			return RunResult{StopReason: STOP_ERROR}, runErr
		}

		if len(resp.ToolCalls) == 0 {
			switch cfg.ToolCallPolicy.Mode {
			case TOOL_REQUIRED_ONE:
				lastFailure = errors.New("model did not call a tool")
				feedback = append(feedback, model.ChatMessage{Role: "system", Content: requiredOneFeedback(allowedNames)})
				continue
			case TOOL_REQUIRED_EXACT:
				lastFailure = errors.New("model did not call required tool")
				feedback = append(feedback, model.ChatMessage{Role: "system", Content: requiredExactFeedback(cfg.ToolCallPolicy.RequiredToolName)})
				continue
			default:
				return RunResult{FinalText: resp.Content, StopReason: STOP_MODEL_FINAL}, nil
			}
		}

		call := resp.ToolCalls[0]
		if cfg.ToolCallPolicy.Mode == TOOL_REQUIRED_EXACT && call.Name != cfg.ToolCallPolicy.RequiredToolName {
			lastFailure = fmt.Errorf("wrong tool called: %s", call.Name)
			feedback = append(feedback, model.ChatMessage{Role: "system", Content: requiredExactFeedback(cfg.ToolCallPolicy.RequiredToolName)})
			continue
		}

		if _, ok := allowedSet[call.Name]; !ok {
			lastFailure = fmt.Errorf("tool not allowed: %s", call.Name)
			feedback = append(feedback, model.ChatMessage{Role: "system", Content: toolNotAllowedFeedback(call.Name, allowedNames)})
			continue
		}

		t, ok := toolByName[call.Name]
		if !ok {
			lastFailure = fmt.Errorf("tool not found: %s", call.Name)
			feedback = append(feedback, model.ChatMessage{Role: "system", Content: toolNotAllowedFeedback(call.Name, allowedNames)})
			continue
		}

		issues := validateToolArgs(call.Arguments, t.Schema())
		if len(issues) > 0 {
			lastFailure = fmt.Errorf("validation failed for tool %s", call.Name)
			feedback = append(feedback, model.ChatMessage{Role: "system", Content: buildValidationFeedback(call.Name, issues)})
			continue
		}

		toolResultAny, invokeErr := t.Invoke(ctx, call.Arguments)
		if invokeErr != nil {
			return RunResult{StopReason: STOP_ERROR}, invokeErr
		}
		toolResult := mustJSON(toolResultAny)
		result := RunResult{ToolName: call.Name, ToolArgs: call.Arguments, ToolResult: toolResult}

		switch cfg.ToolResultPolicy {
		case RETURN_TOOL_RESULT:
			result.StopReason = STOP_TOOL_RESULT_RETURNED
			return result, nil
		case RETURN_BOTH, RETURN_FINAL_ANSWER:
			feedback = append(feedback,
				model.ChatMessage{Role: "assistant", ToolCalls: []model.ToolCall{call}},
				model.ChatMessage{Role: "tool", Content: string(toolResult)},
			)
			finalResp, finalErr := boundModel.Generate(ctx, feedback)
			if finalErr != nil {
				return RunResult{StopReason: STOP_ERROR}, finalErr
			}
			result.FinalText = finalResp.Content
			result.StopReason = STOP_MODEL_FINAL
			if cfg.ToolResultPolicy == RETURN_FINAL_ANSWER {
				result.ToolResult = nil
			}
			return result, nil
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

func validateToolArgs(args json.RawMessage, schema tool.ArgSchema) []validationIssue {
	if len(schema.Required) == 0 && len(schema.Properties) == 0 && !schema.Strict {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal(args, &payload); err != nil {
		return []validationIssue{{Path: "$", Expected: "object", Actual: "invalid_json", Hint: "arguments must be a JSON object"}}
	}
	issues := make([]validationIssue, 0)
	for _, req := range schema.Required {
		if _, ok := payload[req]; !ok {
			issues = append(issues, validationIssue{Path: fmt.Sprintf("$.%s", req), Expected: "present", Actual: "missing", Hint: fmt.Sprintf("field %s is required", req)})
		}
	}
	for key, expected := range schema.Properties {
		val, ok := payload[key]
		if !ok {
			continue
		}
		actual := detectJSONType(val)
		if expected != tool.JSONTypeUnknown && actual != expected {
			issues = append(issues, validationIssue{Path: fmt.Sprintf("$.%s", key), Expected: string(expected), Actual: string(actual), Hint: fmt.Sprintf("field must be a %s", expected)})
		}
	}
	if schema.Strict {
		for key, val := range payload {
			if _, ok := schema.Properties[key]; !ok {
				issues = append(issues, validationIssue{Path: fmt.Sprintf("$.%s", key), Expected: "defined_property", Actual: string(detectJSONType(val)), Hint: "unknown field is not allowed"})
			}
		}
	}
	return issues
}

func detectJSONType(v any) tool.JSONType {
	switch v.(type) {
	case string:
		return tool.JSONTypeString
	case float64:
		return tool.JSONTypeNumber
	case bool:
		return tool.JSONTypeBool
	case []any:
		return tool.JSONTypeArray
	case map[string]any:
		return tool.JSONTypeObject
	default:
		return tool.JSONTypeUnknown
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

func mustJSON(v any) json.RawMessage {
	if raw, ok := v.(json.RawMessage); ok && json.Valid(raw) {
		return raw
	}
	if b, ok := v.([]byte); ok && json.Valid(b) {
		return b
	}
	if b, err := json.Marshal(v); err == nil && json.Valid(b) {
		return b
	}
	fallback, _ := json.Marshal(fmt.Sprint(v))
	return fallback
}
