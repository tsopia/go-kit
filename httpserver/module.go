package httpserver

import "github.com/gin-gonic/gin"

// RouteModule 描述一组可注册到 Gin 路由树的模块。
type RouteModule interface {
	RegisterRoutes(r gin.IRoutes)
}

// RegisterModules 批量注册路由模块。
func (s *Server) RegisterModules(modules ...RouteModule) {
	for _, module := range modules {
		if module == nil {
			continue
		}

		module.RegisterRoutes(s.engine)
	}
}
