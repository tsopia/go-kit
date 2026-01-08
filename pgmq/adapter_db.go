package pgmq

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
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

// NewAdapter 统一适配不同 DB 输入
func NewAdapter(ctx context.Context, source any) (DB, error) {
	switch v := source.(type) {
	case DB:
		return v, nil
	case *sql.DB:
		return NewSQLDBAdapter(v)
	case database.DB:
		return NewDatabaseAdapter(v)
	case string:
		return NewPgxAdapter(ctx, v)
	default:
		return nil, fmt.Errorf("不支持的 DB 类型: %T", source)
	}
}

// NewPgxAdapter 使用 pgx 驱动创建 database/sql 连接池
func NewPgxAdapter(ctx context.Context, connString string) (DB, error) {
	cfg, err := pgx.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("解析连接字符串失败: %w", err)
	}
	db := stdlib.OpenDB(*cfg)
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("连接 pgx 数据库失败: %w", err)
	}
	return &SQLDBAdapter{db: db}, nil
}
