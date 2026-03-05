package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tsopia/go-kit/cmd/gokit/pkg/gokit"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available go-kit capabilities",
	Long:  `List all capabilities provided by go-kit with their descriptions and usage examples.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		caps, err := gokit.LoadCapabilities()
		if err != nil {
			return fmt.Errorf("load capabilities: %w", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tDESCRIPTION\tIMPORT")

		for _, c := range caps {
			fmt.Fprintf(w, "%s\t%s\t%s\n", c.Name, c.Description, c.Import)
		}

		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
