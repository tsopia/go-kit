// Package generator provides GORM model generation functionality.
package generator

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"gorm.io/gen"
	"github.com/tsopia/go-kit/database"
)

// Options holds generation options.
type Options struct {
	DB      *database.Database
	OutPath string
	Tables  []string
}

// Generator handles code generation.
type Generator struct {
	opts Options
}

// New creates a new generator.
func New(opts Options) *Generator {
	return &Generator{opts: opts}
}

// Run executes the code generation.
func (g *Generator) Run(ctx context.Context) error {
	gormDB := g.opts.DB.Raw()
	if gormDB == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Configure generator
	cfg := gen.Config{
		OutPath:      g.opts.OutPath,
		ModelPkgPath: filepath.Join(g.opts.OutPath, "model"),
		Mode:         gen.WithDefaultQuery | gen.WithQueryInterface,
	}

	gormGen := gen.NewGenerator(cfg)
	gormGen.UseDB(gormDB)

	// Generate tables
	if len(g.opts.Tables) > 0 {
		// Generate specific tables
		for _, table := range g.opts.Tables {
			if table = strings.TrimSpace(table); table != "" {
				gormGen.ApplyBasic(gormGen.GenerateModel(table))
			}
		}
	} else {
		// Generate all tables
		gormGen.ApplyBasic(gormGen.GenerateAllTable()...)
	}

	// Execute generation
	gormGen.Execute()
	return nil
}
