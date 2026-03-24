package llm

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ── ToolAdapter 测试 ───────────────────────────────────────────────

type simpleTool struct {
	info  *schema.ToolInfo
	ret   string
	calls int
}

func (s *simpleTool) Info() *schema.ToolInfo { return s.info }
func (s *simpleTool) Invoke(_ context.Context, _ string) (string, error) {
	s.calls++
	return s.ret, nil
}

func TestToolAdapter_ImplementsInterface(t *testing.T) {
	st := &simpleTool{
		info: &schema.ToolInfo{Name: "test", Desc: "a test tool"},
		ret:  `{"ok":true}`,
	}
	adapter := NewToolAdapter(st)

	var _ tool.InvokableTool = adapter

	info, err := adapter.Info(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "test" {
		t.Fatalf("expected name 'test', got %q", info.Name)
	}

	result, err := adapter.InvokableRun(context.Background(), `{"key":"value"}`)
	if err != nil {
		t.Fatal(err)
	}
	if result != `{"ok":true}` {
		t.Fatalf("expected {\"ok\":true}, got %q", result)
	}
}

func TestAdaptTools(t *testing.T) {
	tools := []InvokableTool{
		&simpleTool{info: &schema.ToolInfo{Name: "a"}, ret: "1"},
		&simpleTool{info: &schema.ToolInfo{Name: "b"}, ret: "2"},
	}
	adapted := adaptTools(tools)
	if len(adapted) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(adapted))
	}
	for i, bt := range adapted {
		info, err := bt.Info(context.Background())
		if err != nil {
			t.Fatalf("tool %d Info error: %v", i, err)
		}
		if info.Name != tools[i].Info().Name {
			t.Fatalf("tool %d name mismatch", i)
		}
	}
}

// ── Agent 构建测试 ─────────────────────────────────────────────────

func TestNewAgent_MissingModel(t *testing.T) {
	_, err := NewAgent(context.Background(), AgentConfig{})
	if err == nil {
		t.Fatal("expected error when no model configured")
	}
}

// fakeToolCallingModel 实现 model.ToolCallingChatModel 接口。
type fakeToolCallingModel struct {
	responses []*schema.Message
	idx       int
	calls     int
	withTools int
}

func (f *fakeToolCallingModel) Generate(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	f.calls++
	if f.idx < len(f.responses) {
		resp := f.responses[f.idx]
		f.idx++
		return resp, nil
	}
	return &schema.Message{Content: "done"}, nil
}

func (f *fakeToolCallingModel) Stream(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	f.calls++
	msg := &schema.Message{Content: "stream done"}
	if f.idx < len(f.responses) {
		msg = f.responses[f.idx]
		f.idx++
	}
	sr, sw := schema.Pipe[*schema.Message](1)
	go func() {
		defer sw.Close()
		sw.Send(msg, nil)
	}()
	return sr, nil
}

func (f *fakeToolCallingModel) WithTools(_ []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	f.withTools++
	return f, nil
}

func TestNewAgent_WithExternalModel_NoTools(t *testing.T) {
	fm := &fakeToolCallingModel{responses: []*schema.Message{{Content: "hello"}}}
	agent, err := NewAgent(context.Background(), AgentConfig{Model: AgentModelConfig{Instance: fm}})
	if err != nil {
		t.Fatalf("NewAgent failed: %v", err)
	}
	if agent == nil {
		t.Fatal("agent should not be nil")
	}
}

func TestNewAgent_WithInvokableTools(t *testing.T) {
	fm := &fakeToolCallingModel{responses: []*schema.Message{{Content: "hello"}}}
	st := &simpleTool{
		info: &schema.ToolInfo{
			Name: "hello",
			Desc: "says hello",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"name": {Type: schema.String, Desc: "name"},
			}),
		},
		ret: `"hello world"`,
	}

	agent, err := NewAgent(context.Background(), AgentConfig{
		Model:  AgentModelConfig{Instance: fm},
		Tools:  ToolsConfig{Invokable: []InvokableTool{st}},
		Prompt: PromptConfig{System: "你是一个助手"},
	})
	if err != nil {
		t.Fatalf("NewAgent failed: %v", err)
	}
	if agent == nil {
		t.Fatal("agent should not be nil")
	}
}

