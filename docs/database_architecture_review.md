# Database 包架构评审（架构师视角）

## 现状亮点
- **配置生命周期完整**：`Config` 在 `SetDefaults` 中填充默认值并在 `Validate` 中执行 driver、连接池、超时和重试的专项校验，减少运行时意外。相关逻辑集中在一个结构体方法上，便于使用者快速理解配置入口。【F:database/database.go†L225-L286】【F:database/database.go†L306-L450】
- **连接建立具备降级兜底**：`New` 在建连失败时会关闭已经建立的连接，避免泄漏；`connectWithRetry` 支持指数退避与抖动，提升启动韧性。【F:database/database.go†L500-L606】
- **运行期可观测性基础设施**：暴露了连接池统计、健康检查和查询探针，便于运维侧做活性与压力判断。【F:database/database.go†L732-L854】
- **日志适配层清晰**：`GORMLogger` 适配自定义日志接口，并支持 Context 感知，方便与外部日志体系解耦。【F:database/gorm_adapter.go†L10-L116】

## 主要架构顾虑
1. **职责过于集中**：`database.go` 将配置、错误、连接、健康检查、事务辅助等全部塞入单文件，难以维护，也限制了针对不同关注点的测试与演进。【F:database/database.go†L24-L912】
2. **边界抽象不足**：核心对外暴露的是 GORM `*gorm.DB`，上层很容易跳过封装直接使用底层能力，破坏封装的一致性，也难以在未来切换 ORM 或添加拦截器。【F:database/database.go†L699-L904】
3. **可观测性割裂**：重试与连接建立使用 `log.Printf` 直写标准输出，无法复用自定义 `SimpleLogger`，会导致日志格式混乱或丢失 trace 上下文。【F:database/database.go†L576-L606】
4. **配置安全与环境解耦不足**：目前 `Config` 只是一个数据对象，缺少与配置中心/环境变量的解耦层，也未对敏感信息（如密码）提供统一脱敏输出或审计 hook，仅靠 `SafeString` 被动处理。【F:database/database.go†L904-L911】
5. **缺少生命周期治理**：`HealthCheckWithContext` 仅执行简单探针，没有暴露钩子让业务在探针前后插入自定义检查，`Close` 也未暴露 context 或 shutdown hook，不利于在复杂进程生命周期中协调关闭顺序。【F:database/database.go†L760-L854】【F:database/database.go†L711-L721】

## 改进建议
- **按关注点拆分子包/文件**：
  - `config.go`：配置结构、默认值、校验与安全输出。
  - `errors.go`：错误类型与判定助手。
  - `connection.go`：连接、重试、连接池配置与 DSN 生成。
  - `health.go`：健康检查、探针、统计模型。
  - `adapter/`：GORM 日志与 future driver 适配层。
  这样可以降低单文件复杂度，并允许针对每个维度编写粒度化测试。【F:database/database.go†L24-L912】
- **收敛对外接口**：定义一个面向业务的 `DB` 接口（例如 `Exec(ctx, query, args...)`、`Query(ctx, ... )`、`Tx(ctx, fn)`），内部再委托给 GORM；同时保留受控的 `Raw()`/`Gorm()` 逃逸口，减少随意绕过封装的可能。【F:database/database.go†L699-L904】
- **统一日志/metrics 通道**：在连接重试、健康检查告警、事务失败等路径上注入 `SimpleLogger` / `ContextualLogger` 与 metrics hook（例如可选的 Prometheus 接口），避免 `log.Printf` 混杂输出，并为 SRE 提供统一的观测入口。【F:database/database.go†L576-L606】【F:database/gorm_adapter.go†L10-L116】
- **配置加载与密钥治理**：增加 `Loader` 抽象支持从 env / config center / 文件加载 `Config`，并在创建连接前对敏感字段做审计或屏蔽（如统一输出 `SafeString`）。未来可在此层加入动态刷新或 rotate 功能。【F:database/database.go†L225-L286】【F:database/database.go†L904-L911】
- **生命周期扩展点**：为 `HealthCheck`、`Close`、`Transaction` 提供可注入的 hook（如 `BeforeClose`, `AfterConnect`, `BeforeProbe`），方便在业务中挂载迁移、熔断、流量切换等流程，同时支持 context 传递，确保优雅停机与弹性扩缩容场景可控。【F:database/database.go†L500-L904】

