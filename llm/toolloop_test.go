package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// ── Fakes ──────────────────────────────────────────────────────────

type fakeModel struct {
	cfg       ModelConfig
	responses []*schema.Message
	idx       int
	history   [][]*schema.Message
}

func (f *fakeModel) WithTools(_ []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return f, nil
}
func (f *fakeModel) Generate(_ context.Context, messages []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	cp := make([]*schema.Message, len(messages))
	copy(cp, messages)
	f.history = append(f.history, cp)
	resp := f.responses[f.idx]
	f.idx++
	return resp, nil
}
func (f *fakeModel) Stream(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, ErrStreamNotSupported
}
func (f *fakeModel) BindTools(_ []*schema.ToolInfo) error { return nil }
func (f *fakeModel) GetModelConfig() ModelConfig          { return f.cfg }

type fakeTool struct {
	info  *schema.ToolInfo
	ret   string
	calls int
}

func (f *fakeTool) Info() *schema.ToolInfo { return f.info }
func (f *fakeTool) Invoke(_ context.Context, _ string) (string, error) {
	f.calls++
	return f.ret, nil
}

func makeToolInfo(name string, required []string, props map[string]string) *schema.ToolInfo {
	params := map[string]*schema.ParameterInfo{}
	for k, v := range props {
		params[k] = &schema.ParameterInfo{
			Type: schema.DataType(v),
			Desc: k,
		}
	}
	for _, r := range required {
		if p, ok := params[r]; ok {
			p.Required = true
		} else {
			params[r] = &schema.ParameterInfo{
				Type:     schema.String,
				Desc:     r,
				Required: true,
			}
		}
	}
	return &schema.ToolInfo{
		Name:        name,
		Desc:        name,
		ParamsOneOf: schema.NewParamsOneOfByParams(params),
	}
}

// ── 测试用例 ───────────────────────────────────────────────────────

