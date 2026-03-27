# Database Package

基于 GORM 的数据库封装，提供连接管理、事务执行、健康检查以及受控的底层逃逸能力。

## 定位

`database` 是资源型 SDK：

- 主路径是显式实例：`New` / `NewWithOptions`
- 兼容路径是默认实例：`Configure` / `GetClient`
- 单库应用可用默认实例
- 多库、测试、框架初始化、事务编排优先使用显式实例

## 特性

- 支持 MySQL、PostgreSQL、SQLite
- 配置校验与默认值补齐
- 连接池与连接重试
- `Connector` / `Executor` / `HealthChecker` 可替换
- 支持 `Hooks` 扩展生命周期
- 保留 `GetDB` / `Raw` / `SQLDB` 以支持高级 GORM 或驱动原生能力

## 快速开始

### 显式实例

```go
package main

import (
	"context"
	"log"

	"github.com/tsopia/go-kit/database"
)

func main() {
	cfg := &database.Config{
		Driver:   "sqlite",
		Database: ":memory:",
		LogLevel: "silent",
	}

	db, err := database.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	var one int
	if err := db.Query(context.Background(), &one, "SELECT 1"); err != nil {
		log.Fatal(err)
	}
}
```

### 默认实例

```go
cfg := &database.Config{
	Driver:   "sqlite",
	Database: ":memory:",
}

if _, err := database.Configure(cfg); err != nil {
	return fmt.Errorf("configure database: %w", err)
}
defer database.CloseDefault()

if err := database.Ping(); err != nil {
	return fmt.Errorf("ping database: %w", err)
}

if err := database.Exec(ctx, "UPDATE users SET last_seen = CURRENT_TIMESTAMP WHERE id = ?", []interface{}{userID}); err != nil {
	return fmt.Errorf("update last_seen: %w", err)
}
```

说明：

- 包级 `Ping` / `Exec` / `Query` 是默认实例的便捷包装
- 若未 `Configure` 且未传显式实例，将返回 `ErrMissingClient`
- 实例方法的调用体验更自然，优先推荐

## 配置示例

### MySQL

```go
cfg := &database.Config{
	Driver:   "mysql",
	Host:     "127.0.0.1",
	Port:     3306,
	Username: "root",
	Password: "password",
	Database: "app",
	LogLevel: "warn",
}
```

### PostgreSQL

```go
cfg := &database.Config{
	Driver:   "postgres",
	Host:     "127.0.0.1",
	Port:     5432,
	Username: "postgres",
	Password: "password",
	Database: "app",
	SSLMode:  "disable",
	LogLevel: "warn",
}
```

### SQLite

```go
cfg := &database.Config{
	Driver:   "sqlite",
	Database: "test.db",
	LogLevel: "silent",
}
```

### 重试配置

```go
cfg := &database.Config{
	Driver:             "mysql",
	Host:               "127.0.0.1",
	Port:               3306,
	Username:           "root",
	Password:           "password",
	Database:           "app",
	RetryEnabled:       true,
	RetryMaxAttempts:   5,
	RetryInitialDelay:  500 * time.Millisecond,
	RetryMaxDelay:      10 * time.Second,
	RetryBackoffFactor: 1.5,
	RetryJitterEnabled: true,
}
```

显式关闭重试：

```go
cfg := &database.Config{
	Driver:          "sqlite",
	Database:        "test.db",
	RetryConfigured: true,
	RetryEnabled:    false,
}
```

## 核心 API

### 构造与默认实例

- `New(config *Config) (*Database, error)`
- `NewWithOptions(config *Config, opts ...Option) (*Database, error)`
- `Configure(config *Config, opts ...Option) (*Database, error)`
- `GetClient() *Database`
- `CloseDefault() error`

### 实例方法

- `Exec(ctx context.Context, query string, args ...interface{}) error`
- `Query(ctx context.Context, dest interface{}, query string, args ...interface{}) error`
- `Tx(ctx context.Context, fn func(tx *gorm.DB) error, opts ...*sql.TxOptions) error`
- `BeginTx(ctx context.Context, opts ...*sql.TxOptions) (*gorm.DB, error)`
- `Ping() error`
- `HealthCheck() error`
- `HealthCheckWithContext(ctx context.Context) *HealthStatus`
- `GetDB() *gorm.DB`
- `Raw() *gorm.DB`
- `SQLDB() (*sql.DB, error)`
- `Close() error`

### 包级便捷方法

- `Ping(c ...*Database) error`
- `Exec(ctx context.Context, query string, args []interface{}, c ...*Database) error`
- `Query(ctx context.Context, dest interface{}, query string, args []interface{}, c ...*Database) error`

## 组件化职责

- `Connector`：负责建连、重试、命名策略、连接池
- `Executor`：负责 `Exec` / `Query` / `Tx` / `BeginTx`
- `HealthChecker`：负责 `Ping` / `HealthCheck` / `HealthCheckWithContext`

可通过以下选项替换默认实现：

- `WithConnector`
- `WithExecutor`
- `WithHealthChecker`
- `WithLogger`
- `WithHooks`

## 高级能力

### 与 gorm/gen 集成

```go
q := query.Use(db.GetDB())

err := db.TransactionWithContext(ctx, func(tx *gorm.DB) error {
	qtx := query.Use(tx).WithContext(ctx)
	_, err := qtx.User.Create(&model.User{Name: "foo"})
	return err
})
```

### 使用底层 `*sql.DB`

```go
sqlDB, err := db.SQLDB()
if err != nil {
	return fmt.Errorf("get sql db: %w", err)
}
defer sqlDB.Close()
```

## 测试

```bash
GOCACHE=/tmp/go-build go test ./database -v
```

说明：

- 某些环境直接执行 `go test ./...` 可能受仓库已知 `sonic` 基线问题影响
- 与 `database` 无关时，优先使用包级增量测试验证

## 注意事项

1. `database` 管理连接资源，使用完毕后应调用 `Close()` 或 `CloseDefault()`
2. 默认实例适合单库应用；多库场景优先显式实例
3. 包级 helper 为兼容入口，复杂调用优先实例方法
