# AI Capabilities 文档设计

## 目标

为 go-kit 建立完整的 AI 使用文档体系，使 AI 工具（Claude Code、Cursor 等）能正确理解和使用 go-kit 的能力。

## 架构设计

### 1. 文件结构

```
go-kit (源码库)
├── AGENTS.md                    # AI 开发规范（已存在）
├── CLAUDE.md                    # Claude 入口（已存在）
└── .ai/
    └── capabilities.yaml        # 【新增】完整能力清单

消费项目 (由 gokit CLI 生成)
├── AGENTS.md                    # 【生成】指向 .go-kit/GUIDE.md
└── .go-kit/
    ├── GUIDE.md                 # 【生成】简化版使用指南
    └── capabilities.yaml        # 【生成】仅包含选用功能
```

### 2. 能力清单格式 (capabilities.yaml)

```yaml
version: "1.0.0"
updated_at: "2026-03-05"

capabilities:
  - name: kit
    description: 日志和基础工具
    import: github.com/tsopia/go-kit/kit
    scenarios:
      - name: 打印日志
        snippet: kit.Info(ctx, "message", "key", value)
      - name: 调试日志
        snippet: kit.Debug(ctx, "debug info")
    dependencies: []

  - name: database
    description: 数据库连接管理（GORM 封装）
    import: github.com/tsopia/go-kit/database
    scenarios:
      - name: 初始化数据库
        snippet: |
          db, err := database.New(cfg)
          if err != nil {
              return fmt.Errorf("init db: %w", err)
          }
      - name: 获取连接
        snippet: db := database.GetClient()
    dependencies: [kit]
```

### 3. 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 包名，与目录名一致 |
| `description` | string | 一句话描述功能 |
| `import` | string | 完整 import 路径 |
| `scenarios` | array | 使用场景列表 |
| `scenarios[].name` | string | 场景名称 |
| `scenarios[].snippet` | string | 代码示例 |
| `dependencies` | array | 依赖的其他 go-kit 包 |

### 4. 与 CLI 的集成

#### gokit new 命令行为
```bash
# 创建项目时根据选用的功能生成对应的 capabilities.yaml
gokit new user-api --features=kit,database,cfg
```

生成内容：
- 仅包含 `kit`、`database`、`cfg` 三个能力
- 自动解析并包含依赖（如 database 依赖 kit，已包含则跳过）

#### gokit init 命令行为
```bash
# 为现有项目添加 go-kit 支持，检测已有依赖并生成对应配置
gokit init
```

#### gokit update 命令行为
```bash
# 从 go-kit 源头更新 capabilities
gokit update
```

### 5. 包含的能力清单

| 包名 | 描述 | 状态 |
|------|------|------|
| kit | 日志和基础工具 | 已实现 |
| database | 数据库连接管理 | 已实现 |
| cfg | 配置管理 | 已实现 |
| config | 配置管理（旧包）| 已弃用 |
| pgmq | 消息队列 | 待添加 |
| httpclient | HTTP 客户端 | 已实现 |
| httpserver | HTTP 服务器 | 已实现 |
| errors | 错误处理 | 已实现 |
| llm | 大模型客户端 | 已实现 |

## 实现范围

### P0 - 核心能力清单
- [ ] 创建 `.ai/capabilities.yaml` 包含所有已实现包
- [ ] 更新 CLI 从文件读取（当前是 embedded）

### P1 - CLI 集成
- [ ] `gokit new` 根据 `--features` 生成子集
- [ ] `gokit init` 检测 go.mod 生成对应配置
- [ ] `gokit update` 同步最新能力清单

### P2 - 增强功能
- [ ] 依赖自动解析（A 依赖 B，选 A 自动包含 B）
- [ ] 版本兼容性检查

## 决策记录

### 为什么选择集中式（而非每个包分散）？

| 考虑因素 | 集中式 | 分散式 |
|---------|--------|--------|
| 维护成本 | 低（一处修改） | 高（多处同步） |
| AI 读取效率 | 高（单文件） | 低（遍历多个） |
| 与 CLI 集成 | 简单（一份数据源） | 复杂（合并多个） |
| 包数量 | ~10 个，可控 | 不适合大量包 |

### 为什么消费项目需要子集而非全集？

1. **精确性**：AI 只看到实际可用的功能
2. **避免干扰**：不提示未导入的包
3. **性能**：减少 AI 上下文大小

## 下一步

调用 `superpowers:writing-plans` skill 创建详细实现计划。
