package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tsopia/go-kit/utils"
)

// RealIPConfig 配置可信代理解析。
type RealIPConfig struct {
	// TrustedCIDRs 信任代理的 CIDR 列表。
	// 默认空列表，表示不信任任何代理（直接使用 RemoteAddr）。
	TrustedCIDRs []string

	// OnUntrusted 当检测到不可信代理时的处理函数。
	// 如果为 nil，则使用 RemoteAddr。
	OnUntrusted func(c *gin.Context, remoteAddr string)
}

// RealIP 创建中间件（使用默认配置）。
// 默认配置下，不信任任何代理，直接使用 RemoteAddr。
func RealIP() gin.HandlerFunc {
	return RealIPWithConfig(RealIPConfig{})
}

// RealIPWithConfig 使用自定义配置创建中间件。
func RealIPWithConfig(config RealIPConfig) gin.HandlerFunc {
	cidrs := parseCIDRs(config.TrustedCIDRs)

	return func(c *gin.Context) {
		clientIP := extractClientIP(c, cidrs, config)

		// 写入 gin context
		c.Set(utils.ClientIPKey, clientIP)

		// 写入 request context
		ctx := utils.WithClientIP(c.Request.Context(), clientIP)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// parseCIDRs 解析 CIDR 字符串列表。
func parseCIDRs(cidrs []string) []*net.IPNet {
	if len(cidrs) == 0 {
		return nil
	}

	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		if cidr == "" {
			continue
		}
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			// 解析失败，跳过该 CIDR
			continue
		}
		nets = append(nets, ipNet)
	}
	return nets
}

// extractClientIP 从请求中提取客户端 IP。
func extractClientIP(c *gin.Context, cidrs []*net.IPNet, config RealIPConfig) string {
	// 获取 RemoteAddr 的 IP
	remoteIP, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		// IPv6 或没有端口的情况
		remoteIP = c.Request.RemoteAddr
	}

	remoteAddr := net.ParseIP(remoteIP)
	if remoteAddr == nil {
		// 解析失败，返回原始 RemoteAddr
		return remoteIP
	}

	// 如果没有配置信任 CIDR，直接使用 RemoteAddr
	if len(cidrs) == 0 {
		return remoteIP
	}

	// 检查 RemoteAddr 是否在信任 CIDR 中
	if !utils.IsIPInCIDRs(remoteAddr, cidrs) {
		// RemoteAddr 不在信任列表中，可能是直连或不可信代理
		if config.OnUntrusted != nil {
			config.OnUntrusted(c, remoteIP)
		}
		return remoteIP
	}

	// RemoteAddr 是可信代理，尝试解析 X-Forwarded-For
	clientIP := extractFromForwardedFor(c, cidrs)
	if clientIP != "" {
		return clientIP
	}

	// 尝试 X-Real-IP
	clientIP = extractFromRealIP(c, cidrs)
	if clientIP != "" {
		return clientIP
	}

	// 没有代理头，返回 RemoteAddr
	return remoteIP
}

// extractFromForwardedFor 从 X-Forwarded-For 头提取客户端 IP。
// 从左到右遍历，找到第一个不在信任 CIDR 中的 IP。
func extractFromForwardedFor(c *gin.Context, cidrs []*net.IPNet) string {
	forwardedFor := c.GetHeader(utils.ForwardedForHeader)
	if forwardedFor == "" {
		return ""
	}

	// X-Forwarded-For 格式: client, proxy1, proxy2, ...
	// 最左边的是原始客户端，依次向右是经过的代理
	ips := splitAndTrim(forwardedFor, ",")

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}

		// 找到第一个不在信任 CIDR 中的 IP
		if !utils.IsIPInCIDRs(ip, cidrs) {
			return ipStr
		}
	}

	// 所有 IP 都在信任 CIDR 中，返回最后一个（最接近边缘代理）
	if len(ips) > 0 {
		return ips[len(ips)-1]
	}

	return ""
}

// extractFromRealIP 从 X-Real-IP 头提取客户端 IP。
// 只接受不在信任 CIDR 中的 IP。
func extractFromRealIP(c *gin.Context, cidrs []*net.IPNet) string {
	realIP := c.GetHeader(utils.RealIPHeader)
	if realIP == "" {
		return ""
	}

	ip := net.ParseIP(realIP)
	if ip == nil {
		return ""
	}

	// 如果 X-Real-IP 在信任 CIDR 中，可能是代理配置错误，忽略
	if utils.IsIPInCIDRs(ip, cidrs) {
		return ""
	}

	return realIP
}

// splitAndTrim 分割字符串并去除空白。
func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// clientIPFromContext 从 gin context 获取客户端 IP。
// 优先使用 RealIP 中间件设置的值，降级到 Gin 的 ClientIP()。
func clientIPFromContext(c *gin.Context) string {
	// 优先从 gin context 获取（RealIP 中间件设置）
	if clientIP, exists := c.Get(utils.ClientIPKey); exists {
		if value, ok := clientIP.(string); ok && value != "" {
			return value
		}
	}

	// 降级到 Gin 的 ClientIP()
	return c.ClientIP()
}

// ClientIPFromRequest 从 http.Request 获取客户端 IP。
// 优先使用 RealIP 中间件设置的 context 值，降级到 RemoteAddr。
func ClientIPFromRequest(r *http.Request) string {
	// 优先从 context 获取
	if clientIP := utils.ClientIPFromContext(r.Context()); clientIP != "" {
		return clientIP
	}

	// 降级到 RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
