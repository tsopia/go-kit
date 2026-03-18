# HTTPServer Compression Middleware Design

## 背景

当前 `httpserver/middleware` 已提供：

- `Recovery`
- `Timeout`
- `TraceID`
- `RequestID`
- `AccessLog`
- `CORS`
- `SecurityHeaders`
- `MaxBodySize`

但还没有响应压缩能力。对于 `JSON API`、文本响应、较大的列表接口，当前传输体积完全等于原始响应字节数，缺少一个通用、可控的 transport-level 优化点。

## 目标

为 `httpserver/middleware` 增加一个官方 `Compression` 中间件，满足以下要求：

- 第一版只支持 `gzip`
- 显式挂载时生效，不偷偷进入 `httpserver` core
- 对常见可压缩响应默认生效
- 对不适合压缩的场景稳定跳过
- 与现有 `AccessLog` 和 `preset` 分层兼容

## 非目标

- 不在第一版支持 `brotli`
- 不实现多算法框架或外部依赖适配层
- 不覆盖 `SSE`、WebSocket、流式刷新等场景
- 不改变 `httpserver.NewServer(...)` 的默认行为
- 不记录压缩后的二进制响应体到 payload 日志

## 设计

### 1. 能力边界

`Compression` 放在 `httpserver/middleware`，而不是 `httpserver` core。

推荐默认策略：

- `httpserver` core：默认不开启
- `middleware.Compression()`：显式挂载
- `preset.NewProductionServer(...)`：后续可默认挂载
- `preset.NewDevelopmentServer(...)`：默认不挂载

这样既保持 core 轻量，也让生产装配层可以提供更合理的默认值。

### 2. API 设计

第一版建议：

```go
type CompressionConfig struct {
    MinSizeBytes         int
    Level                int
    AllowedContentTypes  []string
    ExcludedContentTypes []string
    ShouldCompress       func(*gin.Context, int) bool
}

func Compression(configs ...CompressionConfig) gin.HandlerFunc
```

默认值：

- `MinSizeBytes = 1024`
- `Level = gzip.DefaultCompression`
- `AllowedContentTypes` 为空时使用内置默认列表
- `ShouldCompress == nil` 时只走默认规则

### 3. 默认启用条件

同时满足以下条件时才压缩：

1. 请求头 `Accept-Encoding` 包含 `gzip`
2. 响应状态允许压缩
3. 响应体大小不小于 `MinSizeBytes`
4. 响应 `Content-Type` 在允许范围内
5. 尚未设置 `Content-Encoding`
6. 不属于显式跳过场景

压缩后自动写入：

- `Content-Encoding: gzip`
- `Vary: Accept-Encoding`

并删除原有 `Content-Length`，由实际输出重新决定。

### 4. 默认跳过规则

以下情况默认不压：

- `HEAD`
- `1xx`
- `204`
- `304`
- 已设置 `Content-Encoding`
- `text/event-stream`
- WebSocket / upgrade
- 已压缩或不适合压缩的内容类型：
  - `image/*`
  - `video/*`
  - `audio/*`
  - `application/zip`
  - `application/gzip`
  - `application/pdf`
  - `application/octet-stream`

以下情况默认可压：

- `application/json`
- `text/plain`
- `text/html`
- `text/css`
- `application/javascript`
- `application/xml`
- `text/xml`

### 5. 实现方式

第一版采用“先缓冲、后决策”的保守实现：

1. 用自定义 `ResponseWriter` 缓存：
   - status
   - headers
   - body
2. handler 执行完成后统一判断是否压缩
3. 满足条件则写入 gzip body
4. 否则原样透传

选择这个方案的原因：

- 更容易准确判断最终 `Content-Type`
- 更容易按 `MinSizeBytes` 决策
- 更容易跳过 `204/304`、已有 `Content-Encoding` 等边界
- 第一版测试和行为更稳

明确边界：

- 第一版不承诺支持边写边刷新的流式响应
- 对 `SSE` / upgrade / streaming 直接跳过

### 6. 与 AccessLog 的关系

推荐文档中的中间件顺序：

```go
srv.Use(middleware.AccessLog())
srv.Use(middleware.Compression())
```

这样 `AccessLog` 可以在压缩完成后记录最终传输结果。

后续若扩展日志语义，建议：

- `bytes_out`：实际传输字节数
- `bytes_out_raw`：压缩前原始字节数
- `compression`: `"gzip"`
- `compression_ratio`: 压缩比

但这部分不必在第一版 `Compression` 中硬耦合到 `AccessLog`，保持两个中间件解耦更稳。

对于 `payload_log`：

- 如果响应被 gzip 压缩
- 第一版默认不记录压缩后二进制 body
- 最多记录“响应已压缩，跳过 payload body”这一类摘要标记

### 7. 测试策略

重点覆盖以下行为：

- 不带 `Accept-Encoding: gzip` 时不压缩
- 带 `Accept-Encoding: gzip` 且满足阈值时压缩
- 解压后内容与原始响应完全一致
- 小响应低于阈值时不压缩
- `HEAD` / `204` / `304` 不压缩
- `text/event-stream` 不压缩
- 已有 `Content-Encoding` 不重复压缩
- 不适合压缩的 `Content-Type` 被跳过
- `ShouldCompress(...)` 返回 `false` 时跳过

## 结论

第一版 `Compression` 应坚持：

- 只做 `gzip`
- 不进 core 默认值
- 用缓冲后决策保证行为稳定
- 对流式和高风险边界明确跳过

这样最符合当前 `httpserver` 的分层方向，也最适合作为 `middleware` 层的官方默认能力。
