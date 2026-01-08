# PGMQ Package

基于 PostgreSQL `pgmq` 扩展的队列封装，提供泛型消息结构、标准 PGMQ 命名以及带重试与 DLQ 的消费能力。

## ✨ 特性

- **泛型消息**：`Message[T]` 自动完成 JSON 编解码。
- **标准 API**：`Send` / `Read` / `Pop` / `Archive` / `Delete` / `Drop`。
- **自动校验**：初始化时检测 `pgmq` 扩展，可自动创建队列。
- **并发消费者**：支持协程池与渐进式重试、DLQ 自动转移。
- **可观测性**：可插拔 Metrics 接口。

## ✅ 依赖与初始化要求

### pgmq 扩展

- 必须预先安装扩展：
  ```sql
  CREATE EXTENSION IF NOT EXISTS pgmq;
  ```

## 🚀 快速开始

```go
package main

import (
    "context"
    "log"

    "github.com/tsopia/go-kit/database"
    "github.com/tsopia/go-kit/pgmq"
)

func main() {
    db, err := database.New(&database.Config{
        Driver:   "postgres",
        Host:     "127.0.0.1",
        Port:     5432,
        Username: "postgres",
        Password: "password",
        Database: "app",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    adapter, err := pgmq.NewDatabaseAdapter(db)
    if err != nil {
        log.Fatal(err)
    }

    queue, err := pgmq.NewQueue[map[string]string](context.Background(), adapter, "orders")
    if err != nil {
        log.Fatal(err)
    }

    id, err := queue.Send(context.Background(), map[string]string{"order_id": "123"}, 0)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("sent: %d", id)
}
```

## 🧩 消费者示例

```go
err := queue.Consume(context.Background(), func(ctx context.Context, msg *pgmq.Message[map[string]string]) error {
    // 处理逻辑
    return nil
})
if err != nil {
    log.Fatal(err)
}
```

## ⚙️ 配置说明

```go
queue, err := pgmq.NewQueue[map[string]string](ctx, adapter, "orders",
    pgmq.WithCheckExtension(true),
    pgmq.WithEnsureQueue(true),
    pgmq.WithReadQuantity(1),
    pgmq.WithVisibilityTimeout(30*time.Second),
    pgmq.WithRetryConfig(pgmq.RetryConfig{
        MaxRetries:    5,
        InitialDelay:  2 * time.Second,
        MaxDelay:      5 * time.Minute,
        BackoffFactor: 2,
        Jitter:        true,
    }),
    pgmq.WithConsumerConfig(pgmq.ConsumerConfig{Concurrency: 4}),
)
```

## 📌 目录结构

- `config.go`：默认值、校验与重试/消费配置
- `options.go`：Option 注入
- `pgmq.go`：Queue 管理与核心 API
- `adapter_db.go`：DB 适配
- `errors.go`：错误定义
