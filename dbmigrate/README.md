# DBMigrate 包

基于 [golang-migrate](https://github.com/golang-migrate/migrate) 的数据库迁移封装，支持 MySQL、PostgreSQL、SQLite。

## 定位

`dbmigrate` 是工具型包：

- 纯函数式 API，无全局状态
- 接受已有的 `*sql.DB` 连接，不负责建连
- 每次调用独立创建和销毁 migrate 实例，无需生命周期管理

## 安装

```bash
go get github.com/tsopia/go-kit/dbmigrate
```

## 快速开始

```go
package main

import (
	"context"
	"database/sql"
	"log"

	"github.com/tsopia/go-kit/dbmigrate"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := sql.Open("mysql", "user:pass@tcp(127.0.0.1:3306)/app")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	cfg := dbmigrate.Config{
		SourcePath: "./migrations",
		DB:         db,
		DriverName: "mysql",
	}

	// 执行所有待处理的迁移
	if err := dbmigrate.Up(context.Background(), cfg); err != nil {
		log.Fatal(err)
	}
}
```

## 配置说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| SourcePath | string | 是 | 迁移文件目录路径 |
| DB | *sql.DB | 是 | 已建立的数据库连接 |
| DriverName | string | 是 | 驱动名：mysql / postgres / sqlite |

## API

| 函数 | 说明 |
|------|------|
| `Up(ctx, cfg)` | 执行所有待处理的 up 迁移 |
| `Down(ctx, cfg)` | 回退一个迁移版本 |
| `UpTo(ctx, cfg, version)` | 迁移到指定版本 |
| `DownTo(ctx, cfg, version)` | 回退到指定版本 |
| `Version(ctx, cfg)` | 获取当前迁移版本号和脏状态 |
| `Status(ctx, cfg)` | 返回 `MigrateStatus` 结构体 |
| `Force(ctx, cfg, version)` | 强制设置版本（修复脏状态） |

## 迁移文件命名

迁移文件放在 `SourcePath` 指定的目录下，命名格式：

```
000001_create_users.up.sql
000001_create_users.down.sql
000002_add_email_index.up.sql
000002_add_email_index.down.sql
```

## 常用操作

### 查看当前版本

```go
status, err := dbmigrate.Status(ctx, cfg)
if err != nil {
	return fmt.Errorf("check migration status: %w", err)
}
fmt.Printf("Version: %d, Dirty: %v\n", status.Version, status.Dirty)
```

### 回退一个版本

```go
if err := dbmigrate.Down(ctx, cfg); err != nil {
	return fmt.Errorf("rollback migration: %w", err)
}
```

### 修复脏状态

迁移中断后数据库可能处于脏状态，使用 `Force` 修复：

```go
if err := dbmigrate.Force(ctx, cfg, int(targetVersion)); err != nil {
	return fmt.Errorf("force version: %w", err)
}
```

## 与 database 包集成

```go
db, err := database.New(&database.Config{Driver: "mysql", /* ... */})
if err != nil {
	return err
}
defer db.Close()

sqlDB, err := db.SQLDB()
if err != nil {
	return err
}

cfg := dbmigrate.Config{
	SourcePath: "./migrations",
	DB:         sqlDB,
	DriverName: "mysql",
}
dbmigrate.Up(ctx, cfg)
```

## 测试

```bash
go test ./dbmigrate -v
```