func TestNewAgent_Generate_SimpleChat(t *testing.T) {
	fm := &fakeToolCallingModel{
		responses: []*schema.Message{{Role: schema.Assistant, Content: "I'm fine"}},
	}

	agent, err := NewAgent(context.Background(), AgentConfig{Model: AgentModelConfig{Instance: fm}})
	if err != nil {
		t.Fatal(err)
	}

	msg, err := agent.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "How are you?"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if msg.Content != "I'm fine" {
		t.Fatalf("expected 'I'm fine', got %q", msg.Content)
	}
}

func TestNewAgent_Generate_WithToolCall(t *testing.T) {
	fm := &fakeToolCallingModel{
		responses: []*schema.Message{
			{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{
					{ID: "tc1", Function: schema.FunctionCall{Name: "get_price", Arguments: `{"symbol":"AAPL"}`}},
				},
			},
			{Role: schema.Assistant, Content: "AAPL 的股价是 150.25 USD"},
		},
	}

	priceTool := &simpleTool{
		info: &schema.ToolInfo{
			Name: "get_price",
			Desc: "获取股票价格",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"symbol": {Type: schema.String, Desc: "股票代码", Required: true},
			}),
		},
		ret: `{"price": 150.25, "currency": "USD"}`,
	}

	agent, err := NewAgent(context.Background(), AgentConfig{
		Model: AgentModelConfig{Instance: fm},
		Tools: ToolsConfig{Invokable: []InvokableTool{priceTool}},
	})
	if err != nil {
		t.Fatal(err)
	}

	msg, err := agent.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "查一下 AAPL 的股价"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if msg.Content != "AAPL 的股价是 150.25 USD" {
		t.Fatalf("unexpected response: %q", msg.Content)
	}
}

func TestExecutionModeContracts(t *testing.T) {
	tests := []struct {
		name           string
		cfg            AgentConfig
		wantContent    string
		wantModelCalls int
		wantToolCalls  int
	}{
		{
			name: "conversation_no_tools",
			cfg: AgentConfig{
				Model: AgentModelConfig{Instance: &fakeToolCallingModel{
					responses: []*schema.Message{{Role: schema.Assistant, Content: "plain answer"}},
				}},
			},
			wantContent:    "plain answer",
			wantModelCalls: 1,
		},
		{
			name: "assistant_optional_tools",
			cfg: AgentConfig{
				Model: AgentModelConfig{Instance: &fakeToolCallingModel{
					responses: []*schema.Message{
						{
							Role: schema.Assistant,
							ToolCalls: []schema.ToolCall{
								{ID: "tc1", Function: schema.FunctionCall{Name: "lookup_user", Arguments: `{"name":"Alice"}`}},
							},
						},
						{Role: schema.Assistant, Content: "lookup done"},
					},
				}},
				Tools: ToolsConfig{Invokable: []InvokableTool{
					&simpleTool{
						info: &schema.ToolInfo{
							Name: "lookup_user",
							Desc: "lookup user",
							ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
								"name": {Type: schema.String, Desc: "name"},
							}),
						},
						ret: `{"name":"Alice","age":18}`,
					},
				}},
			},
			wantContent:    "lookup done",
			wantModelCalls: 2,
			wantToolCalls:  1,
		},
		{
			name: "forced_retry_direct_return",
			cfg: AgentConfig{
				Model: AgentModelConfig{Instance: &fakeToolCallingModel{
					responses: []*schema.Message{
						{
							Role: schema.Assistant,
							ToolCalls: []schema.ToolCall{
								{ID: "tc1", Function: schema.FunctionCall{Name: "extract_order", Arguments: `{"title":"bad"}`}},
							},
						},
						{
							Role: schema.Assistant,
							ToolCalls: []schema.ToolCall{
								{ID: "tc2", Function: schema.FunctionCall{Name: "extract_order", Arguments: `{"title":"Go 后端工程师","company":"Acme","requirements":["Go","K8s"],"salary":"30-50K"}`}},
							},
						},
					},
				}},
				Tools: ToolsConfig{Invokable: []InvokableTool{
					&failingTool{
						info: &schema.ToolInfo{
							Name: "extract_order",
							Desc: "extract order",
							ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
								"title": {
									Type: schema.String,
									Desc: "title",
								},
							}),
						},
						failCount: 1,
					},
				}},
				Execution: ExecutionConfig{
					Mode:              Extraction,
					MaxRetries:        2,
					DirectReturnTools: map[string]struct{}{"extract_order": {}},
				},
			},
			wantContent:    `{"result":"success"}`,
			wantModelCalls: 2,
			wantToolCalls:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := NewAgent(context.Background(), tt.cfg)
			if err != nil {
				t.Fatal(err)
			}

			resp, err := agent.Generate(context.Background(), []*schema.Message{
				{Role: schema.User, Content: "run test"},
			})
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}
			if resp.Content != tt.wantContent {
				t.Fatalf("unexpected response: got %q want %q", resp.Content, tt.wantContent)
			}

			if model, ok := tt.cfg.Model.Instance.(*fakeToolCallingModel); ok {
				if model.calls != tt.wantModelCalls {
					t.Fatalf("unexpected model call count: got %d want %d", model.calls, tt.wantModelCalls)
				}
			}

			if tt.wantToolCalls > 0 {
				switch tool := tt.cfg.Tools.Invokable[0].(type) {
				case *simpleTool:
					if tool.calls != tt.wantToolCalls {
						t.Fatalf("unexpected tool call count: got %d want %d", tool.calls, tt.wantToolCalls)
					}
				case *failingTool:
					if tool.calls != tt.wantToolCalls {
						t.Fatalf("unexpected tool call count: got %d want %d", tool.calls, tt.wantToolCalls)
					}
				}
			}
		})
	}
}

