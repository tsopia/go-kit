// gokit-gen is a code generation tool for go-kit projects.
// It provides database migration and GORM model generation capabilities.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/tsopia/go-kit/cmd/gokit-gen/internal/cmd"
	"github.com/tsopia/go-kit/cmd/gokit-gen/internal/discover"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	ctx := context.Background()
	subcommand := os.Args[1]

	switch subcommand {
	case "migrate":
		if err := runMigrate(ctx, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "gen":
		if err := runGen(ctx, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "sync":
		if err := runSync(ctx, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`gokit-gen - Code generation tool for go-kit projects

Usage:
  gokit-gen <command> [flags]

Commands:
  migrate    Database migration operations
  gen        Generate GORM models from database
  sync       Run migrate then gen (recommended)

Global Flags:
  --dsn              Database DSN (overrides auto-discovery)
  --driver           Database driver: mysql, postgres (overrides auto-discovery)
  --migrate-source   Migration files directory (default: migrations)
  --out              Output directory for generated code (default: internal/dal)
  --tables           Comma-separated list of tables to generate (default: all)

Examples:
  gokit-gen sync                                           # Auto-discover config, use defaults
  gokit-gen sync --out ./pkg/model                         # Custom output directory
  gokit-gen migrate up                                     # Run pending migrations
  gokit-gen migrate status                                 # Show migration status
  gokit-gen gen --tables user,order                        # Generate specific tables
  gokit-gen sync --dsn "root:pass@tcp(localhost:3306)/db"  # Manual connection
`)
}

func runMigrate(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	dsn := fs.String("dsn", "", "Database DSN")
	driver := fs.String("driver", "", "Database driver")
	source := fs.String("migrate-source", "migrations", "Migration files directory")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("migrate command required: up, down, status, version")
	}

	migrateCmd := fs.Args()[0]

	// Discover or use provided config
	cfg, err := getConfig(*dsn, *driver)
	if err != nil {
		return err
	}

	cmdMigrate := &cmd.MigrateCmd{
		Config:     cfg,
		SourcePath: *source,
		Command:    migrateCmd,
	}

	return cmdMigrate.Run(ctx)
}

func runGen(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("gen", flag.ExitOnError)
	dsn := fs.String("dsn", "", "Database DSN")
	driver := fs.String("driver", "", "Database driver")
	out := fs.String("out", "internal/dal", "Output directory")
	tables := fs.String("tables", "", "Comma-separated table names")

	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := getConfig(*dsn, *driver)
	if err != nil {
		return err
	}

	cmdGen := &cmd.GenCmd{
		Config: cfg,
		OutPath: *out,
		Tables:  parseTables(*tables),
	}

	return cmdGen.Run(ctx)
}

func runSync(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("sync", flag.ExitOnError)
	dsn := fs.String("dsn", "", "Database DSN")
	driver := fs.String("driver", "", "Database driver")
	migrateSource := fs.String("migrate-source", "migrations", "Migration files directory")
	out := fs.String("out", "internal/dal", "Output directory")
	tables := fs.String("tables", "", "Comma-separated table names")

	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := getConfig(*dsn, *driver)
	if err != nil {
		return err
	}

	cmdSync := &cmd.SyncCmd{
		Config:        cfg,
		MigrateSource: *migrateSource,
		OutPath:       *out,
		Tables:        parseTables(*tables),
	}

	return cmdSync.Run(ctx)
}

func getConfig(dsn, driver string) (*discover.Config, error) {
	// If DSN is provided, use it directly
	if dsn != "" {
		if driver == "" {
			return nil, fmt.Errorf("--driver is required when using --dsn")
		}
		return &discover.Config{
			Driver: driver,
			DSN:    dsn,
		}, nil
	}

	// Try auto-discovery
	cfg, err := discover.FromProject()
	if err != nil {
		return nil, fmt.Errorf("failed to discover config: %w\nProvide --dsn and --driver to skip auto-discovery", err)
	}

	return cfg, nil
}

func parseTables(tables string) []string {
	if tables == "" {
		return nil
	}
	return strings.Split(tables, ",")
}
