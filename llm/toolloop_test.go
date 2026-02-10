package llm

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/tsopia/go-kit/llm/model"
	"github.com/tsopia/go-kit/llm/tool"
)

type fakeModel struct {
	cfg       ModelConfig
	responses []model.ChatMessage
	idx       int
	history   [][]model.ChatMessage
}

func (f *fakeModel) WithTools(_ ...tool.InvokableTool) (model.ToolCallingChatModel, error) {
	return f, nil
}
func (f *fakeModel) Generate(_ context.Context, messages []model.ChatMessage) (model.ChatMessage, error) {
	cp := append([]model.ChatMessage(nil), messages...)
	f.history = append(f.history, cp)
	resp := f.responses[f.idx]
	f.idx++
	return resp, nil
}
func (f *fakeModel) GetModelConfig() ModelConfig { return f.cfg }

type fakeTool struct {
	name   string
	schema tool.ArgSchema
	ret    any
	calls  int
}

func (f *fakeTool) Name() string           { return f.name }
func (f *fakeTool) Schema() tool.ArgSchema { return f.schema }
func (f *fakeTool) Invoke(_ context.Context, _ json.RawMessage) (any, error) {
	f.calls++
	return f.ret, nil
}

func TestRequiredOneNoToolThenToolReturnToolResult(t *testing.T) {
	m := &fakeModel{cfg: ModelConfig{ToolCallPolicy: ToolCallPolicy{Mode: TOOL_REQUIRED_ONE}, ToolResultPolicy: RETURN_TOOL_RESULT}, responses: []model.ChatMessage{
		{Content: "no tool"},
		{ToolCalls: []model.ToolCall{{Name: "sum", Arguments: json.RawMessage(`{"a":1}`)}}},
	}}
	ft := &fakeTool{name: "sum", schema: tool.ArgSchema{Required: []string{"a"}, Properties: map[string]tool.JSONType{"a": tool.JSONTypeNumber}}, ret: map[string]any{"ok": true}}

	res, err := RunToolCallLoop(context.Background(), m, []tool.InvokableTool{ft}, RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.StopReason != STOP_TOOL_RESULT_RETURNED || ft.calls != 1 {
		t.Fatalf("unexpected result: %+v calls=%d", res, ft.calls)
	}
	if !strings.Contains(m.history[1][0].Content, "You must call a tool") {
		t.Fatalf("expected required-tool feedback, got: %v", m.history[1])
	}
}

func TestRequiredOneToolNotAllowedRetry(t *testing.T) {
	m := &fakeModel{cfg: ModelConfig{ToolCallPolicy: ToolCallPolicy{Mode: TOOL_REQUIRED_ONE, AllowedTools: []string{"good"}}, ToolResultPolicy: RETURN_TOOL_RESULT}, responses: []model.ChatMessage{
		{ToolCalls: []model.ToolCall{{Name: "bad", Arguments: json.RawMessage(`{"x":1}`)}}},
		{ToolCalls: []model.ToolCall{{Name: "good", Arguments: json.RawMessage(`{"x":1}`)}}},
	}}
	good := &fakeTool{name: "good", ret: map[string]any{"v": 1}}
	bad := &fakeTool{name: "bad", ret: map[string]any{"v": 2}}

	res, err := RunToolCallLoop(context.Background(), m, []tool.InvokableTool{good, bad}, RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.ToolName != "good" || bad.calls != 0 {
		t.Fatalf("unexpected tool selection: %+v badCalls=%d", res, bad.calls)
	}
	if !strings.Contains(m.history[1][0].Content, "Tool bad is not allowed") {
		t.Fatalf("missing not-allowed feedback: %#v", m.history[1])
	}
}

func TestRequiredExactWrongToolThenCorrect(t *testing.T) {
	m := &fakeModel{cfg: ModelConfig{ToolCallPolicy: ToolCallPolicy{Mode: TOOL_REQUIRED_EXACT, RequiredToolName: "only"}, ToolResultPolicy: RETURN_TOOL_RESULT}, responses: []model.ChatMessage{
		{ToolCalls: []model.ToolCall{{Name: "other", Arguments: json.RawMessage(`{}`)}}},
		{ToolCalls: []model.ToolCall{{Name: "only", Arguments: json.RawMessage(`{}`)}}},
	}}
	only := &fakeTool{name: "only", ret: map[string]any{"ok": true}}
	other := &fakeTool{name: "other", ret: map[string]any{"ok": false}}

	res, err := RunToolCallLoop(context.Background(), m, []tool.InvokableTool{only, other}, RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.ToolName != "only" {
		t.Fatalf("expected only, got %+v", res)
	}
	if !strings.Contains(m.history[1][0].Content, "You must call tool only") {
		t.Fatalf("expected exact-tool feedback: %#v", m.history[1])
	}
}

func TestValidationFailureStructuredErrorThenCorrectedArgs(t *testing.T) {
	m := &fakeModel{cfg: ModelConfig{ToolCallPolicy: ToolCallPolicy{Mode: TOOL_REQUIRED_ONE}, ToolResultPolicy: RETURN_TOOL_RESULT}, responses: []model.ChatMessage{
		{ToolCalls: []model.ToolCall{{Name: "sum", Arguments: json.RawMessage(`{"a":"bad"}`)}}},
		{ToolCalls: []model.ToolCall{{Name: "sum", Arguments: json.RawMessage(`{"a":2}`)}}},
	}}
	ft := &fakeTool{name: "sum", schema: tool.ArgSchema{Required: []string{"a"}, Properties: map[string]tool.JSONType{"a": tool.JSONTypeNumber}}, ret: map[string]any{"ok": true}}

	_, err := RunToolCallLoop(context.Background(), m, []tool.InvokableTool{ft}, RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	feedback := m.history[1][0].Content
	if !strings.Contains(feedback, "SCHEMA_VALIDATION_ERROR") || !strings.Contains(feedback, "tool_name") {
		t.Fatalf("unexpected validation feedback: %s", feedback)
	}
	if ft.calls != 1 {
		t.Fatalf("tool should execute once after correction, got %d", ft.calls)
	}
}

func TestReturnFinalAnswer(t *testing.T) {
	m := &fakeModel{cfg: ModelConfig{ToolCallPolicy: ToolCallPolicy{Mode: TOOL_REQUIRED_ONE}, ToolResultPolicy: RETURN_FINAL_ANSWER}, responses: []model.ChatMessage{
		{ToolCalls: []model.ToolCall{{Name: "lookup", Arguments: json.RawMessage(`{"id":1}`)}}},
		{Content: "final answer"},
	}}
	ft := &fakeTool{name: "lookup", ret: map[string]any{"id": 1, "name": "x"}}

	res, err := RunToolCallLoop(context.Background(), m, []tool.InvokableTool{ft}, RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.FinalText != "final answer" || res.StopReason != STOP_MODEL_FINAL {
		t.Fatalf("unexpected final result: %+v", res)
	}
	if len(res.ToolResult) != 0 {
		t.Fatalf("tool result must be omitted for RETURN_FINAL_ANSWER: %s", string(res.ToolResult))
	}
}

func TestMaxRetriesExceeded(t *testing.T) {
	m := &fakeModel{cfg: ModelConfig{ToolCallPolicy: ToolCallPolicy{Mode: TOOL_REQUIRED_ONE}, ToolResultPolicy: RETURN_TOOL_RESULT}, responses: []model.ChatMessage{{}, {}, {}}}
	_, err := RunToolCallLoop(context.Background(), m, nil, RunOptions{MaxRetries: 1})
	if err == nil {
		t.Fatal("expected error")
	}
}
