# AI Capabilities 文档实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 创建完整的 AI 能力文档体系：根目录 `.ai/capabilities.yaml` 包含所有 go-kit 能力定义，供 gokit CLI 读取和生成到消费项目。

**Architecture:** 采用集中式能力清单，单文件维护所有包的 AI 使用信息。CLI 通过 `go:embed` 读取此文件，在项目生成时根据选用的功能筛选子集写入消费项目。

**Tech Stack:** Go 1.24, YAML, go:embed

---

## 前置条件

- 项目结构已存在（cmd/gokit CLI 已实现基础功能）
- 已阅读 `docs/plans/2026-03-05-ai-capabilities-design.md`
- 已阅读 `docs/plans/2026-03-04-gokit-cli-design.md`

---

## Task 1: 创建根目录 .ai 目录结构

**Files:**
- Create: `.ai/capabilities.yaml`

**Step 1: 创建目录**

Run: `mkdir -p /root/projects/go-kit/.ai`
Expected: 目录创建成功

**Step 2: 创建 capabilities.yaml 框架**

```yaml
version: "1.0.0"
updated_at: "2026-03-05"

capabilities: []
```

**Step 3: Commit**

```bash
git add .ai/
git commit -m "feat(ai): create capabilities.yaml framework"
```

---

## Task 2: 添加 kit 包能力定义

**Files:**
- Modify: `.ai/capabilities.yaml`

**Step 1: 添加 kit 能力到 capabilities.yaml**

在 `capabilities: []` 中添加：

```yaml
capabilities:
  - name: kit
    description: 日志和基础工具（Zap 封装，支持结构化日志和追踪）
    import: github.com/tsopia/go-kit/kit
    scenarios:
      - name: 打印信息日志
        snippet: kit.Info(ctx, "message", "key", value)
      - name: 打印调试日志
        snippet: kit.Debug(ctx, "debug info")
      - name: 打印错误日志
        snippet: kit.Error(ctx, "error occurred", "error", err)
      - name: 创建带追踪的上下文
        snippet: ctx = kit.WithTrace(ctx, "operation-name")
      - name: 获取追踪 ID
        snippet: traceID := kit.TraceID(ctx)
    dependencies: []
```

**Step 2: 验证 YAML 格式**

Run: `cd /root/projects/go-kit && cat .ai/capabilities.yaml | head -20`
Expected: 显示正确的 YAML 内容

**Step 3: Commit**

```bash
git add .ai/capabilities.yaml
git commit -m "feat(ai): add kit capability definition"
```

---

## Task 3: 添加 database 包能力定义

**Files:**
- Modify: `.ai/capabilities.yaml`

**Step 1: 添加 database 能力**

在 capabilities 列表末尾追加：

```yaml
  - name: database
    description: 数据库连接管理（GORM 封装，支持 MySQL/PostgreSQL/SQLite）
    import: github.com/tsopia/go-kit/database
    scenarios:
      - name: 初始化数据库连接
        snippet: |
          db, err := database.New(cfg)
          if err != nil {
              return fmt.Errorf("init db: %w", err)
          }
      - name: 获取全局数据库客户端
        snippet: db := database.GetClient()
      - name: 执行带重试的查询
        snippet: |
          err := database.QueryWithRetry(ctx, func(db *gorm.DB) error {
              return db.First(&user, id).Error
          })
      - name: 健康检查
        snippet: |
          if err := database.HealthCheck(ctx); err != nil {
              return fmt.Errorf("db unhealthy: %w", err)
          }
    dependencies: [kit]
```

**Step 2: Commit**

```bash
git add .ai/capabilities.yaml
git commit -m "feat(ai): add database capability definition"
```

---

## Task 4: 添加 cfg 包能力定义

**Files:**
- Modify: `.ai/capabilities.yaml`

**Step 1: 添加 cfg 能力**

```yaml
  - name: cfg
    description: 配置管理（Viper 封装，支持 YAML/JSON/环境变量）
    import: github.com/tsopia/go-kit/cfg
    scenarios:
      - name: 加载配置文件
        snippet: |
          var config AppConfig
          if err := cfg.Load("config.yaml", &config); err != nil {
              return fmt.Errorf("load config: %w", err)
          }
      - name: 从环境变量加载
        snippet: |
          cfg.SetEnvPrefix("MYAPP")
          err := cfg.Load("", &config)
      - name: 获取配置值
        snippet: port := cfg.GetInt("server.port")
    dependencies: []
```

