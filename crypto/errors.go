package crypto

import "errors"

var (
	// ErrMissingClient 客户端未配置
	ErrMissingClient = errors.New("crypto: client not configured, call Configure first")

	// ErrInvalidKey 无效的密钥
	ErrInvalidKey = errors.New("crypto: invalid key")

	// ErrInvalidCiphertext 无效的密文
	ErrInvalidCiphertext = errors.New("crypto: invalid ciphertext")

	// ErrInvalidSignature 签名验证失败
	ErrInvalidSignature = errors.New("crypto: signature verification failed")

	// ErrInvalidToken 无效的 JWT token
	ErrInvalidToken = errors.New("crypto: invalid JWT token")

	// ErrTokenExpired JWT token 已过期
	ErrTokenExpired = errors.New("crypto: JWT token expired")

	// ErrUnsupportedAlg 不支持的算法
	ErrUnsupportedAlg = errors.New("crypto: unsupported algorithm")
)
