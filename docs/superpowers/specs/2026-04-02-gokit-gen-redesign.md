# gokit-gen 重设计

## 背景

当前 `cmd/gokit-gen` 存在以下问题：
- DSN 拼接时密码特殊字符未编码，导致连接失败
- sync 命令建立两次数据库连接
- ctx 参数未实际传递（误导性签名）
- `--dsn` 强制要求 `--driver`（多余）
- MySQL DSN 解析脆弱（手写 strings.Split）
- 零测试覆盖

## 定位

go-kit 内部小工具，不需过度设计。内部为主，开源为辅。

## 核心原则

**所有入口统一解析成 `*database.Config`，只调用一次 `database.New()`，migrate 用 `SQLDB()`，gen 用 `GetDB()`，彻底移除 DSN 手拼和重复建连。**

## 需求

1. **migrate**：执行 migration 文件（up/down/status/version/force）
2. **gen**：从数据库 schema 生成 GORM model 代码
3. **sync**：先 migrate 再 gen，共享连接

## 支持范围

### 第一阶段支持

- 数据库驱动：MySQL、PostgreSQL、SQLite
- DSN 自动识别：MySQL DSN、PostgreSQL URL DSN（`postgres://` / `postgresql://`）
- SQLite：不要求 `--dsn` 自动识别，支持通过配置文件或源码发现进入

### 暂不支持

- PostgreSQL key-value DSN（如 `host=... port=... user=...`）
- 复杂源码分析（变量拼接、函数返回、跨文件引用）

## 配置获取

### 来源与优先级

1. **命令行 `--dsn`**：直接解析 DSN 字符串，自动推断 driver
2. **配置文件**：用 go-kit 的 `cfg.New()` 加载项目配置，读取 `database` 段
3. **Go 源码发现**（保底）：扫描 `main.go`、`cmd/*/main.go` 中的 `database.Config` 字面量

### 来源 1：`--dsn`

- MySQL: 用 `go-sql-driver/mysql` 的 `ParseDSN` 解析
- PostgreSQL: 用 `net/url` 解析
- `--driver` 可选：传了必须与推断一致，不一致直接报错
- 自动推断：MySQL 看是否含 `tcp(`，PostgreSQL 看 `postgres://`/`postgresql://` 前缀

### 来源 2：配置文件

```go
provider, _ := cfg.New(path...)
provider.Sub("database").Unmarshal(&database.Config{})
```

支持两种形态：

```yaml
# 形态 A：结构化字段
database:
  driver: mysql
  host: 127.0.0.1
  port: 3306
  username: root
  password: xxx
  database: demo

# 形态 B：显式 dsn
database:
  dsn: user:pass@tcp(127.0.0.1:3306)/demo
```

如果 `database.dsn` 存在，优先按 DSN 解析。否则直接 Unmarshal 结构化字段。

### 来源 3：Go 源码发现

只扫描 `database.Config{...}` 直接字面量，只提取六个字段：Driver、Host、Port、Username、Password、Database。

错误语义：
- 找不到：返回"未发现数据库配置"
- 找到多个：返回"发现多个候选配置，请显式指定 --dsn 或 --config"
- 字段缺失：返回"源码发现配置不完整"

## 架构

```
用户输入
  │
  ▼
main.go ─── flag 解析 ─── 命令分发
  │                              │
  │                    ┌─────────┼─────────┐
  │                    ▼         ▼         ▼
  │                migrate     gen       sync
  │                    │         │         │
  ▼                    ▼         ▼         ▼
config.go ◄──────── 建连接（共享）──────────►
  │                    │         │         │
  │                    ▼         ▼         ▼
              dbmigrate   gorm gen  migrate+gen
              (接受 *sql.DB)
```

关键决策：
- 连接只建一次，在命令执行层建好 `*database.Database`
- dbmigrate 接受 `*sql.DB`（通过 `db.SQLDB()`），不再拼 DSN
- gen 接受 `*gorm.DB`（通过 `db.GetDB()`）
- sync 复用同一个 `*database.Database`

## dbmigrate 包

### 接口

```go
package dbmigrate

type Config struct {
    SourcePath string
    DB         *sql.DB
    DriverName string // mysql | postgres | sqlite
}

type Status struct {
    Version uint
    Dirty   bool
}

func Up(ctx context.Context, cfg Config) error
func Down(ctx context.Context, cfg Config) error
func UpTo(ctx context.Context, cfg Config, version uint) error
func DownTo(ctx context.Context, cfg Config, version uint) error
func Version(ctx context.Context, cfg Config) (uint, bool, error)
func Status(ctx context.Context, cfg Config) (Status, error)
func Force(ctx context.Context, cfg Config, version int) error
```

### driver 分发

```go
func openDriver(db *sql.DB, driverName string) (database.Driver, error) {
    switch driverName {
    case "mysql":
        return mysql.OpenFromDB(db)
    case "postgres":
        return postgres.OpenFromDB(db)
    case "sqlite":
        return sqlite3.WithInstance(db, &sqlite3.Config{})
    default:
        return nil, fmt.Errorf("unsupported driver: %s", driverName)
    }
}
```

