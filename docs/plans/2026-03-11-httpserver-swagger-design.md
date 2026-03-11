# httpserver Swagger 集成设计

## 背景

当前 `go-kit/httpserver` 已经具备以下能力：

- 基于 Gin 的轻量 HTTP Server 封装
- `RouteModule` 模块化路由注册
- `Handle` / `HandleJSON` typed handler
- `DecodeJSON` / `DecodeQuery` / `DecodeURI` 请求解码
- 生命周期 hooks、优雅关闭、健康检查

这套能力已经足够支撑业务项目编写稳定的 HTTP 接口，但在“接口文档”和“AI coding 生成规范”方面仍有明显缺口：

- 业务项目接入 Swagger 的方式不统一
- AI 生成 HTTP 接口时没有固定模板，容易漏注释、漏状态码、漏文档路由
- 鉴权项目里 Swagger 很容易被全局中间件误伤
- 当前仓库还没有为 `httpserver` 提供 Swagger 接入规范、示例和 AI 友好的使用约定

## 目标

- 为 `httpserver` 提供一套统一的 Swagger 集成方式
- 保持人类开发者使用成本低，优先复用社区主流 `swaggo` 生态
- 为 AI coding 提供稳定模板，统一生成 typed handler、Swagger 注释和路由接入方式
- 明确 Swagger 与鉴权中间件的边界，确保 Swagger 默认公开访问
- 补齐 README、包文档、能力清单和示例，方便项目直接照抄

## 非目标

- 不自研 OpenAPI 生成器
- 不通过结构化 DSL 替代 `swaggo` 注释
- 不强制 `httpserver` 统一项目的响应 envelope
- 不把业务鉴权逻辑内置到 `httpserver/swagger`
- 不在第一阶段引入复杂 CLI 自动化或代码扫描增强

## 方案比较

### 方案 1：纯 `swaggo` 原生接入

项目自行安装并使用：

- `github.com/swaggo/swag/cmd/swag`
- `github.com/swaggo/gin-swagger`
- `github.com/swaggo/files`

优点：

- 接入最快
- 学习成本最低
- 符合 Go 社区常见用法

缺点：

- 项目间风格容易漂移
- AI 容易生成不完整注释
- Swagger 路由挂载方式、鉴权绕过方式不统一

### 方案 2：`swaggo` + `httpserver` 约定封装

底层仍使用 `swaggo`，但由 `httpserver` 提供统一接入方式、注释规范、推荐模板和示例。

优点：

- 保留人类开发者熟悉的 `swaggo` 用法
- AI 可以稳定复用模板
- 能与现有 typed handler 体系自然对齐
- 能系统解决 Swagger 默认公开、路由组织、文档位置和注释约定问题

缺点：

- 仍然依赖注释，不是零注释文档生成

### 方案 3：`swaggo` + `httpserver` 约定封装 + CLI 自动化

在方案 2 的基础上，再通过 `gokit` 或脚本封装 `swag init`、文档输出目录和校验动作。

优点：

- 人工操作更少
- 更适合大规模项目推广

缺点：

- 初始实现范围更大
- 第一阶段收益不如方案 2 明显

## 核心结论

采用方案 2，并为方案 3 预留扩展空间。

也就是：

- 文档生成生态固定为 `swaggo`
- `httpserver` 提供统一接入约定
- Swagger 能力以 `httpserver/swagger` 子包的方式提供
- AI 以后按统一模板生成 `typed handler + swaggo 注释 + public swagger route + protected api group`

该方案兼顾：

- 人类开发者低学习成本
- AI 生成代码的一致性
- `httpserver` 保持工具库定位，不扩张为框架

## 设计原则

### 1. 优先复用成熟生态

生成 Swagger / OpenAPI 文档的职责交给 `swaggo`，`httpserver` 不重复造轮子。

### 2. 以约定代替发明新 DSL

优先通过模板、README、示例和推荐目录收敛写法，而不是强制引入一套新的文档声明 DSL。

### 3. Swagger 默认公开

Swagger UI 和文档接口默认应公开访问，不受业务鉴权中间件影响。

