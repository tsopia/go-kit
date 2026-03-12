package crypto

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims JWT 声明
type JWTClaims struct {
	jwt.RegisteredClaims
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// SignJWT 使用默认客户端和 HS256 算法签名 JWT
func SignJWT(claims JWTClaims) (string, error) {
	clientMu.RLock()
	client := defaultClient
	clientMu.RUnlock()

	if client == nil {
		return "", ErrMissingClient
	}
	return client.SignJWT(claims)
}

// SignJWTWithAlg 使用指定算法签名 JWT
func SignJWTWithAlg(claims JWTClaims, alg string) (string, error) {
	clientMu.RLock()
	client := defaultClient
	clientMu.RUnlock()

	if client == nil {
		return "", ErrMissingClient
	}
	return client.SignJWTWithAlg(claims, alg)
}

// ParseJWT 使用默认客户端解析 JWT
func ParseJWT(token string) (*JWTClaims, error) {
	clientMu.RLock()
	client := defaultClient
	clientMu.RUnlock()

	if client == nil {
		return nil, ErrMissingClient
	}
	return client.ParseJWT(token)
}

// SignJWT 使用 HS256 算法签名 JWT
func (c *Client) SignJWT(claims JWTClaims) (string, error) {
	if c.config.JWTSecret == "" {
		return "", ErrInvalidKey
	}
	setDefaultExpiry(&claims, c.config.JWTExpiry)
	return signJWTWithSecret(claims, c.config.JWTSecret, jwt.SigningMethodHS256)
}

// SignJWTWithAlg 使用指定算法签名 JWT（支持 HS256/HS384/HS512）
func (c *Client) SignJWTWithAlg(claims JWTClaims, alg string) (string, error) {
	if c.config.JWTSecret == "" {
		return "", ErrInvalidKey
	}

	var method jwt.SigningMethod
	switch alg {
	case "HS256":
		method = jwt.SigningMethodHS256
	case "HS384":
		method = jwt.SigningMethodHS384
	case "HS512":
		method = jwt.SigningMethodHS512
	default:
		return "", ErrUnsupportedAlg
	}

	setDefaultExpiry(&claims, c.config.JWTExpiry)
	return signJWTWithSecret(claims, c.config.JWTSecret, method)
}

// ParseJWT 解析并验证 JWT
func (c *Client) ParseJWT(token string) (*JWTClaims, error) {
	if c.config.JWTSecret == "" {
		return nil, ErrInvalidKey
	}
	return parseJWTWithSecret(token, c.config.JWTSecret)
}

func signJWTWithSecret(claims JWTClaims, secret string, method jwt.SigningMethod) (string, error) {
	token := jwt.NewWithClaims(method, claims)
	return token.SignedString([]byte(secret))
}

func parseJWTWithSecret(tokenString string, secret string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrUnsupportedAlg
		}
		return []byte(secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, ErrInvalidToken
}

func setDefaultExpiry(claims *JWTClaims, expiry time.Duration) {
	if claims.ExpiresAt == nil && expiry > 0 {
		claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(expiry))
	}
}