func TestNewAgent_ConversationAndAssistantModes(t *testing.T) {
	tests := []struct {
		name              string
		cfg               AgentConfig
		wantContent       string
		wantModelCalls    int
		wantToolCalls     int
		wantWithToolsCall int
	}{
		{
			name: "conversation_does_not_bind_tools",
			cfg: AgentConfig{
				Model: AgentModelConfig{Instance: &fakeToolCallingModel{
					responses: []*schema.Message{{Role: schema.Assistant, Content: "plain answer"}},
				}},
				Tools: ToolsConfig{Invokable: []InvokableTool{
					&simpleTool{
						info: &schema.ToolInfo{
							Name: "lookup_user",
							Desc: "lookup user",
							ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
								"name": {Type: schema.String, Desc: "name"},
							}),
						},
						ret: `{"name":"Alice"}`,
					},
				}},
				Execution: ExecutionConfig{Mode: Conversation},
			},
			wantContent:       "plain answer",
			wantModelCalls:    1,
			wantWithToolsCall: 0,
		},
		{
			name: "assistant_direct_return_uses_bound_tools",
			cfg: AgentConfig{
				Model: AgentModelConfig{Instance: &fakeToolCallingModel{
					responses: []*schema.Message{
						{
							Role: schema.Assistant,
							ToolCalls: []schema.ToolCall{
								{ID: "tc1", Function: schema.FunctionCall{Name: "lookup_user", Arguments: `{"name":"Alice"}`}},
							},
						},
					},
				}},
				Tools: ToolsConfig{Invokable: []InvokableTool{
					&simpleTool{
						info: &schema.ToolInfo{
							Name: "lookup_user",
							Desc: "lookup user",
							ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
								"name": {Type: schema.String, Desc: "name"},
							}),
						},
						ret: `{"name":"Alice","age":18}`,
					},
				}},
				Execution: ExecutionConfig{
					Mode:              Assistant,
					DirectReturnTools: map[string]struct{}{"lookup_user": {}},
				},
			},
			wantContent:       `{"name":"Alice","age":18}`,
			wantModelCalls:    1,
			wantToolCalls:     1,
			wantWithToolsCall: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := NewAgent(context.Background(), tt.cfg)
			if err != nil {
				t.Fatal(err)
			}

			resp, err := agent.Generate(context.Background(), []*schema.Message{
				{Role: schema.User, Content: "run test"},
			})
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}
			if resp.Content != tt.wantContent {
				t.Fatalf("unexpected response: got %q want %q", resp.Content, tt.wantContent)
			}

			model := tt.cfg.Model.Instance.(*fakeToolCallingModel)
			if model.calls != tt.wantModelCalls {
				t.Fatalf("unexpected model call count: got %d want %d", model.calls, tt.wantModelCalls)
			}
			if model.withTools != tt.wantWithToolsCall {
				t.Fatalf("unexpected WithTools count: got %d want %d", model.withTools, tt.wantWithToolsCall)
			}

			if tt.wantToolCalls > 0 {
				tool := tt.cfg.Tools.Invokable[0].(*simpleTool)
				if tool.calls != tt.wantToolCalls {
					t.Fatalf("unexpected tool call count: got %d want %d", tool.calls, tt.wantToolCalls)
				}
			}
		})
	}
}

