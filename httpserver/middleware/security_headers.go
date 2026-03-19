package middleware

import "github.com/gin-gonic/gin"

// SecurityHeadersConfig 描述安全响应头行为。
type SecurityHeadersConfig struct {
	// HSTS 设置 Strict-Transport-Security 头。
	// 示例: "max-age=31536000; includeSubDomains"
	HSTS string

	// ContentSecurityPolicy 设置 Content-Security-Policy 头。
	// 示例: "default-src 'self'"
	ContentSecurityPolicy string

	// PermissionsPolicy 设置 Permissions-Policy 头。
	// 示例: "camera=(), microphone=()"
	PermissionsPolicy string
}

// SecurityHeaders 添加基础安全响应头。
func SecurityHeaders() gin.HandlerFunc {
	return SecurityHeadersWithConfig(SecurityHeadersConfig{})
}

// SecurityHeadersWithConfig 使用自定义配置创建安全头中间件。
func SecurityHeadersWithConfig(config SecurityHeadersConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 始终设置基础安全头
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")

		if config.HSTS != "" {
			c.Header("Strict-Transport-Security", config.HSTS)
		}
		if config.ContentSecurityPolicy != "" {
			c.Header("Content-Security-Policy", config.ContentSecurityPolicy)
		}
		if config.PermissionsPolicy != "" {
			c.Header("Permissions-Policy", config.PermissionsPolicy)
		}

		c.Next()
	}
}