**Step 2: Commit**

```bash
git add .ai/capabilities.yaml
git commit -m "feat(ai): add cfg capability definition"
```

---

## Task 5: 添加 httpclient 包能力定义

**Files:**
- Modify: `.ai/capabilities.yaml`

**Step 1: 添加 httpclient 能力**

```yaml
  - name: httpclient
    description: HTTP 客户端（支持重试、熔断、调试）
    import: github.com/tsopia/go-kit/httpclient
    scenarios:
      - name: 发送 GET 请求
        snippet: |
          resp, err := httpclient.Get(ctx, "https://api.example.com/data")
          if err != nil {
              return fmt.Errorf("fetch data: %w", err)
          }
      - name: 发送 POST 请求
        snippet: |
          resp, err := httpclient.PostJSON(ctx, url, requestBody)
      - name: 创建带配置的客户端
        snippet: |
          client := httpclient.New(httpclient.Config{
              Timeout: 30 * time.Second,
              Retries: 3,
          })
    dependencies: [kit]
```

**Step 2: Commit**

```bash
git add .ai/capabilities.yaml
git commit -m "feat(ai): add httpclient capability definition"
```

---

## Task 6: 添加 httpserver 包能力定义

**Files:**
- Modify: `.ai/capabilities.yaml`

**Step 1: 添加 httpserver 能力**

```yaml
  - name: httpserver
    description: HTTP 服务器（Gin 封装，支持中间件链）
    import: github.com/tsopia/go-kit/httpserver
    scenarios:
      - name: 创建服务器
        snippet: |
          srv := httpserver.New(httpserver.Config{
              Port: 8080,
          })
      - name: 注册路由
        snippet: |
          srv.GET("/health", func(c *gin.Context) {
              c.JSON(200, gin.H{"status": "ok"})
          })
      - name: 启动服务器
        snippet: |
          if err := srv.Start(); err != nil {
              log.Fatal(err)
          }
    dependencies: [kit]
```

**Step 2: Commit**

```bash
git add .ai/capabilities.yaml
git commit -m "feat(ai): add httpserver capability definition"
```

---

## Task 7: 添加 errors 包能力定义

**Files:**
- Modify: `.ai/capabilities.yaml`

**Step 1: 添加 errors 能力**

```yaml
  - name: errors
    description: 错误处理（统一错误码、错误包装）
    import: github.com/tsopia/go-kit/errors
    scenarios:
      - name: 创建错误
        snippet: err := errors.New(errors.CodeInvalidParam, "参数错误")
      - name: 包装错误
        snippet: err := errors.Wrap(err, errors.CodeInternal, "数据库查询失败")
      - name: 判断错误码
        snippet: |
          if errors.IsCode(err, errors.CodeNotFound) {
              // 处理 404
          }
    dependencies: []
```

**Step 2: Commit**

```bash
git add .ai/capabilities.yaml
git commit -m "feat(ai): add errors capability definition"
```

---

## Task 8: 添加 llm 包能力定义

**Files:**
- Modify: `.ai/capabilities.yaml`

**Step 1: 添加 llm 能力**

```yaml
  - name: llm
    description: 大模型客户端统一封装（Eino 兼容）
    import: github.com/tsopia/go-kit/llm
    scenarios:
      - name: 创建 LLM 客户端
        snippet: |
          client, err := llm.New(llm.Config{
              Provider: llm.ProviderOpenAI,
              APIKey:   os.Getenv("OPENAI_API_KEY"),
          })
      - name: 发送聊天请求
        snippet: |
          resp, err := client.Chat(ctx, llm.ChatRequest{
              Messages: []llm.Message{
                  {Role: llm.RoleUser, Content: "Hello"},
              },
          })
      - name: 流式响应
        snippet: |
          stream, err := client.ChatStream(ctx, request)
          for chunk := range stream {
              fmt.Print(chunk.Content)
          }
    dependencies: [kit]
```

**Step 2: Commit**

```bash
git add .ai/capabilities.yaml
git commit -m "feat(ai): add llm capability definition"
```

---

