package utils

import (
	"context"
	"net"
)

// ClientIPKey 客户端 IP 在 context 中的 key
const ClientIPKey = "client_ip"

const clientIPContextKey contextKey = ClientIPKey

// RealIPHeader 代理转发 IP 的 HTTP 头名称
const RealIPHeader = "X-Real-IP"

// ForwardedForHeader 代理链 IP 的 HTTP 头名称
const ForwardedForHeader = "X-Forwarded-For"

// ParseIP 解析 IP 字符串，支持 IPv4/IPv6。
// 返回 nil 表示解析失败。
func ParseIP(ip string) net.IP {
	return net.ParseIP(ip)
}

// ParseCIDR 解析 CIDR 字符串。
// 返回 nil 表示解析失败。
func ParseCIDR(cidr string) (*net.IPNet, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	return ipNet, err
}

// IsIPInCIDR 检查 IP 是否在指定的 CIDR 段内。
func IsIPInCIDR(ip net.IP, cidr *net.IPNet) bool {
	return cidr.Contains(ip)
}

// IsIPInCIDRs 检查 IP 是否在任意的 CIDR 段内。
func IsIPInCIDRs(ip net.IP, cidrs []*net.IPNet) bool {
	for _, cidr := range cidrs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// WithClientIP 将客户端 IP 添加到 context。
func WithClientIP(ctx context.Context, clientIP string) context.Context {
	return context.WithValue(ctx, clientIPContextKey, clientIP)
}

// ClientIPFromContext 从 context 提取客户端 IP。
func ClientIPFromContext(ctx context.Context) string {
	if clientIP, ok := ctx.Value(clientIPContextKey).(string); ok {
		return clientIP
	}
	if clientIP, ok := ctx.Value(ClientIPKey).(string); ok {
		return clientIP
	}
	return ""
}
