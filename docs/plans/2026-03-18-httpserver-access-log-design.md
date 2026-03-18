# HTTPServer Access Log Design

## 背景

当前 `httpserver/middleware` 已提供：

- `Recovery`
- `Timeout`
- `TraceID`
- `RequestID`
- `CORS`
- `SecurityHeaders`
- `MaxBodySize`

但还缺少一套官方访问日志能力。现阶段最容易出问题的不是“能不能打日志”，而是：

1. 摘要日志和请求/响应内容日志混在一起，检索和成本都不可控
2. 敏感信息脱敏没有统一策略
3. `multipart/form-data` 与普通 JSON body 处理方式不同，不能直接按字符串整体记录

## 目标

在 `httpserver/middleware` 中新增一套可复用的访问日志能力，满足以下要求：

- 默认提供稳定、轻量的摘要访问日志
- 支持可选的请求/响应载荷调试日志
- 对 JSON、表单、multipart 提供结构化脱敏能力
- 不依赖具体日志实现，允许调用方接入任意 logger

## 非目标

- 不直接绑定 `kit`、Zap 或其他日志库
- 不自动集成 metrics/tracing exporter
- 不默认记录完整请求体和响应体
- 不记录 multipart 文件内容
- 不在第一版中支持复杂正则规则或动态 DSL

## 设计

### 1. 分层日志模型

访问日志拆成两类事件：

1. `AccessLog`
2. `PayloadLog`

`AccessLog` 面向主链路检索和问题定位，默认开启，只记录摘要字段。

`PayloadLog` 面向调试和灰度排查，默认关闭，只在显式开启且命中过滤条件时输出。`PayloadLog` 通过同一 `request_id` / `trace_id` 与 `AccessLog` 关联。

### 2. AccessLog 事件字段

默认摘要字段：

- `method`
- `path`
- `route`
- `status`
- `latency_ms`
- `client_ip`
- `host`
- `user_agent`
- `referer`
- `request_id`
- `trace_id`
- `bytes_in`
- `bytes_out`
- `error`

默认级别：

- `INFO`：`2xx`、`3xx`、`4xx`
- `ERROR`：`5xx`

### 3. PayloadLog 事件字段

`PayloadLog` 默认级别为 `DEBUG`，可选记录：

- `request_headers`
- `response_headers`
- `request_body`
- `response_body`
- `request_truncated`
- `response_truncated`
- `content_type`

`PayloadLog` 不直接输出原始 `multipart` 流文本，而是输出经过解析和脱敏后的结构化结果。

### 4. multipart 处理策略

`multipart/form-data` 不能按普通字符串 body 直接记录，原因是原始 body 同时包含：

- boundary
- part headers
- 文本字段
- 文件字段
- 可能的二进制内容

因此需要按 part 维度处理，而不是 `string(body)`。

第一版支持以下模式：

```go
type MultipartCaptureMode string

const (
    MultipartDisabled       MultipartCaptureMode = "disabled"
    MultipartMetadataOnly   MultipartCaptureMode = "metadata_only"
    MultipartFormFieldsOnly MultipartCaptureMode = "form_fields_only"
    MultipartSelectedParts  MultipartCaptureMode = "selected_parts"
)
```

默认使用 `MultipartMetadataOnly`：

- 文本字段：只记录字段名
- 文件字段：记录 `field`、`filename`、`content_type`、`size`

`MultipartFormFieldsOnly`：

- 文本字段：记录脱敏后的值
- 文件字段：仍只记录元数据

`MultipartSelectedParts`：

- 仅对白名单文本 part 记录脱敏后的值
- 文件字段仍不记录内容

第一版不支持记录文件内容。

### 5. 脱敏模型

脱敏采用“先解析、后脱敏、再记录”的策略，而不是在原始字符串上做替换。

支持的载荷类型：

- JSON
- `application/x-www-form-urlencoded`
- `multipart/form-data` 文本字段
- query 参数
- header 白名单

默认策略分四类：

```go
type RedactionStrategy string

const (
    RedactionRedact RedactionStrategy = "redact"
    RedactionMask   RedactionStrategy = "mask"
    RedactionHash   RedactionStrategy = "hash"
    RedactionDrop   RedactionStrategy = "drop"
)
```

规则按 `scope + key` 匹配，第一版先支持精确 key 匹配且忽略大小写。

默认敏感键建议包含：

- `authorization`
- `cookie`
- `set-cookie`
- `password`
- `token`
- `access_token`
- `refresh_token`
- `secret`
- `client_secret`
- `private_key`
- `phone`
- `mobile`
- `email`
- `id_card`
- `bank_card`

默认策略建议：

- `authorization`、`cookie`、`password`、`token`：`redact`
- `phone`、`email`、`id_card`：`mask`
- `filename`：可选 `mask` 或 `hash`

### 6. 配置模型

第一版建议的主配置：

```go
type LoggerFunc func(ctx context.Context, level string, event string, fields map[string]any)

type AccessLogConfig struct {
    Logger              LoggerFunc
    CapturePayload      bool
    PayloadLogLevel     string
    MaxBodyBytes        int64
    AllowedContentTypes []string

    AllowedRequestHeaders  []string
    AllowedResponseHeaders []string

    ShouldCapturePayload func(*gin.Context, int) bool

    Multipart MultipartConfig
    Redaction RedactionConfig
}

type MultipartConfig struct {
    Mode                 MultipartCaptureMode
    PartAllowlist        []string
    MaxPartValueBytes    int64
    RedactFilenames      bool
}

type RedactionConfig struct {
    Rules         []RedactionRule
    MaxValueBytes int
    HashSalt      string
}

type RedactionRule struct {
    Scope    string
    Key      string
    Strategy RedactionStrategy
}
```

### 7. 输出策略

中间件可以在一次请求中输出两条日志：

1. 一条 `access_log`
2. 零或一条 `payload_log`

这样做的好处是：

- access log 保持轻量稳定
- payload log 只在必要时出现
- 两类事件都能通过 `request_id` / `trace_id` 关联

## 实现边界

第一版只做以下内容：

- `AccessLog(config ...AccessLogOption)` 中间件
- 摘要日志输出
- 可选 payload capture
- JSON / form / multipart 文本字段脱敏
- multipart 文件元数据捕获

第一版暂不做：

- 响应流式 body 捕获
- gzip/br 解压后载荷捕获
- 基于正则或 JSONPath 的复杂脱敏规则
- 多后端日志适配层

## 测试策略

关键测试覆盖：

- 摘要 access log 默认输出
- 5xx 日志级别提升为 `error`
- request/response payload 在显式开启时输出
- JSON body 脱敏
- form body 脱敏
- multipart metadata only
- multipart form fields only
- header 白名单与敏感 header 脱敏
- payload 大小截断标记