## 优化计划（初稿）
1. **对外接口收敛与受控逃逸**：设计 `DB`/`Tx` 接口封装常用能力（`Query`/`Exec`/`Tx(ctx, fn)`），同时明确仅在需要底层特性的场景通过受控方法返回 `*gorm.DB`/`*gorm.DB` 的必要子集，确保多数业务留在封装层，减少随意绕过。
2. **配置治理与覆盖策略**：定义 `Config` 的默认值表与优先级规则（默认 < 外部注入 < 运行时覆盖），暴露 `WithConfig`/`WithOptions` 等构造器，保证非必要项一律来自调用方，以便兼容不同部署环境，并可覆盖默认值。
3. **拆分模块与测试基线**：按配置、连接、健康、适配层拆分文件/子包，建立对应的单元测试与接口契约，便于独立演进和回归。
4. **日志与可观测性统一**：替换直接使用 `log.Printf` 的路径，统一使用已有日志接口，并预留 metrics/hook 注入点，覆盖连接重试、事务失败和健康探针。
5. **生命周期与扩展点**：在 `HealthCheck`、`Close`、`Transaction` 等流程暴露前后 hook，支持 `context.Context` 贯穿，便于调用方插入迁移、熔断或流量切换逻辑。

## 需求澄清（基于对话）
- **GORM 逃逸需求**：业务侧没有明确枚举，但曾遇到必须使用底层能力的场景，需保留受控逃逸口。建议仅暴露：
  1) `RawDB()` 返回 `*gorm.DB` 以支持批量写、复杂子查询或插件；
  2) `RawSQL()` 允许执行未覆盖的 SQL；
  3) 事务隔离级别、锁语义等高级选项通过受控参数向下传递。
- **配置来源与环境**：仅接收外部传入的 `Config` 结构体（无内置环境区分），环境切换由调用方自行管理；封装层只负责应用默认值与校验，并允许调用方覆盖所有非必需项。

## 默认配置（架构师建议值）
- **连接池**：`MaxIdleConns=10`、`MaxOpenConns=100`、`ConnMaxLifetime=1h`、`ConnMaxIdleTime=10m`，兼顾中等负载与连接老化控制。【F:database/database.go†L210-L241】
- **GORM 日志**：默认 `LogLevel="silent"`，`SlowThreshold=1s`，对慢查询做显式标记，避免生产环境噪声；可按需提升到 `info`/`warn`。【F:database/database.go†L210-L241】
- **驱动特定**：MySQL 默认 `Charset=utf8mb4`、`Timezone=Local`；PostgreSQL 默认 `SSLMode=disable`（可被外部覆盖）、`Timezone=UTC`。确保跨时区行为一致且默认不开启潜在的自签证书校验风险。【F:database/database.go†L210-L241】
- **重试**：启用指数退避且默认抖动，`RetryMaxAttempts=3`、`RetryInitialDelay=100ms`、`RetryMaxDelay=2s`、`RetryBackoffFactor=2`、`RetryJitterEnabled=true`，平衡启动韧性与快速失败。【F:database/database.go†L210-L241】【F:database/database.go†L500-L606】
- **必须外部提供**：`Driver`、`Host`、`Port`、`Username`、`Password`、`Database` 等敏感或与环境强绑定的字段始终由调用方注入，封装不兜底，以满足安全与合规需求。

## 日志、指标与运维 SLO（架构师建议）
- **日志/metrics 目标**：
  - 统一走自定义 `SimpleLogger`/`ContextualLogger`，覆盖连接重试、慢查询、事务失败与健康探针，禁止直接 `log.Printf`；
  - 预留可选 metrics hook（Prometheus/OTel），标签统一包含 `db_driver`、`db_host`、`db_name`、`op`、`result`、`latency_bucket`；
  - 提供敏感字段脱敏输出，避免 DSN、密码泄漏。
- **健康检查与关闭 SLO**：
  - 健康探针超时默认 2s，失败可配置重试 3 次（与连接重试解耦），失败时输出结构化原因；
  - 优雅停机顺序：先停止写流量/队列消费，再执行 `BeforeClose` hook（可插迁移或数据刷盘），最后 `Close(ctx)` 在 5s 内完成并暴露结果；
  - `Transaction`/`HealthCheck` 暴露 `Before/After` hook 以便业务注入熔断、降级或切换流量逻辑。

