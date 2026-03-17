package preset

import (
	"github.com/tsopia/go-kit/httpserver"
	"github.com/tsopia/go-kit/httpserver/middleware"
)

// NewDevelopmentServer 创建适合开发环境的默认服务器。
func NewDevelopmentServer(config *httpserver.Config, opts ...httpserver.Option) *httpserver.Server {
	srv := httpserver.NewServer(config, opts...)

	srv.Use(middleware.Recovery())
	srv.Use(middleware.RequestID())
	srv.Use(middleware.TraceID())

	return srv
}
