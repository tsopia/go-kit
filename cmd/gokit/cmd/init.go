package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tsopia/go-kit/cmd/gokit/pkg/gokit"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "为现有项目初始化 go-kit 支持",
	Long: `在当前目录初始化 go-kit，创建 AI 指南和能力清单。

检测当前目录是否为 Go 项目（检查 go.mod），并在项目中创建：
- .go-kit/capabilities.yaml - go-kit 能力清单
- .go-kit/GUIDE.md - 项目专属的 go-kit 使用指南
- AGENTS.md - AI 助手入口（如不存在）`,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		// 检查当前目录是否为 Go 项目
		if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
			return fmt.Errorf("当前目录不是 Go 项目（缺少 go.mod），请先运行 go mod init")
		}

		// 创建 .go-kit 目录
		goKitDir := ".go-kit"
		if err := os.MkdirAll(goKitDir, 0755); err != nil {
			return fmt.Errorf("创建 .go-kit 目录失败: %w", err)
		}

		// 复制 capabilities.yaml
		capsPath := filepath.Join(goKitDir, "capabilities.yaml")
		if err := copyCapabilities(capsPath, force); err != nil {
			return err
		}

		// 生成 GUIDE.md
		guidePath := filepath.Join(goKitDir, "GUIDE.md")
		if err := generateGuide(guidePath, force); err != nil {
			return err
		}

		// 创建或更新 AGENTS.md
		if err := ensureAgentsMd(force); err != nil {
			return err
		}

		fmt.Println("✓ go-kit 初始化成功！")
		fmt.Println("\n生成的文件:")
		fmt.Printf("  - %s\n", capsPath)
		fmt.Printf("  - %s\n", guidePath)
		fmt.Printf("  - AGENTS.md\n")
		fmt.Println("\nAI 助手现在可以通过 AGENTS.md 了解 go-kit 能力")

		return nil
	},
}

func copyCapabilities(destPath string, force bool) error {
	// 检查文件是否已存在
	if _, err := os.Stat(destPath); err == nil && !force {
		return fmt.Errorf("%s 已存在，使用 --force 覆盖", destPath)
	}

	// 从 go-kit 嵌入资源读取
	caps, err := gokit.LoadCapabilities()
	if err != nil {
		return fmt.Errorf("读取 go-kit 能力清单失败: %w", err)
	}

	// 生成 YAML 内容
	content, err := gokit.DumpCapabilities(caps)
	if err != nil {
		return fmt.Errorf("序列化能力清单失败: %w", err)
	}

	if err := os.WriteFile(destPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入 %s 失败: %w", destPath, err)
	}

	return nil
}

func generateGuide(destPath string, force bool) error {
	// 检查文件是否已存在
	if _, err := os.Stat(destPath); err == nil && !force {
		return fmt.Errorf("%s 已存在，使用 --force 覆盖", destPath)
	}

	caps, err := gokit.LoadCapabilities()
	if err != nil {
		return fmt.Errorf("读取能力清单失败: %w", err)
	}

	content := generateGuideContent(caps)

	if err := os.WriteFile(destPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入 %s 失败: %w", destPath, err)
	}

	return nil
}

func generateGuideContent(caps []gokit.Capability) string {
	content := `# Go-Kit AI 使用指南

本文档描述本项目使用的 go-kit 工具库能力。

## 能力概览

| 能力 | 描述 | 导入路径 |
|------|------|----------|
`
	for _, c := range caps {
		content += fmt.Sprintf("| %s | %s | `%s` |\n", c.Name, c.Description, c.Import)
	}

	content += `
## 详细使用说明

`
	for _, c := range caps {
		content += fmt.Sprintf("### %s\n\n", c.Name)
		content += fmt.Sprintf("- **描述**: %s\n", c.Description)
		content += fmt.Sprintf("- **导入**: `%s`\n", c.Import)
		if len(c.Dependencies) > 0 {
			content += fmt.Sprintf("- **依赖**: %v\n", c.Dependencies)
		}
		content += "\n**使用场景:**\n\n"
		for _, s := range c.Scenarios {
			content += fmt.Sprintf("**%s**:\n\n```go\n%s\n```\n\n", s.Name, s.Snippet)
		}
		content += "\n"
	}

	content += `## 开发规范

- 所有 error 必须显式处理，使用 fmt.Errorf("context: %w", err) 包装
- 导出用 PascalCase，内部用 camelCase
- 优先使用指针接收者 (s *Service)
- 必须支持 context.Context 传播

## 更新指南

当 go-kit 更新后，运行以下命令同步能力清单：


gokit update

`

	return content
}

func ensureAgentsMd(force bool) error {
	agentsPath := "AGENTS.md"

	// 如果 AGENTS.md 已存在且不强制覆盖，则追加引用
	if _, err := os.Stat(agentsPath); err == nil && !force {
		// 读取现有内容，检查是否已有 go-kit 引用
		content, err := os.ReadFile(agentsPath)
		if err != nil {
			return fmt.Errorf("读取 %s 失败: %w", agentsPath, err)
		}

		// 如果已包含 .go-kit/GUIDE.md 引用，则不修改
		if contains(string(content), ".go-kit/GUIDE.md") {
			fmt.Printf("  (AGENTS.md 已包含 go-kit 引用，跳过)\n")
			return nil
		}

		// 追加引用
		newContent := string(content) + `

## 工具库引用

本项目使用 go-kit 提供基础能力，详细指南请参考 [.go-kit/GUIDE.md](.go-kit/GUIDE.md)。
`
		if err := os.WriteFile(agentsPath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("更新 %s 失败: %w", agentsPath, err)
		}
		return nil
	}

	// 创建新的 AGENTS.md
	content := `# AI 开发指南

## 项目规范

开发本项目时，请参考 [.go-kit/GUIDE.md](.go-kit/GUIDE.md) 使用 go-kit 能力。

## 工具库

本项目使用 [go-kit](https://github.com/tsopia/go-kit) 提供基础能力：
- 日志记录 (kit)
- 数据库连接 (database)
- 配置管理 (cfg)
- HTTP 客户端/服务器 (httpclient/httpserver)
- 消息队列 (pgmq)

详细能力清单和代码示例请参考 [.go-kit/GUIDE.md](.go-kit/GUIDE.md)。
`

	if err := os.WriteFile(agentsPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("创建 %s 失败: %w", agentsPath, err)
	}

	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolP("force", "f", false, "强制覆盖已存在的文件")
}
