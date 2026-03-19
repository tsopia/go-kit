package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tsopia/go-kit/utils"
)

// CORSConfig 描述跨域配置。
type CORSConfig struct {
	// AllowOrigins 允许的 Origin 列表。
	// 如果包含 "*"，则允许所有 Origin（不可与 AllowCredentials 同时使用）。
	AllowOrigins []string

	// AllowOriginFunc 动态判断 Origin 是否允许。
	// 设置后完全接管 origin 判断，AllowOrigins 不再生效。
	// 返回 true 表示允许，返回 false 表示拒绝（不 fallback 到 AllowOrigins）。
	AllowOriginFunc func(origin string) bool

	AllowMethods  string
	AllowHeaders  string
	ExposeHeaders string

	// AllowCredentials 是否允许携带凭证。
	// 为 true 时不可使用 "*" 作为 AllowOrigin。
	AllowCredentials bool

	// MaxAge 预检请求缓存时间。
	MaxAge time.Duration
}

// CORS 返回跨域中间件。
func CORS(config CORSConfig) gin.HandlerFunc {
	config = normalizeCORSConfig(config)

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		allowedOrigin := matchOrigin(origin, config)
		if allowedOrigin == "" {
			c.Next()
			return
		}

		c.Header("Access-Control-Allow-Origin", allowedOrigin)
		c.Header("Access-Control-Allow-Methods", config.AllowMethods)
		c.Header("Access-Control-Allow-Headers", config.AllowHeaders)
		c.Header("Access-Control-Expose-Headers", config.ExposeHeaders)

		if config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == http.MethodOptions {
			if config.MaxAge > 0 {
				c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", int(config.MaxAge.Seconds())))
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func normalizeCORSConfig(config CORSConfig) CORSConfig {
	if len(config.AllowOrigins) == 0 && config.AllowOriginFunc == nil {
		config.AllowOrigins = []string{"*"}
	}
	if config.AllowMethods == "" {
		config.AllowMethods = "GET, POST, PUT, DELETE, OPTIONS"
	}
	if config.AllowHeaders == "" {
		config.AllowHeaders = fmt.Sprintf("Content-Type, Authorization, %s, %s", utils.TraceIDHeader, utils.RequestIDHeader)
	}
	if config.ExposeHeaders == "" {
		config.ExposeHeaders = fmt.Sprintf("%s, %s", utils.TraceIDHeader, utils.RequestIDHeader)
	}

	// W3C: credentials 模式下不允许 wildcard origin
	if config.AllowCredentials {
		filtered := make([]string, 0, len(config.AllowOrigins))
		for _, o := range config.AllowOrigins {
			if o != "*" {
				filtered = append(filtered, o)
			}
		}
		config.AllowOrigins = filtered
	}

	return config
}

func matchOrigin(origin string, config CORSConfig) string {
	if config.AllowOriginFunc != nil {
		if config.AllowOriginFunc(origin) {
			return origin
		}
		return "" // func 是最终判决，不 fallback 到 AllowOrigins
	}

	for _, allowed := range config.AllowOrigins {
		if allowed == "*" {
			return "*"
		}
		if strings.EqualFold(allowed, origin) {
			return origin
		}
	}

	return ""
}
