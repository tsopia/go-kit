# httpserver 能力增强设计

## 背景

当前 `go-kit/httpserver` 的定位是基于 Gin 的轻量 HTTP server 封装，已经具备路由注册、生命周期、优雅关闭、健康检查、Trace ID / Request ID 等基础能力。

现阶段的目标不是把 `httpserver` 做成另一个强约束框架，而是在保留工具库属性的前提下，补足两类能力：

- 让业务项目更容易接入和复用
- 让 AI coding 在接口开发时有更稳定、更统一的推荐模式

## 目标

- 保持 `httpserver` 作为 HTTP 传输层底座
- 修正当前 API 承诺与实现不闭合的问题
- 提供更稳的生命周期、健康检查、可测试性和扩展点
- 提供一套可选的 typed handler 契约，统一 `decode -> validate -> invoke -> map error -> encode`
- 让项目可以采用推荐目录与模块接入方式，但不被强制绑定

## 非目标

- 不把 `httpserver` 直接扩展成类似 `jzero` 的完整工程体系
- 不在核心包里强制目录结构、Handler/Logic/Model 分层或代码生成
- 不在核心层强依赖 `kit` 日志或任何具体日志实现
- 不在 `httpserver` 内置依赖注入容器
- 不强制所有接口统一输出 `{code,message,data}` 这类 envelope

## 核心结论

采用“两层能力、一个包内渐进扩展”的方向：

- 核心层继续负责 server lifecycle、middleware、router mounting、graceful shutdown、health check、基础 hooks
- 在同一个 `httpserver` 包内新增一套可选的 typed handler 契约，作为推荐写法服务项目和 AI coding
- 现有 `Engine()`、`Use()`、`GET/POST/...`、`Group()` 等 Gin 直连能力保持兼容

该方案能同时满足：

- 老项目零迁移继续使用 Gin 原生写法
- 新项目逐步切到统一的 typed handler 写法
- `httpserver` 仍然是工具库，而不是接管业务组织方式的框架

## 设计原则

### 1. 核心层保持克制

核心层只处理 HTTP 传输相关能力：

- 监听与关闭
- 中间件挂载
- 路由注册
- 健康检查
- Trace / Request ID
- 生命周期 hook

不处理：

- 业务目录约束
- 业务错误码体系
- 统一业务响应格式
- 代码生成或脚手架

### 2. 可选能力优先于强绑定

所有高层能力都应以可选接口或适配器暴露，而不是强绑实现：

- 日志通过 hook 注入，不依赖 `kit`
- 校验通过约定接口或自定义 validator 接入
- 业务错误通过 mapper 翻译
- 响应输出通过 encoder 控制

### 3. 保留 Gin 直出，补充推荐契约

`httpserver` 不隐藏 Gin，也不强制替换 `gin.Context`。

推荐模式是新增 typed handler 适配层，让项目和 AI 生成接口时优先遵循固定流水线；遇到特殊需求时，仍然允许回退到 Gin 直写模式。

## 核心 API 收敛

### Config

`Config` 只保留真正属于传输层内核的配置：

- `Host`
- `Port`
- `ReadTimeout`
- `WriteTimeout`
- `IdleTimeout`
- `MaxHeaderBytes`
- `ShutdownTimeout`
- `EnableHealthCheck`
- `HealthCheckPath`
- `HealthCheckPort`

额外扩展能力通过 `Option` 模式承载，避免 `Config` 持续膨胀。

### Option

新增 `Option` 模式承载可选扩展能力，例如：

- 生命周期 hooks
- 健康检查管理器
- 默认中间件组合
- 模块注册器
- typed handler 默认配置

### 生命周期

生命周期语义统一为：

- `Run()` 保持阻塞运行
- `Start()` 改为非阻塞启动，但不再 `panic`
- 推荐补充 `Serve(listener net.Listener)` 能力，支持外部 listener / TLS / 特殊部署场景
- `WaitForShutdown()` 和 `RunWithGracefulShutdown()` 不直接打印日志

启动失败、运行中错误、关闭失败都应通过返回值、错误通道或 hook 暴露，而不是直接中断进程。

### 健康检查

`HealthCheckPort` 既然已经是公开配置，就必须闭合语义：

- `0` 表示复用主端口
- 非 `0` 表示启动独立健康检查监听

不再接受“字段存在但实现缺失”的状态。

健康检查仍属于核心层能力，因为它是 server 运行时的一部分。

### 日志与观测扩展点

核心层不依赖 `kit` 日志，也不依赖任何其他日志库。

只提供基础 hook / observer 扩展点，例如：

- `OnStarting`
- `OnStarted`
- `OnServeError`
- `OnShuttingDown`
- `OnShutdownComplete`

项目方自行在 hook 中接入自己的 logger、metrics、trace、审计系统。

## 路由模块化

为了让项目接入更清晰、AI 更容易批量生成模块，新增轻量路由模块接口：

