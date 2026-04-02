package main

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gen"
	"gorm.io/gorm"
)

type genOptions struct {
	DB      *gorm.DB
	OutPath string
	Tables  []string
}

func runGen(_ context.Context, opts genOptions) (err error) {
	if opts.DB == nil {
		return fmt.Errorf("database connection is nil")
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("gorm gen panic: %v", r)
		}
	}()

	cfg := gen.Config{
		OutPath: opts.OutPath,
		Mode:    gen.WithDefaultQuery | gen.WithQueryInterface,
	}

	gormGen := gen.NewGenerator(cfg)
	gormGen.UseDB(opts.DB)

	if len(opts.Tables) > 0 {
		for _, table := range opts.Tables {
			table = strings.TrimSpace(table)
			if table != "" {
				gormGen.ApplyBasic(gormGen.GenerateModel(table))
			}
		}
	} else {
		gormGen.ApplyBasic(gormGen.GenerateAllTable()...)
	}

	gormGen.Execute()
	return nil
}