func TestRequiredOneNoToolThenToolReturnToolResult(t *testing.T) {
	m := &fakeModel{
		cfg: ModelConfig{
			ToolCallPolicy:   ToolCallPolicy{Mode: TOOL_REQUIRED_ONE},
			ToolResultPolicy: RETURN_TOOL_RESULT,
		},
		responses: []*schema.Message{
			{Content: "no tool"},
			{ToolCalls: []schema.ToolCall{{ID: "tc1", Function: schema.FunctionCall{Name: "sum", Arguments: `{"a":1}`}}}},
		},
	}
	ft := &fakeTool{
		info: makeToolInfo("sum", []string{"a"}, map[string]string{"a": "number"}),
		ret:  `{"ok":true}`,
	}

	msgs := []*schema.Message{{Role: schema.User, Content: "test"}}
	res, err := RunToolCallLoop(context.Background(), m, msgs, []InvokableTool{ft}, RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.StopReason != STOP_TOOL_RESULT_RETURNED || ft.calls != 1 {
		t.Fatalf("unexpected result: %+v calls=%d", res, ft.calls)
	}
	if !strings.Contains(m.history[1][1].Content, "You must call a tool") {
		t.Fatalf("expected required-tool feedback, got: %v", m.history[1])
	}
}

func TestRequiredOneToolNotAllowedRetry(t *testing.T) {
	m := &fakeModel{
		cfg: ModelConfig{
			ToolCallPolicy:   ToolCallPolicy{Mode: TOOL_REQUIRED_ONE, AllowedTools: []string{"good"}},
			ToolResultPolicy: RETURN_TOOL_RESULT,
		},
		responses: []*schema.Message{
			{ToolCalls: []schema.ToolCall{{ID: "tc1", Function: schema.FunctionCall{Name: "bad", Arguments: `{"x":1}`}}}},
			{ToolCalls: []schema.ToolCall{{ID: "tc2", Function: schema.FunctionCall{Name: "good", Arguments: `{"x":1}`}}}},
		},
	}
	good := &fakeTool{info: makeToolInfo("good", nil, nil), ret: `{"v":1}`}
	bad := &fakeTool{info: makeToolInfo("bad", nil, nil), ret: `{"v":2}`}

	msgs := []*schema.Message{{Role: schema.User, Content: "test"}}
	res, err := RunToolCallLoop(context.Background(), m, msgs, []InvokableTool{good, bad}, RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ToolCalls) == 0 || res.ToolCalls[0].Name != "good" || bad.calls != 0 {
		t.Fatalf("unexpected tool selection: %+v badCalls=%d", res, bad.calls)
	}
	if !strings.Contains(m.history[1][1].Content, "Tool bad is not allowed") {
		t.Fatalf("missing not-allowed feedback: %#v", m.history[1])
	}
}

func TestRequiredExactWrongToolThenCorrect(t *testing.T) {
	m := &fakeModel{
		cfg: ModelConfig{
			ToolCallPolicy:   ToolCallPolicy{Mode: TOOL_REQUIRED_EXACT, RequiredToolName: "only"},
			ToolResultPolicy: RETURN_TOOL_RESULT,
		},
		responses: []*schema.Message{
			{ToolCalls: []schema.ToolCall{{ID: "tc1", Function: schema.FunctionCall{Name: "other", Arguments: `{}`}}}},
			{ToolCalls: []schema.ToolCall{{ID: "tc2", Function: schema.FunctionCall{Name: "only", Arguments: `{}`}}}},
		},
	}
	only := &fakeTool{info: makeToolInfo("only", nil, nil), ret: `{"ok":true}`}
	other := &fakeTool{info: makeToolInfo("other", nil, nil), ret: `{"ok":false}`}

	msgs := []*schema.Message{{Role: schema.User, Content: "test"}}
	res, err := RunToolCallLoop(context.Background(), m, msgs, []InvokableTool{only, other}, RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ToolCalls) == 0 || res.ToolCalls[0].Name != "only" {
		t.Fatalf("expected only, got %+v", res)
	}
	if !strings.Contains(m.history[1][1].Content, "You must call tool only") {
		t.Fatalf("expected exact-tool feedback: %#v", m.history[1])
	}
}

func TestValidationFailureStructuredErrorThenCorrectedArgs(t *testing.T) {
	m := &fakeModel{
		cfg: ModelConfig{
			ToolCallPolicy:   ToolCallPolicy{Mode: TOOL_REQUIRED_ONE},
			ToolResultPolicy: RETURN_TOOL_RESULT,
		},
		responses: []*schema.Message{
			{ToolCalls: []schema.ToolCall{{ID: "tc1", Function: schema.FunctionCall{Name: "sum", Arguments: `{"a":"bad"}`}}}},
			{ToolCalls: []schema.ToolCall{{ID: "tc2", Function: schema.FunctionCall{Name: "sum", Arguments: `{"a":2}`}}}},
		},
	}
	ft := &fakeTool{
		info: makeToolInfo("sum", []string{"a"}, map[string]string{"a": "number"}),
		ret:  `{"ok":true}`,
	}

	msgs := []*schema.Message{{Role: schema.User, Content: "test"}}
	_, err := RunToolCallLoop(context.Background(), m, msgs, []InvokableTool{ft}, RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	feedback := m.history[1][1].Content
	if !strings.Contains(feedback, "SCHEMA_VALIDATION_ERROR") || !strings.Contains(feedback, "tool_name") {
		t.Fatalf("unexpected validation feedback: %s", feedback)
	}
	if ft.calls != 1 {
		t.Fatalf("tool should execute once after correction, got %d", ft.calls)
	}
}

func TestReturnFinalAnswer(t *testing.T) {
	m := &fakeModel{
		cfg: ModelConfig{
			ToolCallPolicy:   ToolCallPolicy{Mode: TOOL_OPTIONAL},
			ToolResultPolicy: RETURN_FINAL_ANSWER,
		},
		responses: []*schema.Message{
			{ToolCalls: []schema.ToolCall{{ID: "tc1", Function: schema.FunctionCall{Name: "lookup", Arguments: `{"id":1}`}}}},
			{Content: "final answer"},
		},
	}
	ft := &fakeTool{
		info: makeToolInfo("lookup", nil, nil),
		ret:  `{"id":1,"name":"x"}`,
	}

	msgs := []*schema.Message{{Role: schema.User, Content: "test"}}
	res, err := RunToolCallLoop(context.Background(), m, msgs, []InvokableTool{ft}, RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.FinalText != "final answer" || res.StopReason != STOP_MODEL_FINAL {
		t.Fatalf("unexpected final result: %+v", res)
	}
}

func TestMaxRetriesExceeded(t *testing.T) {
	m := &fakeModel{
		cfg: ModelConfig{
			ToolCallPolicy:   ToolCallPolicy{Mode: TOOL_REQUIRED_ONE},
			ToolResultPolicy: RETURN_TOOL_RESULT,
		},
		responses: []*schema.Message{{}, {}, {}},
	}
	msgs := []*schema.Message{{Role: schema.User, Content: "test"}}
	_, err := RunToolCallLoop(context.Background(), m, msgs, nil, RunOptions{MaxRetries: 1})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMultipleToolCalls(t *testing.T) {
	m := &fakeModel{
		cfg: ModelConfig{
			ToolCallPolicy:   ToolCallPolicy{Mode: TOOL_OPTIONAL},
			ToolResultPolicy: RETURN_TOOL_RESULT,
		},
		responses: []*schema.Message{
			{ToolCalls: []schema.ToolCall{
				{ID: "tc1", Function: schema.FunctionCall{Name: "add", Arguments: `{"a":1,"b":2}`}},
				{ID: "tc2", Function: schema.FunctionCall{Name: "mul", Arguments: `{"a":3,"b":4}`}},
			}},
		},
	}
	add := &fakeTool{info: makeToolInfo("add", nil, nil), ret: `3`}
	mul := &fakeTool{info: makeToolInfo("mul", nil, nil), ret: `12`}

	msgs := []*schema.Message{{Role: schema.User, Content: "test"}}
	res, err := RunToolCallLoop(context.Background(), m, msgs, []InvokableTool{add, mul}, RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(res.ToolCalls))
	}
	if add.calls != 1 || mul.calls != 1 {
		t.Fatalf("expected each tool called once: add=%d mul=%d", add.calls, mul.calls)
	}
}

func TestUserMessagesPassedToModel(t *testing.T) {
	m := &fakeModel{
		cfg: ModelConfig{ToolCallPolicy: ToolCallPolicy{Mode: TOOL_OPTIONAL}},
		responses: []*schema.Message{
			{Content: "I see your question"},
		},
	}

	msgs := []*schema.Message{
		{Role: schema.System, Content: "You are a helper"},
		{Role: schema.User, Content: "What is 1+1?"},
	}
	res, err := RunToolCallLoop(context.Background(), m, msgs, nil, RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.FinalText != "I see your question" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if len(m.history[0]) != 2 {
		t.Fatalf("expected 2 messages passed to model, got %d", len(m.history[0]))
	}
	if m.history[0][0].Content != "You are a helper" || m.history[0][1].Content != "What is 1+1?" {
		t.Fatalf("user messages not passed correctly: %v", m.history[0])
	}
}

func TestMultiRoundToolChain(t *testing.T) {
	m := &fakeModel{
		cfg: ModelConfig{
			ToolCallPolicy:   ToolCallPolicy{Mode: TOOL_OPTIONAL},
			ToolResultPolicy: RETURN_FINAL_ANSWER,
		},
		responses: []*schema.Message{
			{ToolCalls: []schema.ToolCall{{ID: "tc1", Function: schema.FunctionCall{Name: "search", Arguments: `{"q":"go"}`}}}},
			{ToolCalls: []schema.ToolCall{{ID: "tc2", Function: schema.FunctionCall{Name: "summarize", Arguments: `{"text":"Go is..."}`}}}},
			{Content: "Go is a programming language"},
		},
	}
	search := &fakeTool{info: makeToolInfo("search", nil, nil), ret: `{"results":["Go is..."]}`}
	summarize := &fakeTool{info: makeToolInfo("summarize", nil, nil), ret: `"Go is a programming language"`}

	msgs := []*schema.Message{{Role: schema.User, Content: "tell me about Go"}}
	res, err := RunToolCallLoop(context.Background(), m, msgs, []InvokableTool{search, summarize}, RunOptions{MaxRetries: 5})
	if err != nil {
		t.Fatal(err)
	}
	if res.FinalText != "Go is a programming language" || res.StopReason != STOP_MODEL_FINAL {
		t.Fatalf("unexpected result: %+v", res)
	}
	if search.calls != 1 || summarize.calls != 1 {
		t.Fatalf("expected each tool called once: search=%d summarize=%d", search.calls, summarize.calls)
	}
}
