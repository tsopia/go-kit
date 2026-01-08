package pgmq

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/tsopia/go-kit/database"
)

// DB 适配最小数据库能力
type DB interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// SQLDBAdapter 适配 *sql.DB
type SQLDBAdapter struct {
	db *sql.DB
}

// NewSQLDBAdapter 创建 SQLDBAdapter
func NewSQLDBAdapter(db *sql.DB) (*SQLDBAdapter, error) {
	if db == nil {
		return nil, ErrMissingDB
	}
	return &SQLDBAdapter{db: db}, nil
}

func (a *SQLDBAdapter) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return a.db.ExecContext(ctx, query, args...)
}

func (a *SQLDBAdapter) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return a.db.QueryContext(ctx, query, args...)
}

func (a *SQLDBAdapter) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return a.db.QueryRowContext(ctx, query, args...)
}

// NewDatabaseAdapter 适配 go-kit/database.DB
func NewDatabaseAdapter(db database.DB) (DB, error) {
	if db == nil {
		return nil, ErrMissingDB
	}
	sqlDB, err := db.SQLDB()
	if err != nil {
		return nil, fmt.Errorf("获取 sql.DB 失败: %w", err)
	}
	return &SQLDBAdapter{db: sqlDB}, nil
}