### 4. 保持 `httpserver` 核心克制

`httpserver` 继续负责传输层能力。Swagger 集成属于可选扩展能力，应隔离在子包中。

### 5. AI 模板必须贴近人类常用写法

AI 生成的代码应该是人看到就能维护的普通 Go + Gin + `swaggo` 代码，而不是只对 AI 友好的特殊 DSL。

## 组件设计

### 包形态

新增子包：

`github.com/tsopia/go-kit/httpserver/swagger`

推荐使用方式：

```go
import (
	_ "your/module/internal/docs"

	"github.com/tsopia/go-kit/httpserver"
	httpswagger "github.com/tsopia/go-kit/httpserver/swagger"
)
```

### 核心 API

子包提供一个注册函数：

```go
func Register(r gin.IRoutes, cfg Config)
```

推荐配置结构：

```go
type Config struct {
	Enabled bool
	Path    string
	DocURL  string
}
```

默认值建议：

- `Enabled: true`
- `Path: "/swagger/*any"`
- `DocURL: "doc.json"`

### 设计选择说明

不把 Swagger 直接塞进 `httpserver.Server` 主类型，也不优先设计成 `WithSwagger(...) Option)`，原因是：

- `Server` 当前职责已经比较清晰，主要处理生命周期与路由装配
- Swagger 是可选的文档暴露能力，不应成为所有用户默认依赖
- 独立子包更容易隔离 `gin-swagger` 依赖和概念
- 子包方式更容易表达“把 Swagger 注册到哪个路由组，由调用方决定”

## 路由与鉴权约定

### 默认公开策略

Swagger 默认公开访问。

推荐路由组织方式：

```go
srv := httpserver.NewServer(cfg)

public := srv.Group("")
protected := srv.Group("/api")
protected.Use(AuthMiddleware())

httpswagger.Register(public, httpswagger.Config{
	Path: "/swagger/*any",
})
```

设计约束：

- Swagger 必须注册在公共路由或不受鉴权保护的路由组上
- 鉴权中间件推荐只挂在受保护的路由组上，不挂全局 `Engine`

### 历史项目兼容策略

若历史项目已经采用全局鉴权中间件：

```go
srv.Use(AuthMiddleware())
```

则鉴权中间件必须支持 Swagger 路径白名单，至少跳过：

- `/swagger/`
- `/swagger/index.html`
- `/swagger/doc.json`

但这只作为兼容路径，不作为推荐组织方式。

## 文档生成约定

### 文档生成工具

固定使用 `swaggo`：

- `github.com/swaggo/swag/cmd/swag`
- `github.com/swaggo/gin-swagger`
- `github.com/swaggo/files`

### 输出目录

推荐统一输出到：

`internal/docs`

原因：

- 避免与仓库中的 Markdown 文档目录 `docs/` 混淆
- 生成代码属于运行时导入产物，不属于设计文档目录

### 统一命令

推荐统一命令形态：

```bash
swag init -g cmd/server/main.go -o internal/docs
```

实际入口文件可由业务项目调整，但 `README` 与示例应统一用这套形态表达。

### 主程序导入约定

业务主程序需要显式导入生成文档包：

```go
import _ "your/module/internal/docs"
```

## AI Coding 约定

### 约定 1：注释写在 transport 层 typed handler 上

Swagger 注释应写在被 `httpserver.Handle...` 包装的 transport handler 上，而不是 service 层。

示意：

```go
// login godoc
// @Summary 用户登录
// ...
func (m *UserModule) login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	return m.auth.Login(ctx, req)
}
```

### 约定 2：统一生成模式

AI 生成 HTTP 接口时，默认应生成以下四部分：

- 请求结构体与响应结构体
- typed handler
- `swaggo` 注释
- 路由注册

### 约定 3：Swagger 与业务路由分组

AI 生成 main / router 装配代码时，默认应生成：

- 公共路由组
- 受保护路由组
- Swagger 注册在公共路由组

### 约定 4：注释模板与状态码覆盖

AI 生成注释时，应至少反映当前 `httpserver` 默认行为：

