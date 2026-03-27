package database

import (
	"context"
	"database/sql"

	"gorm.io/gorm"
)

// DB 定义对外暴露的最小能力集合，便于业务代码依赖接口而非具体实现。
//
// 绝大多数场景建议通过这些方法完成常规操作；若确需底层能力，可使用
// Raw() 受控地获取底层 *gorm.DB。
type DB interface {
	// Exec 执行写操作，适用于 INSERT/UPDATE/DELETE 等语句。
	Exec(ctx context.Context, query string, args ...interface{}) error

	// Query 执行查询并将结果扫描到 dest。
	Query(ctx context.Context, dest interface{}, query string, args ...interface{}) error

	// Tx 启动事务并在回调中执行业务逻辑，支持传入可选的 sql.TxOptions。
	Tx(ctx context.Context, fn func(tx *gorm.DB) error, opts ...*sql.TxOptions) error

	// Raw 返回底层 *gorm.DB 以支持少数无法封装的高级用法（请谨慎使用）。
	Raw() *gorm.DB

	// SQLDB 返回底层 *sql.DB，便于执行需要专用连接的场景（如 PostgreSQL LISTEN/NOTIFY、
	// 使用特定扩展或 Driver 原生能力）。调用方负责确保合理的生命周期管理。
	SQLDB() (*sql.DB, error)
}

// Ping 使用默认客户端或显式覆盖客户端进行连通性检测。
func Ping(c ...*Database) error {
	client, err := resolveClient(c...)
	if err != nil {
		return err
	}

	return client.Ping()
}

// Exec 使用默认客户端或显式覆盖客户端执行写操作。
func Exec(ctx context.Context, query string, args []interface{}, c ...*Database) error {
	client, err := resolveClient(c...)
	if err != nil {
		return err
	}

	return client.Exec(ctx, query, args...)
}

// Query 使用默认客户端或显式覆盖客户端执行查询。
func Query(ctx context.Context, dest interface{}, query string, args []interface{}, c ...*Database) error {
	client, err := resolveClient(c...)
	if err != nil {
		return err
	}

	return client.Query(ctx, dest, query, args...)
}
