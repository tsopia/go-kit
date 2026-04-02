package dbmigrate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/file"
)

// Config holds the configuration for migration operations.
type Config struct {
	// SourcePath is the path to the migration files directory.
	SourcePath string
	// DB is an established *sql.DB connection.
	DB *sql.DB
	// DriverName is the database driver name: mysql | postgres | sqlite.
	DriverName string
}

// MigrateStatus represents the migration status.
type MigrateStatus struct {
	Version uint
	Dirty   bool
}

// Up executes all pending up migrations.
func Up(ctx context.Context, cfg Config) error {
	m, err := createMigrate(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// Down rolls back one migration version.
func Down(ctx context.Context, cfg Config) error {
	m, err := createMigrate(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Steps(-1); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate down: %w", err)
	}
	return nil
}

// UpTo migrates to the specified version.
func UpTo(ctx context.Context, cfg Config, version uint) error {
	m, err := createMigrate(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Migrate(version); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate to version %d: %w", version, err)
	}
	return nil
}

// DownTo rolls back to the specified version.
func DownTo(ctx context.Context, cfg Config, version uint) error {
	m, err := createMigrate(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Migrate(version); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate down to version %d: %w", version, err)
	}
	return nil
}

// Version returns the current migration version and dirty state.
func Version(ctx context.Context, cfg Config) (uint, bool, error) {
	m, err := createMigrate(cfg)
	if err != nil {
		return 0, false, err
	}
	defer m.Close()

	v, dirty, err := m.Version()
	if err == migrate.ErrNilVersion {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("get version: %w", err)
	}
	return v, dirty, nil
}

// Status returns the migration status.
func Status(ctx context.Context, cfg Config) (MigrateStatus, error) {
	v, dirty, err := Version(ctx, cfg)
	if err != nil {
		return MigrateStatus{}, err
	}
	return MigrateStatus{Version: v, Dirty: dirty}, nil
}

// Force forces the migration version, useful for fixing dirty state.
func Force(ctx context.Context, cfg Config, version int) error {
	m, err := createMigrate(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Force(version); err != nil {
		return fmt.Errorf("force version %d: %w", version, err)
	}
	return nil
}

func createMigrate(cfg Config) (*migrate.Migrate, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	if cfg.SourcePath == "" {
		return nil, fmt.Errorf("source path is empty")
	}

	source, err := (&file.File{}).Open("file://" + cfg.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("open migration source: %w", err)
	}

	dbDriver, err := openDriver(cfg.DB, cfg.DriverName)
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithInstance("file", source, cfg.DriverName, dbDriver)
	if err != nil {
		return nil, fmt.Errorf("create migrate instance: %w", err)
	}
	return m, nil
}
