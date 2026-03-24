package llm

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/eino/schema"
)

// ── 结构化输出提取测试 ────────────────────────────────────────────────

type ComplexStruct struct {
	UserInfo    UserInfo  `json:"user_info" desc:"用户信息" required:"true"`
	Order       OrderInfo `json:"order" desc:"订单信息" required:"true"`
	RequestType string    `json:"request_type" desc:"请求类型" enum:"QUERY,CREATE,UPDATE,DELETE" required:"true"`
	Tags        []string  `json:"tags" desc:"标签列表"`
}

type UserInfo struct {
	Name string `json:"name" desc:"用户姓名" required:"true"`
	Age  int    `json:"age" desc:"用户年龄"`
}

type OrderInfo struct {
	OrderID string  `json:"order_id" desc:"订单ID"`
	Amount  float64 `json:"amount" desc:"订单金额"`
}

func TestStructTool_FullScenario(t *testing.T) {
	tool := NewStructTool[ComplexStruct]("extract_order", "从自然语言中提取订单和用户信息")
	model := &fakeToolCallingModel{
		responses: []*schema.Message{
			{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{
					{
						ID: "tc1",
						Function: schema.FunctionCall{
							Name:      "extract_order",
							Arguments: `{"user_info":{"name":"张三","age":25},"order":{"order_id":"A001","amount":100.5},"request_type":"CREATE","tags":["VIP","新用户"]}`,
						},
					},
				},
			},
		},
	}

	forced := schema.ToolChoiceForced
	agent, err := NewAgent(context.Background(), AgentConfig{
		Model: AgentModelConfig{Instance: model},
		Tools: ToolsConfig{Invokable: []InvokableTool{tool}},
		Execution: ExecutionConfig{
			ToolChoice:        &forced,
			DirectReturnTools: map[string]struct{}{"extract_order": {}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	msg, err := agent.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "帮我创建一个订单，用户是张三，25岁，订单金额 100.5 元，打上 VIP 和 新用户 的标签"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if model.calls != 1 {
		t.Fatalf("unexpected model call count: got %d want 1", model.calls)
	}

	var got ComplexStruct
	if err := json.Unmarshal([]byte(msg.Content), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if got.UserInfo.Name != "张三" {
		t.Fatalf("unexpected user name: %q", got.UserInfo.Name)
	}
	if got.Order.Amount != 100.5 {
		t.Fatalf("unexpected order amount: %v", got.Order.Amount)
	}
	if got.RequestType != "CREATE" {
		t.Fatalf("unexpected request type: %q", got.RequestType)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "VIP" || got.Tags[1] != "新用户" {
		t.Fatalf("unexpected tags: %#v", got.Tags)
	}
}

func TestStructTool_ExtractionModeScenario(t *testing.T) {
	tool := NewStructTool[ComplexStruct]("extract_order", "从自然语言中提取订单和用户信息")
	model := &fakeToolCallingModel{
		responses: []*schema.Message{
			{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{
					{
						ID: "tc1",
						Function: schema.FunctionCall{
							Name:      "extract_order",
							Arguments: `{request_type:"CREATE"}`,
						},
					},
				},
			},
			{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{
					{
						ID: "tc2",
						Function: schema.FunctionCall{
							Name:      "extract_order",
							Arguments: `{"user_info":{"name":"李四","age":31},"order":{"order_id":"B002","amount":88.8},"request_type":"CREATE","tags":["回头客"]}`,
						},
					},
				},
			},
		},
	}

	agent, err := NewAgent(context.Background(), AgentConfig{
		Model: AgentModelConfig{Instance: model},
		Tools: ToolsConfig{Invokable: []InvokableTool{tool}},
		Execution: ExecutionConfig{
			Mode:              Extraction,
			DirectReturnTools: map[string]struct{}{"extract_order": {}},
		},
		Prompt: PromptConfig{
			System: "把用户请求提取为合法 JSON；如果工具报错，修正参数后重试。",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	msg, err := agent.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "帮我创建订单，用户李四，31岁，订单 B002，金额 88.8 元，标签回头客"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if model.calls != 2 {
		t.Fatalf("unexpected model call count: got %d want 2", model.calls)
	}

	var got ComplexStruct
	if err := json.Unmarshal([]byte(msg.Content), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if got.UserInfo.Name != "李四" {
		t.Fatalf("unexpected user name: %q", got.UserInfo.Name)
	}
	if got.Order.OrderID != "B002" {
		t.Fatalf("unexpected order id: %q", got.Order.OrderID)
	}
	if got.Order.Amount != 88.8 {
		t.Fatalf("unexpected order amount: %v", got.Order.Amount)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "回头客" {
		t.Fatalf("unexpected tags: %#v", got.Tags)
	}
}
