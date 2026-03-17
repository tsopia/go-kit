// Package otel 提供 HTTP tracing 中间件。
//
// 该包只负责 span 创建、上下文传播和错误状态记录，不负责 exporter 或 provider
// 初始化，后者应由应用装配层显式提供。
package otel
