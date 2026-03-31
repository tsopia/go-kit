// Package dbmigrate provides database migration support using golang-migrate.
// This is an optional extension to the database package and is not required for
// the core database functionality.
package dbmigrate

import (
	"context"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/tsopia/go-kit/database"
)

// Config holds the configuration for migration operations.
type Config struct {
	// SourcePath is the path to the migration files directory.
	// Example: "migrations" or "db/migrations"
	SourcePath string

	// Database is the go-kit database instance.
	Database *database.Database
}

// Up executes all pending up migrations.
func Up(ctx context.Context, cfg Config) error {
	dsn, err := buildDSN(cfg.Database)
	if err != nil {
		return fmt.Errorf("build dsn: %w", err)
	}

	m, err := migrate.New(
		"file://"+cfg.SourcePath,
		dsn,
	)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}

	return nil
}

// UpTo migrates up to the specific version.
// Returns error if target version is less than current version.
func UpTo(ctx context.Context, cfg Config, version uint) error {
	m, err := createMigrate(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	currentVersion, err := getCurrentVersion(m)
	if err != nil {
		return fmt.Errorf("get current version: %w", err)
	}

	if version < currentVersion {
		return fmt.Errorf("target version %d must be greater than or equal to current version %d", version, currentVersion)
	}

	if err := m.Migrate(version); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate to version %d: %w", version, err)
	}

	return nil
}

// Down executes one down migration (rollback one version).
func Down(ctx context.Context, cfg Config) error {
	dsn, err := buildDSN(cfg.Database)
	if err != nil {
		return fmt.Errorf("build dsn: %w", err)
	}

	m, err := migrate.New(
		"file://"+cfg.SourcePath,
		dsn,
	)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Steps(-1); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate down: %w", err)
	}

	return nil
}

// DownTo migrates down to the specific version.
// Returns error if target version is greater than or equal to current version.
func DownTo(ctx context.Context, cfg Config, version uint) error {
	m, err := createMigrate(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	currentVersion, err := getCurrentVersion(m)
	if err != nil {
		return fmt.Errorf("get current version: %w", err)
	}

	if version >= currentVersion {
		return fmt.Errorf("target version %d must be less than current version %d", version, currentVersion)
	}

	if err := m.Migrate(version); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate down to version %d: %w", version, err)
	}

	return nil
}

// Version returns the current migration version and dirty state.
func Version(cfg Config) (version uint, dirty bool, err error) {
	dsn, err := buildDSN(cfg.Database)
	if err != nil {
		return 0, false, fmt.Errorf("build dsn: %w", err)
	}

	m, err := migrate.New(
		"file://"+cfg.SourcePath,
		dsn,
	)
	if err != nil {
		return 0, false, fmt.Errorf("create migrate instance: %w", err)
	}
	defer m.Close()

	version, dirty, err = m.Version()
	if err == migrate.ErrNilVersion {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("get version: %w", err)
	}

	return version, dirty, nil
}

// Status returns the migration status as a string.
func Status(cfg Config) (string, error) {
	version, dirty, err := Version(cfg)
	if err != nil {
		return "", err
	}

	if version == 0 {
		return "no migrations applied", nil
	}

	status := fmt.Sprintf("version %d", version)
	if dirty {
		status += " (dirty)"
	}

	return status, nil
}

// Force forces the migration version without running migrations.
// Useful for fixing dirty state.
func Force(cfg Config, version int) error {
	dsn, err := buildDSN(cfg.Database)
	if err != nil {
		return fmt.Errorf("build dsn: %w", err)
	}

	m, err := migrate.New(
		"file://"+cfg.SourcePath,
		dsn,
	)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Force(version); err != nil {
		return fmt.Errorf("force version %d: %w", version, err)
	}

	return nil
}

func buildDSN(db *database.Database) (string, error) {
	if db == nil {
		return "", fmt.Errorf("database is nil")
	}
	cfg := db.GetConfig()

	switch cfg.Driver {
	case "mysql":
		return fmt.Sprintf("mysql://%s:%s@tcp(%s:%d)/%s?multiStatements=true",
			cfg.Username,
			cfg.Password,
			cfg.Host,
			cfg.Port,
			cfg.Database,
		), nil
	case "postgres", "postgresql":
		sslmode := cfg.SSLMode
		if sslmode == "" {
			sslmode = "prefer"
		}
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			cfg.Username,
			cfg.Password,
			cfg.Host,
			cfg.Port,
			cfg.Database,
			sslmode,
		), nil
	default:
		return "", fmt.Errorf("unsupported driver: %s", cfg.Driver)
	}
}

// createMigrate creates a new migrate instance from config.
func createMigrate(cfg Config) (*migrate.Migrate, error) {
	dsn, err := buildDSN(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("build dsn: %w", err)
	}

	m, err := migrate.New("file://"+cfg.SourcePath, dsn)
	if err != nil {
		return nil, fmt.Errorf("create migrate instance: %w", err)
	}
	return m, nil
}

// getCurrentVersion returns the current migration version, handling nil version as 0.
func getCurrentVersion(m *migrate.Migrate) (uint, error) {
	version, _, err := m.Version()
	if err == migrate.ErrNilVersion {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return version, nil
}
