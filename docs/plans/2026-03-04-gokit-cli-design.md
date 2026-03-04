# Go-Kit CLI 脚手架设计方案

## 背景与目标

go-kit 是一个为公司内部项目提供基础能力的 Go 工具库。为了让 AI 模型（如 Claude Code）能正确使用这个库，需要：

1. 让 AI 知晓 go-kit 提供的能力
2. 让 AI 知道在特定场景下应该使用什么 API
3. 简化新项目接入和现有项目集成的流程

## 设计方案

### 1. 总体架构

```
┌─────────────────────────────────────────────────────────────┐
│                      消费项目 (Consumer Project)              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  AGENTS.md (AI 使用指南)                             │   │
│  │  "参考 .go-kit/GUIDE.md 使用 go-kit 能力"            │   │
│  └─────────────────────────────────────────────────────┘   │
│                         ↓                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  .go-kit/GUIDE.md (自动同步的 AI 使用指南)            │   │
│  │  .go-kit/capabilities.yaml (能力清单)               │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              ↑
┌─────────────────────────────────────────────────────────────┐
│                      go-kit (Library)                       │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  .ai/GUIDE.md (规范源头)                             │   │
│  │  .ai/capabilities.yaml (能力清单)                    │   │
│  │  .ai/snippets/*.md (各包详细指南)                    │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  cmd/gokit (CLI 工具)                               │   │
│  │    - new: 创建新项目                                │   │
│  │    - init: 为现有项目添加 go-kit                    │   │
│  │    - add: 添加特定功能                              │   │
│  │    - update: 更新 AI 指南                           │   │
│  │    - list: 列出可用能力                             │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### 2. AI 配置策略

采用多文件策略，兼容不同 AI 工具：

| 文件 | 用途 |
|------|------|
| `AGENTS.md` | 主规范文件，跨工具支持（Claude Code、Codex、Windsurf） |
| `CLAUDE.md` | Claude Code 入口，指向 AGENTS.md |
| `.cursorrules` | Cursor 专用（如有需要） |

### 3. 能力清单（Capability Registry）

集中式能力清单文件 `.ai/capabilities.yaml`：

```yaml
version: "1.0.0"
updated_at: "2026-03-04"

capabilities:
  - name: kit
    description: 日志和基础工具
    import: github.com/yourcompany/go-kit/kit
    scenarios:
      - name: 打印日志
        snippet: kit.Info(ctx, "message", "key", value)
      - name: 调试日志
        snippet: kit.Debug(ctx, "debug info")
    dependencies: []

  - name: database
    description: 数据库连接管理（GORM 封装）
    import: github.com/yourcompany/go-kit/database
    scenarios:
      - name: 初始化数据库
        snippet: |
          db, err := database.New(cfg)
          if err != nil {
              return fmt.Errorf("init db: %w", err)
          }
    dependencies: [kit]
```

**新包开发要求**：更新包的同时必须更新 `capabilities.yaml`。

### 4. CLI 命令设计

```bash
# 创建新项目
gokit new user-service                    # 默认 api 模板
gokit new order-worker --template=worker  # worker 模板
gokit new report-cron --template=cron     # 定时任务模板

# 为现有项目添加 go-kit
gokit init

# 添加特定功能
gokit add database    # 添加 database 支持
gokit add pgmq        # 添加消息队列支持
gokit add kit         # 添加日志工具

# 更新 AI 指南
gokit update

# 列出可用能力
gokit list
```

### 5. 项目结构

```
cmd/gokit/
├── main.go                      # CLI 入口
├── cmd/
│   ├── new.go                   # new 命令
│   ├── init.go                  # init 命令
│   ├── add.go                   # add 命令
│   ├── update.go                # update 命令
│   └── list.go                  # list 命令
├── pkg/
│   ├── scaffold/
│   │   ├── project.go           # 项目结构生成
│   │   ├── detector.go          # 现有项目检测
│   │   └── modifier.go          # go.mod 修改器
│   └── template/
│       ├── engine.go            # 模板渲染引擎
│       └── loader.go            # 模板加载器
│   └── gokit/
│       ├── version.go           # 版本信息
│       └── capabilities.go      # 能力清单读取
└── templates/                   # 内置模板
    ├── common/
    │   ├── .gitignore
    │   ├── .golangci.yml
    │   ├── Dockerfile
    │   └── Makefile
    ├── api/
    │   ├── cmd/{{.Name}}/main.go.tpl
    │   ├── internal/
    │   │   ├── handler/handler.go.tpl
    │   │   ├── service/service.go.tpl
    │   │   └── config/config.go.tpl
    │   ├── configs/config.yaml
    │   ├── go.mod.tpl
    │   ├── README.md.tpl
    │   └── AGENTS.md.tpl
    ├── worker/
    ├── cron/
    └── library/
```

### 6. 模板变量

```go
type TemplateData struct {
    Name         string   // 项目名称
    Module       string   // go module 名称
    GoKitModule  string   // go-kit 模块路径
    GoKitVersion string   // go-kit 版本
    Features     []string // 启用的功能
}
```

### 7. 项目迁移策略

当需要将 `cmd/gokit` 拆分为独立项目 `go-kit-cli`：

```bash
# 1. 在 go-kit 根目录执行子树拆分
git subtree split --prefix=cmd/gokit -b go-kit-cli-main

# 2. 推送到新仓库
git push https://github.com/yourcompany/go-kit-cli.git go-kit-cli-main:main

# 3. 本地清理
git branch -D go-kit-cli-main
```

迁移后：
- `go-kit-cli` 通过 `go.mod` 依赖 `go-kit`
- 模板中的 import 路径自动替换
- 共享代码提取到 `go-kit/scaffold` 包

## 实现优先级

1. **P0** - 基础 CLI 框架 (`main.go`, `cmd/` 结构)
2. **P0** - `new` 命令（api 模板）
3. **P1** - `init` 命令
4. **P1** - `list` 命令（读取 capabilities.yaml）
5. **P2** - `add` 命令
6. **P2** - `update` 命令
7. **P2** - 其他模板（worker, cron, library）

## 下一步行动

1. 使用 `superpowers:writing-plans` skill 创建详细实现计划
2. 开始 CLI 开发
