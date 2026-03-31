package cmd

import (
	"context"
	"fmt"

	"github.com/tsopia/go-kit/dbmigrate"
	"github.com/tsopia/go-kit/cmd/gokit-gen/internal/discover"
)

// MigrateCmd handles database migration operations.
type MigrateCmd struct {
	Config     *discover.Config
	SourcePath string
	Command    string // up, down, status, version
}

// Run executes the migration command.
func (c *MigrateCmd) Run(ctx context.Context) error {
	db, err := connectDatabase(c.Config)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	cfg := dbmigrate.Config{
		SourcePath: c.SourcePath,
		Database:   db,
	}

	switch c.Command {
	case "up":
		if err := dbmigrate.Up(ctx, cfg); err != nil {
			return err
		}
		fmt.Println("Migration completed: up")

	case "down":
		if err := dbmigrate.Down(ctx, cfg); err != nil {
			return err
		}
		fmt.Println("Migration completed: down")

	case "status":
		status, err := dbmigrate.Status(cfg)
		if err != nil {
			return err
		}
		fmt.Println("Migration status:", status)

	case "version":
		version, dirty, err := dbmigrate.Version(cfg)
		if err != nil {
			return err
		}
		fmt.Printf("Version: %d (dirty: %v)\n", version, dirty)

	default:
		return fmt.Errorf("unknown migrate command: %s", c.Command)
	}

	return nil
}
