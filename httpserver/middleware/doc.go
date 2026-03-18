// Package middleware 提供可复用的 HTTP 中间件集合。
//
// 该包面向 `httpserver` 以及直接使用 Gin 的项目，提供 Recovery、Timeout、
// TraceID、RequestID、AccessLog、Compression、ConcurrencyLimit、CORS、安全响应头和请求体大小限制等通用中间件。
// 其中 ConcurrencyLimit 用于单进程全局并发闸门，超限请求会立即返回 503，不排队等待。
package middleware
