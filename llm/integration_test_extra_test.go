package llm

import (
	"context"
	"fmt"
	"strings"
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

// TestReal_StructTool_FullScenario 验证真实模型下的结构化输出提取
// 场景：解析用户复杂的自然语言请求，提取嵌套结构和枚举值
func TestReal_StructTool_FullScenario(t *testing.T) {
	ensureConfig(t)

	// 1. 定义复杂结构体工具
	// 需求：解析 "帮我创建一个订单，用户是张三，25岁，订单金额 100.5 元，标签是 VIP 和 新用户"
	tool := NewStructTool[ComplexStruct]("extract_order", "从自然语言中提取订单和用户信息")

	// 2. 创建 Agent
	forced := schema.ToolChoiceForced
	agent, err := NewAgent(context.Background(), AgentConfig{
		ModelConfig:        IntegrationTestConfig,
		InvokableTools:     []InvokableTool{tool},
		ToolChoice:         &forced,
		ToolReturnDirectly: map[string]struct{}{"extract_order": {}},
		MaxRetries:         3,
		SystemPrompt:       "你是一个订单助手。请根据用户输入提取结构化信息。",
	})
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("\n=== TestReal_StructTool_FullScenario ===")
	input := "帮我创建一个订单，用户是张三，25岁，订单金额 100.5 元，打上 VIP 和 新用户 的标签"
	fmt.Printf("Input: %s\n", input)

	msg, err := agent.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: input},
	})
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Raw Response (JSON): %s\n", msg.Content)

	// 3. 验证结果
	// 注意：真实模型可能并不总是完美遵循 schema（尤其是非常复杂的嵌套），但在简单-中等复杂度下通常表现良好。
	// 这里我们验证核心字段是否正确解析。

	// 使用 StructTool 的 Invoke 来验证（虽然这里已经是结果了，但可以用 tool.Invoke 做一次反序列化检查 + 格式化）
	// 或者直接 json.Unmarshal
	if !strings.Contains(msg.Content, "张三") {
		t.Error("Validation Failed: expected '张三'")
	}
	if !strings.Contains(msg.Content, "CREATE") {
		t.Error("Validation Failed: expected enum 'CREATE'")
	}
	if !strings.Contains(msg.Content, "100.5") {
		t.Error("Validation Failed: expected amount 100.5")
	}
}
