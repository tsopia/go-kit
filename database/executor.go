package database

import (
	"context"
	"database/sql"

	"gorm.io/gorm"
)

// Executor 承载常规执行能力，便于与连接/健康检查解耦。
type Executor interface {
	Exec(ctx context.Context, query string, args ...interface{}) error
	Query(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	Tx(ctx context.Context, fn func(tx *gorm.DB) error, opts ...*sql.TxOptions) error
	BeginTx(ctx context.Context, opts ...*sql.TxOptions) (*gorm.DB, error)
}

type gormExecutor struct {
	dbProvider func() *gorm.DB
	logger     SimpleLogger
}

func newGormExecutor(dbProvider func() *gorm.DB, logger SimpleLogger) *gormExecutor {
	return &gormExecutor{dbProvider: dbProvider, logger: logger}
}

func (e *gormExecutor) Exec(ctx context.Context, query string, args ...interface{}) error {
	return e.dbProvider().WithContext(ctx).Exec(query, args...).Error
}

func (e *gormExecutor) Query(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return e.dbProvider().WithContext(ctx).Raw(query, args...).Scan(dest).Error
}

func (e *gormExecutor) Tx(ctx context.Context, fn func(tx *gorm.DB) error, opts ...*sql.TxOptions) error {
	tx, err := e.BeginTx(ctx, opts...)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		if e.logger != nil {
			e.logger.Error("transaction failed", "error", err)
		}
		return tx.Rollback().Error
	}

	return tx.Commit().Error
}

func (e *gormExecutor) BeginTx(ctx context.Context, opts ...*sql.TxOptions) (*gorm.DB, error) {
	session := e.dbProvider()
	if ctx != nil {
		session = session.WithContext(ctx)
	}

	var tx *gorm.DB
	if len(opts) > 0 && opts[0] != nil {
		tx = session.Begin(opts[0])
	} else {
		tx = session.Begin()
	}

	return tx, tx.Error
}