```go
type RouteModule interface {
	RegisterRoutes(r gin.IRoutes)
}
```

并提供批量注册辅助能力，例如：

```go
server.RegisterModules(modules...)
```

这只是统一接入点，不等于强制目录结构。

## AI Coding 友好的 typed handler 契约

### 目标

为接口开发提供一条推荐写法，统一如下流程：

`decode request -> validate -> invoke business -> map error -> encode response`

### 业务函数形态

推荐业务入口函数使用强类型签名：

```go
type HandlerFunc[Req any, Resp any] func(ctx context.Context, req Req) (Resp, error)
```

`httpserver` 提供适配器把它转换成 `gin.HandlerFunc`。

### 可插拔组件

#### Decoder

负责从 HTTP 请求中构造 `Req`，默认提供：

- JSON body decoder
- Query decoder
- URI decoder
- 可组合 decoder

#### Validator

不强依赖具体校验库，默认采用轻量约定：

- 如果 `Req` 实现 `Validate() error`，则自动调用
- 如果 `Req` 实现 `Validate(context.Context) error`，则自动调用
- 如果项目已有自己的 validator，则通过接口适配接入

#### ErrorMapper

负责把传输层错误和领域错误翻译为 HTTP 响应。

默认映射策略保持保守：

- decode / bind 错误 -> `400`
- validate 错误 -> `422`
- 未识别业务错误 -> `500`

业务项目可以替换为自己的错误映射规则。

#### Encoder

负责输出 HTTP 响应。

默认行为是直接编码 JSON，不强制统一 envelope。

如果项目需要 `{code,message,data}` 或其他标准响应结构，可以替换 encoder，而不需要修改业务 handler。

### 契约价值

这套契约的价值在于：

- 项目接口写法更稳定
- AI 生成代码时更容易遵循统一模式
- 业务函数只处理 `context.Context` 和强类型请求，不直接耦合 Gin
- 传输层错误、业务错误、响应编码职责边界更清晰

## 推荐依赖组织方式

依赖组织采用“应用层手动构造函数注入”，不在 `httpserver` 内置 DI 容器。

推荐装配链路：

```text
main
-> 加载配置
-> 初始化 db
-> 初始化 repo
-> 初始化 service
-> 初始化 module
-> 注册到 httpserver
```

推荐调用链路：

```text
HTTP request
-> route
-> typed handler adapter
-> decode
-> validate
-> service/usecase
-> repo
-> db
-> error mapper / encoder
-> HTTP response
```

推荐由 `internal/app` 之类的 composition root 完成依赖装配，再通过构造函数把依赖传给模块、service、repo。

不推荐：

- 在 `httpserver` 中内置容器
- 在 handler 中读取全局 `db` 或全局 `service`

## 推荐项目组织方式

`httpserver` 不强制目录结构，但文档中给出推荐范式，便于项目和 AI 对齐：

- `cmd/<app>/main.go` 负责启动
- `internal/app` 负责 server、配置和依赖装配
- `internal/module/<name>` 负责单个业务模块
- 模块内部按 `handler/service/repo` 或 `transport/usecase/store` 拆分，由项目自己决定

统一的是“模块通过 `RouteModule` 接入 server”，而不是统一业务目录模板。

## 测试策略

测试必须继续遵循 table-driven tests。

重点补充以下覆盖：

- `Start/Run/Serve/Shutdown` 正常与异常路径
- `HealthCheckPort=0` 与独立端口模式
- 生命周期 hook 调用时机
- typed handler 的 decode / validate / error map / encode
- 原生 Gin handler 与 typed handler 共存

## 分阶段 roadmap

### P0：补稳核心内核

- 去掉 `Start()` 中的 `panic`
- 闭合 `HealthCheckPort` 语义并补实现
- 去掉直接 `fmt.Println`
- 补生命周期和异常路径测试

### P1：增强组织与扩展能力

- 引入 `Option`
- 引入生命周期 hooks
- 引入 `RouteModule`
- 增加批量模块注册能力

### P2：引入 typed handler 契约

- decoder
- validator
- error mapper
- encoder
- 默认 JSON 实现

### P3：完善文档与 AI 使用材料

- 更新 `docs/httpserver.md`
- 更新 `.ai/capabilities.yaml`
- 增加更贴近 typed handler 的 AI 使用示例
- 明确推荐目录与依赖装配方式

## 风险与约束

- 如果把过多应用层约束塞进 `httpserver`，会让工具库边界失控
- 如果 typed handler 设计不够克制，可能和 Gin 直写模式产生重复抽象
- 现有文档里包含一些与真实实现不一致的 API 描述，后续需要一起校正

## 最终定位

`go-kit/httpserver` 的最终定位是：

一个保持克制、稳定可复用的 HTTP 传输层底座，同时在包内提供一套可选的、AI coding 友好的 typed handler 契约。

它不替代项目自己的日志体系、业务分层和依赖装配；它做的是把 HTTP 接入路径变得更稳、更一致、更容易复用。
