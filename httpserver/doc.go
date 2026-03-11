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
