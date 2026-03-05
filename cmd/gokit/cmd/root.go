package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gokit",
	Short: "Go-kit project scaffolding tool",
	Long: `gokit is a CLI tool for creating and managing Go projects
that use the go-kit library. It provides project templates,
AI-friendly configuration, and go-kit integration.`,
}

func Execute() error {
	return rootCmd.Execute()
}
