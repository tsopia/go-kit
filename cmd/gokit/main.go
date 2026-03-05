package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gokit",
	Short: "Go-kit project scaffolding tool",
	Long: `gokit is a CLI tool for creating and managing Go projects
that use the go-kit library. It provides project templates,
AI-friendly configuration, and go-kit integration.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