- 解码失败：`400`
- 校验失败：`422`
- 未映射错误：`500`

若接口需要鉴权，再补充：

- `401`
- `403`

## 响应结构约定

本设计不在 `httpserver` 核心中强制统一响应 envelope。

原因：

- 当前 `httpserver` 成功响应默认直接输出业务返回值
- 错误响应默认是轻量 JSON 结构
- `WithEncoder` / `WithErrorMapper` 已允许项目自行约定响应结构

因此：

- Swagger 集成只负责文档能力和写法规范
- 若团队希望统一 `{code,message,data}` 或其他 envelope，应作为项目级 API 规范单独约定
- AI 模板可以引用团队约定，但不由 `httpserver/swagger` 强制实现

## 示例模板

### 主程序模板

```go
package main

import (
	_ "your/module/internal/docs"

	"github.com/tsopia/go-kit/httpserver"
	httpswagger "github.com/tsopia/go-kit/httpserver/swagger"
)

// @title Example API
// @version 1.0
// @description Example service API
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	srv := httpserver.NewServer(nil)

	public := srv.Group("")
	protected := srv.Group("/api/v1")
	protected.Use(AuthMiddleware())

	httpswagger.Register(public, httpswagger.Config{
		Path: "/swagger/*any",
	})

	userModule := NewUserModule()
	userModule.RegisterRoutes(protected)
}
```

### JSON typed handler 模板

```go
type LoginRequest struct {
	Email string `json:"email" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

// login godoc
// @Summary 用户登录
// @Description 使用邮箱登录并返回访问令牌
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录请求"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} map[string]string
// @Failure 422 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auth/login [post]
func (m *UserModule) login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	return m.auth.Login(ctx, req)
}
```

### 鉴权接口模板

```go
// profile godoc
// @Summary 获取当前用户信息
// @Tags user
// @Produce json
// @Security BearerAuth
// @Success 200 {object} ProfileResponse
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/me [get]
func (m *UserModule) profile(ctx context.Context, req ProfileRequest) (ProfileResponse, error) {
	return m.user.Profile(ctx, req)
}
```

### Query 接口模板

```go
type ListUsersRequest struct {
	Page int `form:"page"`
	Size int `form:"size"`
}

// listUsers godoc
// @Summary 用户列表
// @Tags user
// @Produce json
// @Param page query int false "页码"
// @Param size query int false "每页数量"
// @Security BearerAuth
// @Success 200 {object} ListUsersResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users [get]
func (m *UserModule) listUsers(ctx context.Context, req ListUsersRequest) (ListUsersResponse, error) {
	return m.user.List(ctx, req)
}
```

## 需要更新的仓库内容

本次能力实现完成后，至少需要更新：

- `.ai/capabilities.yaml`
- `httpserver/doc.go`
- `httpserver/README.md`
- `AGENTS.md` 中的“库能力速查表”
- `httpserver/swagger` 子包文档与示例

如新增公开子包，也应补充相应测试与能力描述。

## 测试与验收标准

### 测试范围

至少覆盖以下测试：

- `swagger.Register(...)` 能将 UI 路由注册到指定 `gin.IRoutes`
- Swagger 注册到公共路由时不受受保护路由组影响
- 受保护业务路由仍然受鉴权中间件约束
- 示例接入方式能正常访问 Swagger UI
- `gokit list` 可看到更新后的 `httpserver` Swagger 场景

### 验收标准

实现完成后应满足：

- 新项目可按模板快速接入 Swagger UI
- AI 能稳定生成 `typed handler + swaggo 注释 + public swagger route + protected api group`
- Swagger 默认公开，不被鉴权误伤
- 文档生成路径和命令清晰统一
- 文档、示例、能力清单和测试全部更新完成

## 后续扩展

第一阶段完成后，可视需要评估：

- 为 `gokit` 增加 Swagger 生成辅助命令
- 增加更完整的 AI 示例片段
- 提供更细的注释模板，例如 URI 参数、分页、文件上传、多响应示例

但这些都不属于当前设计的必须范围。
