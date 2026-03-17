package preset

import (
	"time"

	"github.com/tsopia/go-kit/httpserver"
	"github.com/tsopia/go-kit/httpserver/middleware"
)

// NewProductionServer 创建带官方生产默认链路的服务器。
func NewProductionServer(config *httpserver.Config, opts ...httpserver.Option) *httpserver.Server {
	srv := httpserver.NewServer(config, opts...)

	srv.Use(middleware.Recovery())
	srv.Use(middleware.RequestID())
	srv.Use(middleware.TraceID())
	srv.Use(middleware.Timeout(productionTimeout(config)))
	srv.Use(middleware.SecurityHeaders())

	return srv
}

func productionTimeout(config *httpserver.Config) time.Duration {
	if config != nil && config.ReadTimeout > 0 {
		return config.ReadTimeout
	}

	return httpserver.DefaultConfig().ReadTimeout
}
