package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tsopia/go-kit/cmd/gokit/pkg/scaffold"
)

var newCmd = &cobra.Command{
	Use:   "new [name]",
	Short: "Create a new go-kit project",
	Long:  `Create a new Go project with go-kit integration and AI-friendly configuration.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		templateType, _ := cmd.Flags().GetString("template")
		module, _ := cmd.Flags().GetString("module")
		output, _ := cmd.Flags().GetString("output")

		// Default module name if not specified
		if module == "" {
			module = fmt.Sprintf("github.com/yourcompany/%s", name)
		}

		// Default output if not specified
		if output == "" {
			output = name
		}

		cfg := scaffold.Config{
			Name:         name,
			Module:       module,
			GoKitModule:  "github.com/tsopia/go-kit",
			GoKitVersion: "v0.0.0",
			Template:     templateType,
			OutputDir:    output,
		}

		fmt.Printf("Creating new %s project: %s\n", templateType, name)
		fmt.Printf("Module: %s\n", module)
		fmt.Printf("Output: %s\n", output)

		if err := scaffold.CreateProject(cfg); err != nil {
			return fmt.Errorf("create project: %w", err)
		}

		fmt.Printf("✓ Project created successfully!\n")
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  cd %s\n", output)
		fmt.Printf("  go mod tidy\n")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(newCmd)

	newCmd.Flags().StringP("template", "t", "api", "Project template (api|worker|cron|library)")
	newCmd.Flags().StringP("module", "m", "", "Go module name (default: github.com/yourcompany/<name>)")
	newCmd.Flags().StringP("output", "o", "", "Output directory (default: ./<name>)")
}
