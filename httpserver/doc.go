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
//	srv.WS("/chat", func(session httpserver.WSSession) {})
//
// 生命周期管理：
//
//	srv.Run()                              // 阻塞启动
//	srv.Start()                            // 非阻塞启动
//	srv.WaitForShutdown()                  // 等待信号并优雅关闭
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
//	    DrainTimeout:    5 * time.Second,  // 收到关闭信号后等待时间，让负载均衡器切走流量
//	    ShutdownTimeout: 10 * time.Second, // http.Server.Shutdown 的超时时间
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
//
// 可观测性扩展：
//
//	prometheus.Register(public, prometheus.Config{Path: "/metrics"})
//	srv.Use(otel.Middleware(otel.Config{TracerName: "user-service"}))
//
// 官方默认装配：
//
//	srv := preset.NewProductionServer(nil)
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
// 更多信息请参考 README.md、httpserver/swagger/README.md 和 docs/httpserver.md
package httpserver