func TestNewAgent_ExtractionMode(t *testing.T) {
	model := &fakeToolCallingModel{
		responses: []*schema.Message{
			{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{
					{ID: "tc1", Function: schema.FunctionCall{Name: "extract_order", Arguments: `{"title":"bad"}`}},
				},
			},
			{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{
					{ID: "tc2", Function: schema.FunctionCall{Name: "extract_order", Arguments: `{"title":"Go 后端工程师","company":"Acme","requirements":["Go","K8s"],"salary":"30-50K"}`}},
				},
			},
		},
	}
	tool := &failingTool{
		info: &schema.ToolInfo{
			Name: "extract_order",
			Desc: "extract order",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"title": {Type: schema.String, Desc: "title"},
			}),
		},
		failCount: 1,
	}

	agent, err := NewAgent(context.Background(), AgentConfig{
		Model: AgentModelConfig{Instance: model},
		Tools: ToolsConfig{Invokable: []InvokableTool{tool}},
		Execution: ExecutionConfig{
			Mode:              Extraction,
			DirectReturnTools: map[string]struct{}{"extract_order": {}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := agent.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "提取数据"},
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if resp.Content != `{"result":"success"}` {
		t.Fatalf("unexpected response: got %q want %q", resp.Content, `{"result":"success"}`)
	}
	if model.calls != 2 {
		t.Fatalf("unexpected model call count: got %d want 2", model.calls)
	}
	if model.withTools != 1 {
		t.Fatalf("unexpected WithTools count: got %d want 1", model.withTools)
	}
	if tool.calls != 2 {
		t.Fatalf("unexpected tool call count: got %d want 2", tool.calls)
	}
}

// ── 兼容路径测试 ───────────────────────────────────────────────────

func TestNewAgent_ToolChoiceCompatibilityBuild(t *testing.T) {
	fm := &fakeToolCallingModel{
		responses: []*schema.Message{{Content: "hello"}},
	}
	st := &simpleTool{
		info: &schema.ToolInfo{
			Name: "my_tool",
			Desc: "test",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"input": {Type: schema.String, Desc: "input"},
			}),
		},
		ret: "ok",
	}

	forced := schema.ToolChoiceForced
	agent, err := NewAgent(context.Background(), AgentConfig{
		Model: AgentModelConfig{Instance: fm},
		Tools: ToolsConfig{Invokable: []InvokableTool{st}},
		Execution: ExecutionConfig{
			ToolChoice: &forced,
			MaxRetries: 2,
		},
	})
	if err != nil {
		t.Fatalf("NewAgent with legacy ToolChoice failed: %v", err)
	}
	if agent == nil {
		t.Fatal("agent should not be nil")
	}
}

