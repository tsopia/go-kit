package crypto

import (
	"errors"
	"time"
)

// Config 加密配置
type Config struct {
	// AES 配置
	AESKey string // 32字节用于 AES-256-GCM

	// RSA 配置
	RSAPrivateKey []byte // PEM 格式私钥
	RSAPublicKey  []byte // PEM 格式公钥

	// JWT 配置
	JWTSecret     string        // HMAC 密钥
	JWTPrivateKey []byte        // RSA 私钥（用于 RS256）
	JWTPublicKey  []byte        // RSA 公钥（用于 RS256）
	JWTExpiry     time.Duration // 默认过期时间（默认 24h）
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		JWTExpiry: 24 * time.Hour,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is nil")
	}
	return nil
}

// normalize 设置默认值
func (c *Config) normalize() {
	if c.JWTExpiry == 0 {
		c.JWTExpiry = 24 * time.Hour
	}
}
