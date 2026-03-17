// Package prometheus 提供 HTTP 指标中间件与 metrics 路由注册能力。
//
// 该包默认不向 httpserver core 注入任何路由；调用方需要显式调用 Register
// 挂载 metrics 端点。
package prometheus
