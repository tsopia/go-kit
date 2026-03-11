package swagger

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Register 将 Swagger UI 路由注册到传入的 Gin 路由组。
func Register(r gin.IRoutes, cfg Config) {
	cfg = applyDefaults(cfg)

	r.GET(
		cfg.Path,
		ginSwagger.WrapHandler(
			swaggerFiles.Handler,
			ginSwagger.URL(cfg.DocURL),
		),
	)
}
