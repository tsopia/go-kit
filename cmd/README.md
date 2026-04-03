# cmd 目录

包含 go-kit 提供的 CLI 工具，分为两个独立的可执行文件：

| 工具 | 目录 | 用途 |
|------|------|------|
| `gokit` | `cmd/gokit/` | 项目脚手架工具，初始化、创建、管理 go-kit 项目 |
| `gokit-gen` | `cmd/gokit-gen/` | 代码生成工具，数据库迁移与 GORM 模型生成 |

---

## gokit — 项目脚手架

### 安装

```bash
go install github.com/tsoria/go-kit/cmd/gokit@latest
```

### 命令

| 命令 | 说明 |
|------|------|
| `gokit init` | 为现有项目初始化 go-kit 支持 |
| `gokit new [name]` | 创建新项目 |
| `gokit list` | 列出 go-kit 可用能力 |
| `gokit update` | 更新 AI 指南和能力清单 |

### init — 初始化

在已有 Go 项目根目录执行，创建 `.go-kit/` 目录、`AGENTS.md` 等文件。

```bash
gokit init           # 初始化
gokit init --force   # 强制覆盖已存在文件
```

Flags:

| Flag | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `--force` | `-f` | false | 强制覆盖已存在的文件 |

### new — 创建项目

```bash
gokit new myproject                          # 默认 api 模板
gokit new myworker --template worker         # worker 模板
gokit new myproject --module github.com/org/myproject --output ./myapp
```

Flags:

| Flag | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `--template` | `-t` | api | 模板类型：api / worker / cron / library |
| `--module` | `-m` | github.com/yourcompany/\<name\> | Go module 名称 |
| `--output` | `-o` | ./\<name\> | 输出目录 |

### list — 查看能力

```bash
gokit list
```

以表格形式输出 go-kit 提供的所有能力，包含名称、描述和 import 路径。

### update — 更新资源

```bash
gokit update
```

从 go-kit 源同步最新的能力清单（`.go-kit/capabilities.yaml`）和使用指南（`.go-kit/GUIDE.md`）。

---

## gokit-gen — 代码生成

### 安装

```bash
go install github.com/tsoria/go-kit/cmd/gokit-gen@latest
```

### 命令

| 命令 | 说明 |
|------|------|
| `gokit-gen sync` | 执行迁移 + 生成模型（推荐） |
| `gokit-gen migrate up` | 执行待处理的迁移 |
| `gokit-gen migrate down` | 回退一个迁移版本 |
| `gokit-gen migrate status` | 查看迁移状态 |
| `gokit-gen gen` | 从数据库生成 GORM 模型 |

### Flags

| Flag | 说明 |
|------|------|
| `--dsn` | 数据库 DSN（跳过自动发现） |
| `--driver` | 驱动：mysql / postgres |
| `--migrate-source` | 迁移文件目录（默认 migrations） |
| `--out` | 输出目录（默认 internal/dal） |
| `--tables` | 指定生成的表（逗号分隔） |

### 示例

```bash
# 一键同步
gokit-gen sync

# 指定输出目录
gokit-gen sync --out ./pkg/model

# 指定表和连接
gokit-gen gen --tables user,order --driver mysql --dsn "root:pass@tcp(localhost:3306)/mydb"
```

详细文档见 [gokit-gen/README.md](./gokit-gen/README.md)。
