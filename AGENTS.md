# Go-Kit 开发规范与 AI 指南

## 角色与目标

你是 go-kit 工具库的开发专家。go-kit 是一个为公司内部项目提供基础能力的 Go 工具库，包含日志、数据库、消息队列、HTTP、配置管理等组件。

## 开发规范

### 代码风格
- 所有 `error` 必须显式处理（禁止 `_` 忽略）
- 使用 `fmt.Errorf("context: %w", err)` 包装错误
- 导出用 `PascalCase`，内部用 `camelCase`，缩写全大写（`JSON`, `API`, `ID`）
- 优先使用指针接收者 `(s *Service)`
- 必须支持 `context.Context` 传播

### 测试要求
- 强制使用 **Table-driven tests**
- 测试文件与被测代码同级目录，命名 `*_test.go`
- 优先通过 Interface 抽象进行 Mock

### 构建命令
```bash
go mod tidy          # 安装依赖
go build ./...       # 构建
go test -v ./...     # 测试
golangci-lint run    # 代码检查
```

### 包封装架构（SDK 风格）
- 包内维护未导出的全局 `_client *Client`
- 通过 `Configure(...)` 初始化，通过 `GetClient()` 获取
- 高层函数支持可选 `*Client` 参数：`func Do(ctx context.Context, ..., c ...*Client)`
- 未配置时返回 `ErrMissingClient`
- `Client` 只持配置，业务逻辑在 `Manager/Queue` 或工具函数中

### 目录结构
```
database/
├── config.go          # 配置结构与默认化/校验
├── options.go         # 选项定义
├── errors.go          # 包级错误
├── database.go        # 管理器/入口
├── connector.go       # 连接器组件
├── executor.go        # 执行器组件
├── health.go          # 健康检查组件
├── api.go             # 对外高层 API
├── README.md          # 使用说明
└── examples/          # 使用示例
```

## 新能力开发工作流（强制）

当开发任何新能力、新包或新功能时，AI 必须按以下顺序执行，**禁止跳过任何步骤**：

### 触发条件
以下情况必须执行完整工作流：
- 新建包/模块目录
- 添加公开 API / 导出函数
- 修改包的主要职责或用途
- 添加新的使用场景

### 强制步骤

**步骤 1：首先更新能力清单（必须最先做）**
- [ ] 读取 `.ai/capabilities.yaml` 了解现有能力格式
- [ ] 添加新能力定义到 capabilities 列表
- [ ] 验证 YAML 格式正确

**步骤 2：开发包代码**
- [ ] 创建包目录和代码文件
- [ ] 创建 `doc.go` 包含 AI 使用提示
- [ ] 创建 `README.md` 说明角色、依赖、初始化方式
- [ ] 覆盖关键路径的测试用例
- [ ] 遵循 SDK 封装规范（Configure + Helper 模式）

**步骤 3：更新关联文档**
- [ ] 更新本文档中的"库能力速查表"
- [ ] 创建 `.ai-snippet.md` 描述典型使用场景（可选）

**步骤 4：验证**
- [ ] 运行 `gokit list` 确认新能力已显示
- [ ] 运行 `go test ./...` 确保测试通过

---

## 新包开发检查清单

⚠️ **警告：不要先写代码！按以下顺序执行：**

1. **首先（必须）**: 更新 `.ai/capabilities.yaml` 添加能力定义
2. 创建包目录和代码
3. 创建 `doc.go` 包含 AI 使用提示
4. 创建 `README.md` 说明角色、依赖、初始化方式
5. 覆盖关键路径的测试用例
6. 遵循 SDK 封装规范（Configure + Helper 模式）
7. **最后**: 更新本文档中的"库能力速查表"

## 能力清单规范

每个新包必须更新 `.ai/capabilities.yaml`：

```yaml
- name: 包名（与目录名一致）
  description: 一句话描述功能
  import: 完整 import 路径
  scenarios:
    - name: 使用场景名称
      snippet: 单行或简短代码示例
  dependencies: [依赖的其他 go-kit 包]
```

**示例**：

```yaml
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
    - name: 获取连接
      snippet: db := database.GetClient()
  dependencies: [kit]
```

## 库能力速查（给使用方 AI）

| 场景 | 使用包 | 典型调用 |
|------|--------|----------|
| 打印日志 | `kit` | `kit.Info(ctx, "msg", fields...)` |
| 数据库连接 | `database` | `database.New(cfg)` |
| 消息队列 | `pgmq` | `pgmq.New(cfg)` |
| 配置管理 | `cfg` | `cfg.Load(path, &config)` |
| HTTP 服务 | `httpserver` | `httpserver.New(cfg)` |
| HTTP 客户端 | `httpclient` | `httpclient.Get(ctx, url)` |
| 对象存储 | `storage` | `storage.Upload(ctx, "file", reader)` |

## 项目迁移指南

### CLI 工具拆分到独立仓库

当需要将 `cmd/gokit` 拆分为独立项目 `go-kit-cli`：

```bash
# 1. 在 go-kit 根目录执行子树拆分
git subtree split --prefix=cmd/gokit -b go-kit-cli-main

# 2. 推送到新仓库
git push https://github.com/yourcompany/go-kit-cli.git go-kit-cli-main:main

# 3. 本地清理
git branch -D go-kit-cli-main
```

### 多 AI 工具配置

- `AGENTS.md` - 主规范文件（跨工具支持）
- `CLAUDE.md` - Claude Code 专用入口
- `.cursorrules` - Cursor 专用（如有需要）

## AI 文档维护检查清单

**AI 注意：每新增/修改一个包，必须完成以下检查，禁止遗漏：**

- [ ] **必须**: 新包添加时首先更新 `.ai/capabilities.yaml`
- [ ] **必须**: 验证 YAML 格式正确（可用在线 YAML 验证工具）
- [ ] **必须**: 运行 `gokit list` 确认新能力显示
- [ ] **必须**: 更新本文档中的"库能力速查表"

**如果以上任何一项未完成，必须告知用户并优先完成文档更新。**


## 工具库引用

本项目使用 go-kit 提供基础能力，详细指南请参考 [.go-kit/GUIDE.md](.go-kit/GUIDE.md)。