func TestNewAgent_LegacyToolChoiceCompatibilityModes(t *testing.T) {
	tests := []struct {
		name              string
		toolChoice        schema.ToolChoice
		wantContent       string
		wantWithToolsCall int
	}{
		{
			name:              "forbidden_disables_tools",
			toolChoice:        schema.ToolChoiceForbidden,
			wantContent:       "plain answer",
			wantWithToolsCall: 0,
		},
		{
			name:              "allowed_keeps_tools_enabled",
			toolChoice:        schema.ToolChoiceAllowed,
			wantContent:       "plain answer",
			wantWithToolsCall: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			choice := tt.toolChoice
			model := &fakeToolCallingModel{
				responses: []*schema.Message{{Role: schema.Assistant, Content: "plain answer"}},
			}
			agent, err := NewAgent(context.Background(), AgentConfig{
				Model: AgentModelConfig{Instance: model},
				Tools: ToolsConfig{Invokable: []InvokableTool{
					&simpleTool{
						info: &schema.ToolInfo{
							Name: "lookup_user",
							Desc: "lookup user",
							ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
								"name": {Type: schema.String, Desc: "name"},
							}),
						},
						ret: `{"name":"Alice"}`,
					},
				}},
				Execution: ExecutionConfig{ToolChoice: &choice},
			})
			if err != nil {
				t.Fatal(err)
			}

			resp, err := agent.Generate(context.Background(), []*schema.Message{
				{Role: schema.User, Content: "run test"},
			})
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}
			if resp.Content != tt.wantContent {
				t.Fatalf("unexpected response: got %q want %q", resp.Content, tt.wantContent)
			}
			if model.withTools != tt.wantWithToolsCall {
				t.Fatalf("unexpected WithTools count: got %d want %d", model.withTools, tt.wantWithToolsCall)
			}
		})
	}
}

// ── Extraction 内部状态单元测试 ───────────────────────────────────

func TestExtractionState_ShouldForceToolCall(t *testing.T) {
	state := &extractionState{maxRetries: 3}

	// 初始状态：应该强制
	if !state.shouldForceToolCall() {
		t.Fatal("should force initially")
	}

	// 失败1次：仍然强制
	state.recordFailure()
	if !state.shouldForceToolCall() {
		t.Fatal("should force after 1 failure")
	}

	// 失败2次：仍然强制
	state.recordFailure()
	if !state.shouldForceToolCall() {
		t.Fatal("should force after 2 failures")
	}

	// 失败3次：超限，不再强制
	state.recordFailure()
	if state.shouldForceToolCall() {
		t.Fatal("should NOT force after 3 failures (maxRetries reached)")
	}
	if !state.retriesExhausted() {
		t.Fatal("retries should be exhausted")
	}
}

func TestExtractionState_Success(t *testing.T) {
	state := &extractionState{maxRetries: 3}

	// 成功后不再强制
	state.recordSuccess("tool", "result")
	if state.shouldForceToolCall() {
		t.Fatal("should NOT force after success")
	}
	if state.retriesExhausted() {
		t.Fatal("retries should NOT be exhausted after success")
	}
}

func TestExtractionState_FailThenSuccess(t *testing.T) {
	state := &extractionState{maxRetries: 3}

	state.recordFailure()
	if !state.shouldForceToolCall() {
		t.Fatal("should force after 1 failure")
	}

	state.recordSuccess("tool", "result")
	if state.shouldForceToolCall() {
		t.Fatal("should NOT force after success")
	}
}

func TestExtractionRuntime_DirectReturnMessage(t *testing.T) {
	runtime := newExtractionRuntime(3)
	runtime.state.recordSuccess("extract", `{"result":"ok"}`)

	msg, ok := runtime.directReturnMessage(map[string]struct{}{"extract": {}})
	if !ok {
		t.Fatal("expected direct return message")
	}
	if msg.Role != schema.Assistant {
		t.Fatalf("unexpected role: %v", msg.Role)
	}
	if msg.Content != `{"result":"ok"}` {
		t.Fatalf("unexpected content: %q", msg.Content)
	}

	if _, ok := runtime.directReturnMessage(map[string]struct{}{"other": {}}); ok {
		t.Fatal("did not expect direct return for other tool")
	}
}

// ── retryMiddleware 单元测试 ──────────────────────────────────────

type failingTool struct {
	info      *schema.ToolInfo
	failCount int
	calls     int
}

