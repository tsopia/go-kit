package cmd

import (
	"context"
	"fmt"

	"github.com/tsopia/go-kit/cmd/gokit-gen/internal/discover"
)

// SyncCmd runs migrate then gen.
type SyncCmd struct {
	Config        *discover.Config
	MigrateSource string
	OutPath       string
	Tables        []string
}

// Run executes the sync command.
func (c *SyncCmd) Run(ctx context.Context) error {
	// Step 1: Migrate
	migrateCmd := &MigrateCmd{
		Config:     c.Config,
		SourcePath: c.MigrateSource,
		Command:    "up",
	}

	if err := migrateCmd.Run(ctx); err != nil {
		return fmt.Errorf("migrate failed: %w", err)
	}

	// Step 2: Gen
	genCmd := &GenCmd{
		Config:  c.Config,
		OutPath: c.OutPath,
		Tables:  c.Tables,
	}

	if err := genCmd.Run(ctx); err != nil {
		return fmt.Errorf("gen failed: %w", err)
	}

	fmt.Println("Sync completed successfully")
	return nil
}
