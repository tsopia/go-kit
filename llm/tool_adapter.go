package llm

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ToolAdapter 将 llm.InvokableTool 适配为 Eino tool.InvokableTool。
// 用于桥接简化接口与 Eino 标准接口。
type ToolAdapter struct {
	inner InvokableTool
}

// NewToolAdapter 创建一个工具适配器。
func NewToolAdapter(t InvokableTool) *ToolAdapter {
	return &ToolAdapter{inner: t}
}

func (a *ToolAdapter) Info(_ context.Context) (*schema.ToolInfo, error) {
	return a.inner.Info(), nil
}

func (a *ToolAdapter) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	return a.inner.Invoke(ctx, argumentsInJSON)
}

// adaptTools 将 InvokableTool 切片适配为 Eino tool.BaseTool 切片。
func adaptTools(tools []InvokableTool) []tool.BaseTool {
	adapted := make([]tool.BaseTool, len(tools))
	for i, t := range tools {
		adapted[i] = NewToolAdapter(t)
	}
	return adapted
}
