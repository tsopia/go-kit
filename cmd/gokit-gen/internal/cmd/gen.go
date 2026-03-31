package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/tsopia/go-kit/cmd/gokit-gen/internal/discover"
	"github.com/tsopia/go-kit/cmd/gokit-gen/internal/generator"
)

// GenCmd handles code generation operations.
type GenCmd struct {
	Config  *discover.Config
	OutPath string
	Tables  []string
}

// Run executes the code generation.
func (c *GenCmd) Run(ctx context.Context) error {
	db, err := connectDatabase(c.Config)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(c.OutPath, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	gen := generator.New(generator.Options{
		DB:      db,
		OutPath: c.OutPath,
		Tables:  c.Tables,
	})

	if err := gen.Run(ctx); err != nil {
		return fmt.Errorf("generate code: %w", err)
	}

	fmt.Printf("Generated code to: %s\n", c.OutPath)
	if len(c.Tables) > 0 {
		fmt.Printf("Tables: %s\n", strings.Join(c.Tables, ", "))
	} else {
		fmt.Println("Tables: all")
	}

	return nil
}
