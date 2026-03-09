package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tsopia/go-kit/cmd/gokit/pkg/gokit"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "更新 go-kit AI 指南和能力清单",
	Long: `从 go-kit 源同步最新的能力清单和 AI 指南到当前项目。

更新以下文件：
- .go-kit/capabilities.yaml - 最新能力清单
- .go-kit/GUIDE.md - 更新的使用指南`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 检查当前目录是否为 go-kit 项目（检查 .go-kit 目录）
		goKitDir := ".go-kit"
		if _, err := os.Stat(goKitDir); os.IsNotExist(err) {
			return fmt.Errorf("当前项目未初始化 go-kit，请先运行: gokit init")
		}

		// 更新 capabilities.yaml
		capsPath := filepath.Join(goKitDir, "capabilities.yaml")
		if err := copyCapabilities(capsPath, true); err != nil {
			return err
		}
		fmt.Printf("✓ 已更新: %s\n", capsPath)

		// 更新 GUIDE.md
		guidePath := filepath.Join(goKitDir, "GUIDE.md")
		if err := generateGuide(guidePath, true); err != nil {
			return err
		}
		fmt.Printf("✓ 已更新: %s\n", guidePath)

		fmt.Println("\n✓ go-kit 能力清单已更新！")

		// 显示更新内容摘要
		caps, err := gokit.LoadCapabilities()
		if err == nil {
			fmt.Printf("\n当前可用能力 (%d 个):\n", len(caps))
			for _, c := range caps {
				fmt.Printf("  - %s: %s\n", c.Name, c.Description)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