func (f *failingTool) Info() *schema.ToolInfo { return f.info }
func (f *failingTool) Invoke(_ context.Context, _ string) (string, error) {
	f.calls++
	if f.calls <= f.failCount {
		return "", errors.New("tool execution failed")
	}
	return `{"result":"success"}`, nil
}

func TestNewAgent_ExtractionMode_WithRetry(t *testing.T) {
	// 模型第一次被迫调工具 → 工具失败 → 错误回模型 → 模型再调工具 → 工具成功 → 模型总结
	toolInfo := &schema.ToolInfo{
		Name: "extract",
		Desc: "extract data",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {Type: schema.String, Desc: "query"},
		}),
	}

	ft := &failingTool{info: toolInfo, failCount: 1}

	fm := &fakeToolCallingModel{
		responses: []*schema.Message{
			// 第1次调用：模型调工具
			{Role: schema.Assistant, ToolCalls: []schema.ToolCall{
				{ID: "tc1", Function: schema.FunctionCall{Name: "extract", Arguments: `{"query":"test"}`}},
			}},
			// 第2次调用（工具失败后）：模型再调工具
			{Role: schema.Assistant, ToolCalls: []schema.ToolCall{
				{ID: "tc2", Function: schema.FunctionCall{Name: "extract", Arguments: `{"query":"test2"}`}},
			}},
			// 第3次调用（工具成功后）：模型总结
			{Role: schema.Assistant, Content: "提取结果: success"},
		},
	}
	agent, err := NewAgent(context.Background(), AgentConfig{
		Model: AgentModelConfig{Instance: fm},
		Tools: ToolsConfig{Invokable: []InvokableTool{ft}},
		Execution: ExecutionConfig{
			Mode:       Extraction,
			MaxRetries: 3,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	msg, err := agent.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "提取数据"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if ft.calls != 2 {
		t.Fatalf("expected 2 tool calls (1 fail + 1 success), got %d", ft.calls)
	}
	if msg.Content != "提取结果: success" {
		t.Fatalf("unexpected response: %q", msg.Content)
	}
}

func TestNewAgent_ToolReturnDirectly(t *testing.T) {
	toolInfo := &schema.ToolInfo{
		Name: "generate_jd",
		Desc: "生成 JD",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"title": {Type: schema.String, Desc: "职位"},
		}),
	}

	jdTool := &simpleTool{info: toolInfo, ret: `{"title":"Go 工程师","requirements":["Go","K8s"]}`}

	fm := &fakeToolCallingModel{
		responses: []*schema.Message{
			{Role: schema.Assistant, ToolCalls: []schema.ToolCall{
				{ID: "tc1", Function: schema.FunctionCall{Name: "generate_jd", Arguments: `{"title":"Go 工程师"}`}},
			}},
		},
	}

	agent, err := NewAgent(context.Background(), AgentConfig{
		Model: AgentModelConfig{Instance: fm},
		Tools: ToolsConfig{Invokable: []InvokableTool{jdTool}},
		Execution: ExecutionConfig{
			Mode:              Extraction,
			DirectReturnTools: map[string]struct{}{"generate_jd": {}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	msg, err := agent.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "生成 Go 工程师 JD"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// ToolReturnDirectly → msg.Content 应该是工具的原始返回
	if msg.Content != `{"title":"Go 工程师","requirements":["Go","K8s"]}` {
		t.Fatalf("unexpected response (expected tool result directly): %q", msg.Content)
	}
}

// ── StructTool 测试 ──────────────────────────────────────────────

// JobPosting 是用户的实际业务结构体。
type JobPosting struct {
	Title        string   `json:"title" desc:"职位名称" required:"true"`
	Company      string   `json:"company" desc:"公司名称" required:"true"`
	Requirements []string `json:"requirements" desc:"任职要求"`
	Salary       string   `json:"salary" desc:"薪资范围"`
}

func TestStructTool_Info(t *testing.T) {
	st := NewStructTool[JobPosting]("generate_jd", "根据需求生成职位描述")

	info := st.Info()
	if info.Name != "generate_jd" {
		t.Fatalf("expected name 'generate_jd', got %q", info.Name)
	}
	if info.Desc != "根据需求生成职位描述" {
		t.Fatalf("expected desc '根据需求生成职位描述', got %q", info.Desc)
	}
}

func TestStructTool_Invoke_Success(t *testing.T) {
	st := NewStructTool[JobPosting]("generate_jd", "生成 JD")

	validJSON := `{"title":"Go 后端工程师","company":"Acme","requirements":["Go","K8s"],"salary":"30-50K"}`
	result, err := st.Invoke(context.Background(), validJSON)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	// 验证反序列化后的结构
	var jd JobPosting
	if err := json.Unmarshal([]byte(result), &jd); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if jd.Title != "Go 后端工程师" {
		t.Fatalf("expected title 'Go 后端工程师', got %q", jd.Title)
	}
	if jd.Company != "Acme" {
		t.Fatalf("expected company 'Acme', got %q", jd.Company)
	}
}

func TestStructTool_Invoke_InvalidJSON(t *testing.T) {
	st := NewStructTool[JobPosting]("generate_jd", "生成 JD")

	_, err := st.Invoke(context.Background(), `{invalid json}`)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestStructTool_RetryAndDirectReturn 完整模拟用户场景：
//
//	用户：「生成 Go 后端工程师 JD」
//	→ 模型进入 Extraction 模式并先调 generate_jd 工具
//	→ 第1次：模型生成了非法 JSON → 工具 json.Unmarshal 失败 → 自动重试
//	→ 第2次：模型生成了合法 JSON → 工具成功 → 直接返回（ToolReturnDirectly）
//	→ 用户拿到 JobPosting 结构体
func TestStructTool_RetryAndDirectReturn(t *testing.T) {
	st := NewStructTool[JobPosting]("generate_jd", "根据需求生成职位描述")

	fm := &fakeToolCallingModel{
		responses: []*schema.Message{
			// 第1次：模型生成非法 JSON（缺少引号）
			{Role: schema.Assistant, ToolCalls: []schema.ToolCall{
				{ID: "tc1", Function: schema.FunctionCall{
					Name:      "generate_jd",
					Arguments: `{title: Go工程师}`, // 非法 JSON
				}},
			}},
			// 第2次（重试后）：模型生成合法 JSON
			{Role: schema.Assistant, ToolCalls: []schema.ToolCall{
				{ID: "tc2", Function: schema.FunctionCall{
					Name:      "generate_jd",
					Arguments: `{"title":"Go 后端工程师","company":"Acme","requirements":["Go","Docker","K8s"],"salary":"30-50K"}`,
				}},
			}},
		},
	}

	agent, err := NewAgent(context.Background(), AgentConfig{
		Model: AgentModelConfig{Instance: fm},
		Tools: ToolsConfig{Invokable: []InvokableTool{st}},
		Execution: ExecutionConfig{
			Mode:              Extraction,
			MaxRetries:        3,
			DirectReturnTools: map[string]struct{}{"generate_jd": {}},
		},
		Prompt: PromptConfig{System: "根据用户需求生成职位描述，输出必须是合法的 JSON。"},
	})
	if err != nil {
		t.Fatal(err)
	}

	msg, err := agent.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "生成一份 Go 后端工程师的 JD"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// msg.Content 应该是工具直接返回的合法 JSON
	var jd JobPosting
	if err := json.Unmarshal([]byte(msg.Content), &jd); err != nil {
		t.Fatalf("result should be valid JSON, got error: %v\nraw: %s", err, msg.Content)
	}

	if jd.Title != "Go 后端工程师" {
		t.Fatalf("expected title 'Go 后端工程师', got %q", jd.Title)
	}
	if jd.Company != "Acme" {
		t.Fatalf("expected company 'Acme', got %q", jd.Company)
	}
	if len(jd.Requirements) != 3 {
		t.Fatalf("expected 3 requirements, got %d", len(jd.Requirements))
	}
	if jd.Salary != "30-50K" {
		t.Fatalf("expected salary '30-50K', got %q", jd.Salary)
	}

	t.Logf("✅ 结构化输出成功: %+v", jd)
}
