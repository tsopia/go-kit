# PGMQ Package

基于 PostgreSQL `pgmq` 扩展的队列封装，提供泛型消息结构、标准 PGMQ 命名以及带重试与 DLQ 的消费能力。

## ✨ 特性

- **泛型消息**：`Message[T]` 自动完成 JSON 编解码。
- **标准 API**：`Send` / `SendBatch` / `Read` / `Pop` / `Archive` / `Delete` / `Drop`。
- **自动校验**：初始化时检测 `pgmq` 扩展，可自动创建队列。
- **并发消费者**：支持协程池与渐进式重试、DLQ 自动转移。
- **可观测性**：可插拔 Metrics 接口。

## ✅ 依赖与初始化要求

### pgmq 扩展

- 必须预先安装扩展：
  ```sql
  CREATE EXTENSION IF NOT EXISTS pgmq;
  ```
  或使用内置方法：
  ```go
  if err := pgmq.CreateExtension(ctx, adapter); err != nil {
      log.Fatal(err)
  }
  ```

## 🚀 SDK 风格快速开始

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

    adapter, err := pgmq.NewAdapter(context.Background(), db)
    if err != nil {
        log.Fatal(err)
    }

    _, err = pgmq.Configure(adapter)
    if err != nil {
        log.Fatal(err)
    }

    id, err := pgmq.SendMsg(context.Background(), "orders", map[string]string{"order_id": "123"})
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("sent: %d", id)
}
```

## 🔌 单一连接串接入（推荐）

```go
adapter, err := pgmq.NewAdapter(ctx, "postgres://user:pass@localhost:5432/app")
if err != nil {
    log.Fatal(err)
}

_, err = pgmq.Configure(adapter)
if err != nil {
    log.Fatal(err)
}

queue, err := pgmq.NewQueue[map[string]string](ctx, adapter, "orders")
if err != nil {
    log.Fatal(err)
}
```

## 🧩 消费者示例

```go
consumer, err := queue.StartConsumer(context.Background(), func(ctx context.Context, msg *pgmq.Message[map[string]string]) error {
    // 处理逻辑
    return nil
})
if err != nil {
    log.Fatal(err)
}
defer consumer.Stop(context.Background())
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
    pgmq.WithConsumerConfig(pgmq.ConsumerConfig{
        VisibilityTimeout: 30 * time.Second,
        PollInterval:      200 * time.Millisecond,
        MaxConcurrency:    4,
    }),
)
```

## 🧰 SDK 快捷方法示例

```go
_, err := pgmq.Configure(adapter,
    pgmq.WithEnsureQueue(true),
)
if err != nil {
    log.Fatal(err)
}

id, err := pgmq.SendMsg(ctx, "orders", map[string]string{"order_id": "1"})
if err != nil {
    log.Fatal(err)
}
log.Printf("sent: %d", id)
```

## 📦 批量发送示例

```go
ids, err := queue.SendBatch(ctx, []map[string]string{
    {"order_id": "1"},
    {"order_id": "2"},
}, 0)
if err != nil {
    log.Fatal(err)
}
log.Printf("sent batch: %v", ids)
```

## 📌 目录结构

- `config.go`：默认值、校验与重试/消费配置
- `options.go`：Option 注入
- `queue.go`：Queue 管理与核心 API
- `client.go`：SDK Client 与全局实例管理
- `helpers.go`：SDK 快捷方法
- `consumer.go`：Consumer 生命周期管理
- `types.go`：基础类型与消息结构
- `adapter_db.go`：DB 适配
- `errors.go`：错误定义

## 🔒 API 稳定性约定

- `Configure`、`GetClient` 以及 `SendMsg/ReadMsg/...` 等 SDK 快捷函数属于公开 API，优先保证签名和语义稳定。
- 内部可通过提取公共流程减少重复代码，但不改变缺省 `Client` 解析和错误返回行为。