## Task 9: 更新 CLI 读取根目录的 capabilities.yaml

**Files:**
- Modify: `cmd/gokit/pkg/gokit/capability.go`

**Step 1: 修改读取逻辑**

当前代码从 `embed.FS` 读取，需要改为优先读取项目根目录的文件（如果存在），否则使用 embedded。

```go
package gokit

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

//go:embed capabilities.yaml
capabilitiesFS embed.FS

const CapabilitiesPath = ".ai/capabilities.yaml"

type Scenario struct {
	Name    string `yaml:"name"`
	Snippet string `yaml:"snippet"`
}

type Capability struct {
	Name         string     `yaml:"name"`
	Description  string     `yaml:"description"`
	Import       string     `yaml:"import"`
	Scenarios    []Scenario `yaml:"scenarios"`
	Dependencies []string   `yaml:"dependencies"`
}

type CapabilityRegistry struct {
	Version      string       `yaml:"version"`
	UpdatedAt    string       `yaml:"updated_at"`
	Capabilities []Capability `yaml:"capabilities"`
}

// LoadCapabilities 从项目根目录读取，如果不存在则使用 embedded
func LoadCapabilities() ([]Capability, error) {
	// 先尝试从项目根目录读取
	if data, err := os.ReadFile(CapabilitiesPath); err == nil {
		return parseCapabilities(data)
	}

	// 使用 embedded 版本
	data, err := capabilitiesFS.ReadFile("capabilities.yaml")
	if err != nil {
		return nil, fmt.Errorf("read embedded capabilities: %w", err)
	}

	return parseCapabilities(data)
}

func parseCapabilities(data []byte) ([]Capability, error) {
	var registry CapabilityRegistry
	if err := yaml.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("parse capabilities: %w", err)
	}

	return registry.Capabilities, nil
}

func GetCapability(name string) (*Capability, error) {
	caps, err := LoadCapabilities()
	if err != nil {
		return nil, err
	}

	for _, c := range caps {
		if c.Name == name {
			return &c, nil
		}
	}

	return nil, fmt.Errorf("capability not found: %s", name)
}
```

**Step 2: 测试读取**

Run: `cd /root/projects/go-kit/cmd/gokit && go test ./pkg/gokit/... -v`
Expected: PASS

**Step 3: Commit**

```bash
git add cmd/gokit/pkg/gokit/capability.go
git commit -m "feat(cli): support loading capabilities from project root"
```

---

## Task 10: 验证完整能力清单

**Files:**
- None (验证任务)

**Step 1: 运行 list 命令**

Run: `cd /root/projects/go-kit/cmd/gokit && go run main.go list`
Expected: 显示所有能力（kit, database, cfg, httpclient, httpserver, errors, llm）

**Step 2: 验证输出格式**

Expected output:
```
NAME         DESCRIPTION                                    IMPORT
kit          日志和基础工具（Zap 封装...                     github.com/tsopia/go-kit/kit
database     数据库连接管理（GORM 封装...                     github.com/tsopia/go-kit/database
...
```

**Step 3: 提交验证结果**

```bash
git add -A
git commit -m "feat(ai): complete all capability definitions"
```

---

## Task 11: 更新 AGENTS.md 检查清单

**Files:**
- Modify: `AGENTS.md`

**Step 1: 在 AGENTS.md 底部添加**

```markdown
## AI 文档维护检查清单

- [ ] 新包添加时更新 `.ai/capabilities.yaml`
- [ ] 验证 YAML 格式正确
- [ ] 运行 `gokit list` 确认新能力显示
- [ ] 更新此文档中的库能力速查表
```

**Step 2: Commit**

```bash
git add AGENTS.md
git commit -m "docs: add AI documentation maintenance checklist"
```

---

## 完成后的验证清单

- [ ] `.ai/capabilities.yaml` 包含 7 个包的能力定义
- [ ] 每个能力有完整的 scenarios 和 dependencies
- [ ] `gokit list` 正确显示所有能力
- [ ] YAML 格式验证通过
- [ ] 所有修改已提交到 git

---

## 后续工作（不在本计划内）

1. **P1** - `gokit new --features` 支持生成子集
2. **P1** - `gokit init` 检测现有依赖
3. **P2** - 依赖自动解析（选 database 自动包含 kit）
