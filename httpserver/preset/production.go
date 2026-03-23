package preset

import (
	"github.com/tsopia/go-kit/httpserver"
	"github.com/tsopia/go-kit/httpserver/middleware"
)

// NewProductionServer 创建带官方生产默认链路的服务器。
func NewProductionServer(config *httpserver.Config, opts ...httpserver.Option) *httpserver.Server {
	// 1. 创建基础 server（此时还没有任何中间件）
	srv := httpserver.NewServer(config, opts...)

	// 2. 添加共享中间件（Recovery, RequestID, TraceID, SecurityHeaders）
	srv.Use(middleware.Recovery())
	srv.Use(middleware.RequestID())
	srv.Use(middleware.TraceID())
	srv.Use(middleware.SecurityHeaders())

	// 3. 创建 streamingGroup（不挂 Timeout）
	streamingGroup := srv.Engine().Group("/")

	// 4. 创建 regularGroup，根据 HandlerTimeout 决定是否挂 Timeout
	regularGroup := srv.Engine().Group("/")
	if config.HandlerTimeout > 0 {
		regularGroup.Use(middleware.Timeout(config.HandlerTimeout))
	}

	// 5. 设置两个 Group 到 server
	srv.SetGroups(regularGroup, streamingGroup)

	return srv
}
