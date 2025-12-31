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
}
