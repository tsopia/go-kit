package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/tsopia/go-kit/cmd/gokit-gen/internal"
	"github.com/tsopia/go-kit/database"
	"github.com/tsopia/go-kit/dbmigrate"
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
		if err := runGenCmd(ctx, os.Args[2:]); err != nil {
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
  --dsn              Database DSN (auto-detects driver)
  --driver           Database driver: mysql, postgres, sqlite (optional, validated against DSN)
  --config           Config file path
  --migration-path   Migration files directory (default: migrations)
  --out              Output directory for generated code (default: internal/model)
  --tables           Comma-separated list of tables to generate (default: all)

Examples:
  gokit-gen sync                                                        # Auto-discover config
  gokit-gen sync --out ./pkg/model                                      # Custom output directory
  gokit-gen migrate up                                                  # Run pending migrations
  gokit-gen migrate down                                                # Rollback one migration
  gokit-gen migrate status                                              # Show migration status
  gokit-gen gen --tables user,order                                     # Generate specific tables
  gokit-gen sync --dsn "root:pass@tcp(localhost:3306)/db"               # MySQL DSN
  gokit-gen sync --dsn "postgres://user:pass@host:5432/db"              # PostgreSQL DSN
`)
}

// --- migrate command ---

func runMigrate(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	dsn := fs.String("dsn", "", "Database DSN")
	driver := fs.String("driver", "", "Database driver")
	config := fs.String("config", "", "Config file path")
	source := fs.String("migration-path", "migrations", "Migration files directory")
	fs.Parse(args)

	if len(fs.Args()) == 0 {
		return fmt.Errorf("migrate command required: up, down, status, version, force <version>")
	}
	command := fs.Args()[0]

	dbCfg, err := internal.LoadDatabaseConfig(internal.LoadOptions{
		DSN:        *dsn,
		Driver:     *driver,
		ConfigPath: *config,
		WorkDir:    ".",
	})
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := database.New(dbCfg)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	sqlDB, err := db.SQLDB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}

	mCfg := dbmigrate.Config{
		SourcePath: *source,
		DB:         sqlDB,
		DriverName: dbCfg.Driver,
	}

	switch command {
	case "up":
		if err := dbmigrate.Up(ctx, mCfg); err != nil {
			return err
		}
		fmt.Println("Migration completed: up")
	case "down":
		if err := dbmigrate.Down(ctx, mCfg); err != nil {
			return err
		}
		fmt.Println("Migration completed: down")
	case "status":
		st, err := dbmigrate.Status(ctx, mCfg)
		if err != nil {
			return err
		}
		if st.Version == 0 {
			fmt.Println("Migration status: no migrations applied")
		} else {
			dirty := ""
			if st.Dirty {
				dirty = " (dirty)"
			}
			fmt.Printf("Migration status: version %d%s\n", st.Version, dirty)
		}
	case "version":
		v, dirty, err := dbmigrate.Version(ctx, mCfg)
		if err != nil {
			return err
		}
		fmt.Printf("Version: %d (dirty: %v)\n", v, dirty)
	case "force":
		if len(fs.Args()) < 2 {
			return fmt.Errorf("force requires a version argument")
		}
		var version int
		if _, err := fmt.Sscanf(fs.Args()[1], "%d", &version); err != nil {
			return fmt.Errorf("invalid version: %s", fs.Args()[1])
		}
		if err := dbmigrate.Force(ctx, mCfg, version); err != nil {
			return err
		}
		fmt.Printf("Forced version: %d\n", version)
	default:
		return fmt.Errorf("unknown migrate command: %s", command)
	}

	return nil
}

// --- gen command ---

func runGenCmd(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("gen", flag.ExitOnError)
	dsn := fs.String("dsn", "", "Database DSN")
	driver := fs.String("driver", "", "Database driver")
	config := fs.String("config", "", "Config file path")
	out := fs.String("out", "internal/model", "Output directory")
	tables := fs.String("tables", "", "Comma-separated table names")
	fs.Parse(args)

	dbCfg, err := internal.LoadDatabaseConfig(internal.LoadOptions{
		DSN:        *dsn,
		Driver:     *driver,
		ConfigPath: *config,
		WorkDir:    ".",
	})
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := database.New(dbCfg)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	if err := runGen(ctx, genOptions{
		DB:      db.GetDB(),
		OutPath: *out,
		Tables:  parseTables(*tables),
	}); err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	fmt.Printf("Generated code to: %s\n", *out)
	return nil
}

// --- sync command ---

func runSync(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("sync", flag.ExitOnError)
	dsn := fs.String("dsn", "", "Database DSN")
	driver := fs.String("driver", "", "Database driver")
	config := fs.String("config", "", "Config file path")
	migrateSource := fs.String("migration-path", "migrations", "Migration files directory")
	out := fs.String("out", "internal/model", "Output directory")
	tables := fs.String("tables", "", "Comma-separated table names")
	fs.Parse(args)

	// Parse config, create single connection
	dbCfg, err := internal.LoadDatabaseConfig(internal.LoadOptions{
		DSN:        *dsn,
		Driver:     *driver,
		ConfigPath: *config,
		WorkDir:    ".",
	})
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := database.New(dbCfg)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	// Step 1: migrate up
	sqlDB, err := db.SQLDB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}

	if err := dbmigrate.Up(ctx, dbmigrate.Config{
		SourcePath: *migrateSource,
		DB:         sqlDB,
		DriverName: dbCfg.Driver,
	}); err != nil {
		return fmt.Errorf("migrate failed: %w", err)
	}
	fmt.Println("Migration completed: up")

	// Step 2: gen
	if err := runGen(ctx, genOptions{
		DB:      db.GetDB(),
		OutPath: *out,
		Tables:  parseTables(*tables),
	}); err != nil {
		return fmt.Errorf("gen failed: %w", err)
	}

	fmt.Printf("Generated code to: %s\n", *out)
	fmt.Println("Sync completed successfully")
	return nil
}

func parseTables(tables string) []string {
	if tables == "" {
		return nil
	}
	return strings.Split(tables, ",")
}
