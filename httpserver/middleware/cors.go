package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tsopia/go-kit/utils"
)

// CORSConfig 描述基础跨域配置。
type CORSConfig struct {
	AllowOrigin   string
	AllowMethods  string
	AllowHeaders  string
	ExposeHeaders string
}

// CORS 返回基础跨域中间件。
func CORS(config CORSConfig) gin.HandlerFunc {
	allowOrigin := config.AllowOrigin
	if allowOrigin == "" {
		allowOrigin = "*"
	}

	allowMethods := config.AllowMethods
	if allowMethods == "" {
		allowMethods = "GET, POST, PUT, DELETE, OPTIONS"
	}

	allowHeaders := config.AllowHeaders
	if allowHeaders == "" {
		allowHeaders = fmt.Sprintf("Content-Type, Authorization, %s, %s", utils.TraceIDHeader, utils.RequestIDHeader)
	}

	exposeHeaders := config.ExposeHeaders
	if exposeHeaders == "" {
		exposeHeaders = fmt.Sprintf("%s, %s", utils.TraceIDHeader, utils.RequestIDHeader)
	}

	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", allowOrigin)
		c.Header("Access-Control-Allow-Methods", allowMethods)
		c.Header("Access-Control-Allow-Headers", allowHeaders)
		c.Header("Access-Control-Expose-Headers", exposeHeaders)

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