不再有 `buildDSN()` 函数。用 `migrate.NewWithInstance` 替代 `migrate.New`。

## gen 实现

```go
type genOptions struct {
    DB      *gorm.DB
    OutPath string
    Tables  []string
}

func runGen(ctx context.Context, opts genOptions) (err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("gorm gen failed: %v", r)
        }
    }()
    // ...
}
```

- `Tables` 为空：生成全部表
- 指定表但不存在：报错
- `OutPath` 默认 `internal/model`
- 不硬编码 `ModelPkgPath`
- 不手动 `os.MkdirAll`
- panic 保护，转成 error

## CLI 设计

```bash
gokit-gen migrate --up
gokit-gen migrate --down
gokit-gen migrate --status
gokit-gen migrate --version
gokit-gen gen --out internal/model --tables users,orders
gokit-gen sync --out internal/model
```

### 公共参数

- `--dsn`：数据库 DSN
- `--driver`：数据库驱动（可选，与 DSN 推断一致）
- `--config`：配置文件路径
- `--migration-path`：migration 文件目录（默认 migrations）
- `--out`：输出目录（默认 internal/model）
- `--tables`：逗号分隔的表名

## sync 行为

1. 解析配置
2. `db, err := database.New(cfg)`
3. `sqlDB, _ := db.SQLDB()` → `dbmigrate.Up(...)`
4. `runGen(ctx, genOptions{DB: db.GetDB(), ...})`

失败语义：
- migrate 失败：立即退出，不执行 gen
- gen 失败：返回错误，不回滚 migration
- 退出前统一关闭数据库连接

## 文件结构

```
cmd/gokit-gen/
├── main.go
├── gen.go
├── internal/
│   ├── config.go
│   ├── config_test.go
│   └── discover.go
dbmigrate/
├── migrate.go
└── migrate_test.go
```

## 删除文件

| 文件 | 原因 |
|------|------|
| `cmd/gokit-gen/internal/discover/discover.go` | 用 config.go + discover.go 替代 |
| `cmd/gokit-gen/internal/cmd/common.go` | 连接逻辑内联到 main.go |
| `cmd/gokit-gen/internal/cmd/migrate.go` | 重写 |
| `cmd/gokit-gen/internal/cmd/gen.go` | 重写 |
| `cmd/gokit-gen/internal/cmd/sync.go` | 重写 |
| `cmd/gokit-gen/internal/generator/generator.go` | gen 逻辑内联到 gen.go |
| `dbmigrate/migrate.go` | 重写 |

## 必要的仓库改造

### 修 `database/connector.go` 的 DSN 构造

当前仓库数据库连接层还在手拼 DSN。必须一起改掉：

- `buildMySQLDSN(config)` 改为 `mysql.Config.FormatDSN()`
- PostgreSQL 统一构造方式
- 严禁再用 `fmt.Sprintf` 拼连接串

### 评估添加 `database.Config.DSN`

建议在 `database.Config` 中添加：

```go
DSN string `mapstructure:"dsn" json:"dsn" yaml:"dsn"`
```

规则：`DSN` 有值时优先，但最终仍归一化到 driver + 结构化配置。

## 分阶段实施

### Phase 1：底层能力改造

修 `database/connector.go` 的 DSN 构造，评估添加 `database.Config.DSN`，补数据库连接测试。

### Phase 2：配置解析层

实现 `config.go` + `discover.go` + `config_test.go`。

### Phase 3：dbmigrate 重写

改为 `NewWithInstance`，接受 `*sql.DB + driverName`，补测试。

### Phase 4：gen 与 sync 接入

写 `gen.go` + `main.go` 命令分发 + sync，验证连接只初始化一次。

### Phase 5：删除旧逻辑

删除所有旧文件，前提是新实现已跑通测试。

## 技术约束

- golang-migrate 的每个数据库 driver 各自实现 `database.Driver` 接口，没有通用 `OpenFromDB(*sql.DB)`，需按 driver 名分发
- gorm gen 的 `Execute()` 返回 void，生成失败通过 panic 或 log 输出，需 recover
- MySQL DSN 格式与 PostgreSQL URL 格式差异大，解析需分开处理

## 测试计划

### config_test.go

- MySQL DSN 解析（含特殊字符密码）
- PostgreSQL DSN 解析
- `--dsn + --driver` 一致/不一致
- 配置文件结构化字段 / 显式 dsn
- 源码发现：单候选、多候选报错、缺字段

### migrate_test.go

- unsupported driver
- source path 不存在
- nil db 报错
- 集成测试：MySQL/PostgreSQL/SQLite 的 up/down/version/force

### gen 测试

- 空表列表生成全部
- 指定不存在表时报错
- panic recover 转 error

## 交付标准

- migrate/gen/sync 三个命令可用
- sync 只建立一次数据库连接
- MySQL 特殊字符密码可正常连接
- 配置来源优先级生效
- 不再有手拼 DSN
- 测试覆盖三类配置来源 + driver 分发 + 基本迁移动作
