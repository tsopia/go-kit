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

## 已知基线问题

- 当前部分环境执行 `go test ./...` 可能因 `github.com/bytedance/sonic@v1.14.1` 编译失败而中断，典型报错为 `undefined: GoMapIterator`
- 若本次改动与 `sonic` 无关，可先使用受影响包的局部测试作为增量验证，并在交付时明确说明全仓测试基线不干净
- 若改动涉及 `config`、JSON 编解码或可能触发 `sonic` 依赖链，必须优先说明该基线问题，不得把失败误报为本次改动引入

### 包封装架构（SDK 风格）
- 包内维护未导出的全局 `_client *Client`
- 通过 `Configure(...)` 初始化，通过 `GetClient()` 获取
- 高层函数支持可选 `*Client` 参数：`func Do(ctx context.Context, ..., c ...*Client)`
- 未配置时返回 `ErrMissingClient`
- `Client` 只持配置，业务逻辑在 `Manager/Queue` 或工具函数中
- 对 `database`、`httpserver` 这类资源型包，允许“显式实例优先，默认实例为辅”；默认实例只作为便利入口，不能替代显式生命周期管理

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
| HTTP 服务 | `httpserver` | `httpserver.NewServer(cfg)` |
| HTTP 中间件 | `httpserver/middleware` | `srv.Use(middleware.Recovery())`, `srv.Use(middleware.AccessLog())`, `srv.Use(middleware.Compression())`, `srv.Use(middleware.ConcurrencyLimit(100))` |
| HTTP 指标 | `httpserver/observability/prometheus` | `prometheusmiddleware.Register(public, prometheusmiddleware.Config{})` |
| HTTP Trace | `httpserver/observability/otel` | `srv.Use(httpotel.Middleware(httpotel.Config{}))` |
| HTTP 默认装配 | `httpserver/preset` | `preset.NewProductionServer(cfg)` |
| HTTP 错误桥接 | `httpserver/integration/errorsx` | `httpserver.WithErrorMapper(errorsx.Mapper())` |
| Swagger 文档 | `httpserver/swagger` | `httpswagger.Register(public, httpswagger.Config{})` |
| HTTP 客户端 | `httpclient` | `httpclient.Get(ctx, url)` |
| 对象存储 | `storage` | `storage.Upload(ctx, "file", reader)`, `storage.AuthorizeDirectUpload(ctx, req)` |
| 加解密/JWT | `crypto` | `crypto.EncryptAES(data)`, `crypto.SignJWT(claims)` |
| LLM Agent | `llm` | `llm.NewAgent(ctx, llm.AgentConfig{...})` |
| LLM 模型创建 | `llm` | `llm.NewModel(ctx, llm.ModelConfig{Protocol, Model, APIKey, Thinking, ExtraFields})` |
| LLM 思考模式 | `llm` | `Thinking: &llm.ThinkingConfig{Enable: true, BudgetTokens: 10000}` |
| LLM ADK Agent | `llm` | `llm.NewADKAgent(ctx, llm.AgentConfig{...})` |
| LLM 并发控制 | `llm` | `Concurrency: llm.ConcurrencyConfig{MaxConcurrency: 3}` |
| LLM Deep Agent | `llm` | `llm.NewDeepAgent(ctx, llm.DeepAgentConfig{...})` |

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

---

## AI 严谨开发协议（使用 Superpowers 时强制生效）

本协议在使用 superpowers:brainstorming / writing-plans / executing-plans 时自动生效。
**所有检查清单必须显式勾选，不得跳过。**

### 全局阅读纪律（任何任务开始前）

```markdown
## 前置检查

- [ ] 已完整阅读相关代码文件（完整文件，非片段）
- [ ] 已完整阅读相关测试文件
- [ ] 已列出代码现状（每个判断附行号）
```

### 置信度标注规范

| 级别 | 定义 | 示例 |
|------|------|------|
| **确定** | 完整阅读源码，行号明确 | `middleware/recovery.go:10-22` |
| **推测** | 基于部分代码或模式推断 | "在 x.go 看到 slog，未全局确认" |
| **未知** | 需要进一步验证 | "未阅读相关代码" |

**规则**：推测和未知必须转化为确定，或已标注风险并获得用户确认。

### Phase 1: Brainstorming（设计）

**强制输入**：
- [ ] 完整阅读相关代码文件
- [ ] 完整阅读相关测试文件
- [ ] 列出至少 2 种替代方案
- [ ] 列出"可能否定本设计的技术约束"

**退出检查**：
- [ ] 所有「推测」和「未知」已转化为「确定」或已标注风险
- [ ] 已列出证伪证据

### Phase 2: Writing-plans（计划）

**文件结构约束**：
- 每个 Task 修改 ≤ 50 行
- 测试代码 : 实现代码 ≥ 1.5 : 1

**Task 模板**：
```markdown
### Task N: [描述]

**目标**：[一句话]

**文件变更**：
- 修改：`path/file.go:10-25`（预计 +X 行）

**约束检查**：
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**子步骤（TDD）**：
- [ ] **Step 1: 写失败测试**（完整代码）
- [ ] **Step 2: 运行测试 → 确认失败**（命令 + 预期输出）
- [ ] **Step 3: 写最简实现**（完整代码，行数检查）
- [ ] **Step 4: 运行测试 → 确认通过**（命令 + 预期输出）
- [ ] **Step 5: Commit**（完整命令）
```

### Phase 3: Execution（执行）

**每个 Task 前置检查**：
```markdown
- [ ] 我已完整阅读本 Task 涉及的所有文件
- [ ] 我理解本 Task 的所有边界条件
- [ ] 本次修改预计 ≤ 50 行（超过已拆分）
- [ ] 我知道验证命令
```

**TDD 强制流程**：
```
Step 1: 写测试 → 测试必须失败
Step 2: 运行 → 确认失败
Step 3: 写实现（最简代码）
Step 4: 运行 → 确认通过
Step 5: 重构（保持通过）
Step 6: Commit
```

**代码自检清单（提交前）**：
```markdown
- [ ] 错误处理完整（所有 error 被检查或显式忽略）
- [ ] 并发安全（共享状态有锁或文档）
- [ ] 边界覆盖（所有 if/else 有测试）
- [ ] 命名一致（符合项目风格）
```

**验证闭环（每个 Task 结束）**：
```markdown
- [ ] 编译：`go build ./...` → PASS
- [ ] 测试：`go test -v ./path` → PASS
- [ ] Lint：`golangci-lint run` → PASS
- [ ] 行数：修改 X 行，测试 Y 行（比例 Y/X ≥ 1.5）
```

### 约束违反处理

如果无法遵守约束：
1. **立即停止执行**
2. 向用户说明：违反了哪条、为什么、建议方案
3. **获得明确同意后再继续**

### 快速参考

| 约束 | 数值 |
|------|------|
| 单次 Task 修改行数 | ≤ 50 |
| 测试:实现代码比例 | ≥ 1.5:1 |
| Task 执行时间 | 2-5 分钟 |
