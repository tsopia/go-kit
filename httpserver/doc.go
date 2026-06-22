// Package httpserver 提供基于 Gin 的 HTTP 服务封装。
//
// 支持直接使用 Gin 风格注册路由，也支持通过 typed handler 统一请求解码与响应编码。
//
// 基本使用：
//
//	srv := httpserver.NewServer(&httpserver.Config{
//	    Port: 8080,
//	})
//
//	srv.GET("/healthz", func(c *gin.Context) {
//	    c.JSON(200, gin.H{"status": "ok"})
//	})
//
// Typed handler：
//
//	srv.POST("/login", httpserver.HandleJSON(login))
//	srv.POST("/upload", httpserver.HandleForm(upload, httpserver.WithMaxBodyBytes(100<<20)))
//
// 流式接口：
//
//	srv.SSE("/events", func(stream httpserver.SSEStream) {})
//	srv.WS("/chat", func(session httpserver.WSSession) {}, httpserver.WithWSAllowedOrigins("https://app.example.com"))
//
// 生命周期管理：
//
//	srv.Run()                              // 阻塞启动
//	srv.Start()                            // 非阻塞启动
//	srv.RunWithContext(ctx)                // ctx 取消时执行优雅关闭
//	srv.RunWithGracefulShutdown()          // 监听系统信号并优雅关闭
//	srv.WaitForShutdown()                  // 等待信号并优雅关闭
//
// 启动方法选择指南：
//
//   - 一般 Web 服务          → RunWithGracefulShutdown()（监听信号、自动 drain + shutdown）
//   - errgroup / 并发控制    → RunWithContext(ctx)（ctx 取消时自动优雅关闭）
//   - 单元/集成测试          → Start() + defer srv.Shutdown(ctx)（非阻塞启动）
//   - 自定义 listener（测试）→ Serve(ln)（阻塞，需自行管理关闭）
//
// 服务器状态：
//
//	srv.State()      // 获取当前状态: new/starting/ready/draining/stopping/stopped/failed
//	srv.IsRunning()  // 是否运行中 (ready 或 draining 状态)
//	srv.HealthAddr() // 获取健康检查地址
//	if err := srv.MarkReady(); err != nil {
//	    // 非法状态迁移会返回错误，而不是 panic
//	}
//
// 优雅关闭配置：
//
//	srv := httpserver.NewServer(&httpserver.Config{
//	    DrainTimeout:    5 * time.Second,          // 收到关闭信号后等待时间，让负载均衡器切走流量
//	    ShutdownTimeout: 10 * time.Second,         // http.Server.Shutdown 的超时时间
//	    // DrainTimeout:    httpserver.DisableTimeout, // 显式关闭 drain 等待
//	    // ShutdownTimeout: httpserver.DisableTimeout, // 显式关闭 shutdown deadline
//	})
//
// 自定义 http.Server 配置：
//
//	srv := httpserver.NewServer(cfg, httpserver.WithHTTPServerMutator(func(s *http.Server) {
//	    s.MaxHeaderBytes = 1 << 20
//	}))
//
// 通用中间件：
//
//	srv.Use(middleware.Recovery())
//	srv.Use(middleware.Timeout(2 * time.Second))
//	srv.Use(middleware.CORS(middleware.CORSConfig{
//	    AllowOrigins: []string{"https://app.example.com"},
//	}))
//
// 注意：
//
//   - `middleware.CORS(middleware.CORSConfig{})` 表示不启用 CORS
//   - 浏览器 WebSocket 握手默认拒绝，必须显式配置 `WithWSAllowedOrigins(...)` 或 `WithWSOriginChecker(...)`
//   - 如果挂了 `middleware.AccessLog(...)`，SSE/WS 会额外输出 `stream_connect` / `stream_disconnect` / `ws_upgrade_failed`
//   - 共享主端口（HealthCheckPort == 0）时，/health、/readyz、/livez 在 NewServer 内部就已注册，
//     后续 srv.Use() 添加的中间件不会应用到这些路由；如需完整中间件链，请使用独立 HealthCheckPort
//
// 可观测性扩展：
//
//	prometheus.Register(public, prometheus.Config{Path: "/metrics"})
//	srv.Use(otel.Middleware(otel.Config{TracerName: "user-service"}))
//
// 官方默认装配：
//
//	srv := preset.NewProductionServer(nil)
//	srv.Use(middleware.AccessLog()) // 后续 helper 路由会继承新增 middleware
//	// srv.GET(...) 会走 regular helper group
//	// srv.SSE(...) / srv.WS(...) 会走 streaming helper group（不自动挂 HandlerTimeout）
//	// preset 是 transport baseline，不会自动配置 CORS、RealIP、AccessLog、认证或限流
//
// Swagger 集成：
//
//	import (
//	    _ "your/module/internal/docs"
//
//	    httpswagger "github.com/tsopia/go-kit/httpserver/swagger"
//	)
//
//	public := srv.Group("")
//	httpswagger.Register(public, httpswagger.Config{})
//
// # 流式连接
//
// SSE/WS 与普通请求共享同一中间件链。流式 handler 自动标记 utils.StreamingKey，
// 观测型中间件（AccessLog/prometheus/otel）据此跳过请求级汇总，改用连接级
// 日志与活跃连接 gauge。鉴权中间件经 c.Set 写入的数据可经 SSEStream.Get /
// WSSession.Get 读取。
//
// 更多信息请参考 README.md、httpserver/swagger/README.md 和 docs/httpserver.md
package httpserver
