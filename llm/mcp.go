package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/tool"

	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// MCPProtocol 定义 MCP 的传输协议。
type MCPProtocol string

const (
	MCPProtocolStdio MCPProtocol = "stdio"
	MCPProtocolSSE   MCPProtocol = "sse"
)

// MCPConfig 定义单个 MCP 服务器连接的配置。
type MCPConfig struct {
	Name    string // 此客户端的标识符
	Version string // 客户端版本，默认为 1.0.0

	Protocol MCPProtocol

	// Stdio 特定配置
	Command string
	Args    []string
	Env     []string

	// SSE 特定配置
	BaseURL string

	// 可选：过滤要包含的工具
	// 若为空，则加载所有工具
	ToolWhitelist []string
}

// NewMCPTools 创建 MCP Client 并加载工具。
// 返回工具列表和清理函数。清理函数用于关闭 Client。
// 注意：Client 的底层运行依赖于内部创建的 Context，该 Context 会在调用 cleanup 时取消。
// 传入的 ctx 仅用于初始化过程（握手超时控制）。
func NewMCPTools(ctx context.Context, cfg MCPConfig) ([]tool.BaseTool, func() error, error) {
	var (
		cli client.MCPClient
		err error
	)

	// 创建一个独立的 Context 用于 Client 的生命周期管理
	// 这样可以支持用户手动 Close，而不受传入 ctx (可能是请求级) 的限制
	clientCtx, cancel := context.WithCancel(context.Background())

	// 定义清理函数
	cleanup := func() error {
		cancel() // 取消 Context，触发 Start 退出
		return cli.Close()
	}

	// 1. 创建并启动 Client
	switch cfg.Protocol {
	case MCPProtocolStdio:
		c, err := client.NewStdioMCPClient(cfg.Command, cfg.Env, cfg.Args...)
		if err != nil {
			cancel()
			return nil, nil, fmt.Errorf("create Stdio client: %w", err)
		}
		if err := c.Start(clientCtx); err != nil {
			cancel()
			return nil, nil, fmt.Errorf("start Stdio client: %w", err)
		}
		cli = c
	case MCPProtocolSSE:
		c, err := client.NewSSEMCPClient(cfg.BaseURL)
		if err != nil {
			cancel()
			return nil, nil, fmt.Errorf("create SSE client: %w", err)
		}
		if err := c.Start(clientCtx); err != nil {
			cancel()
			return nil, nil, fmt.Errorf("start SSE client: %w", err)
		}
		cli = c
	default:
		cancel()
		return nil, nil, fmt.Errorf("unsupported MCP protocol: %s", cfg.Protocol)
	}

	// 2. 初始化握手
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    cfg.Name,
		Version: cfg.Version,
	}
	if initRequest.Params.ClientInfo.Name == "" {
		initRequest.Params.ClientInfo.Name = "eino-agent"
	}
	if initRequest.Params.ClientInfo.Version == "" {
		initRequest.Params.ClientInfo.Version = "1.0.0"
	}

	// 设置初始化超时
	initCtx, initCancel := context.WithTimeout(ctx, 60*time.Second)
	defer initCancel()

	_, err = cli.Initialize(initCtx, initRequest)
	if err != nil {
		// 初始化失败，清理资源
		cleanup()
		return nil, nil, fmt.Errorf("initialize MCP client: %w", err)
	}

	// 3. 加载工具
	mcpCfg := &einomcp.Config{
		Cli:          cli,
		ToolNameList: cfg.ToolWhitelist,
	}

	// 使用传入的 ctx 进行工具发现 (单次操作)
	tools, err := einomcp.GetTools(ctx, mcpCfg)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("get tools from MCP client: %w", err)
	}

	return tools, cleanup, nil
}
