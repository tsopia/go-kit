package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"sync"
)

// Client 加密客户端
type Client struct {
	aesKey []byte
	config *Config

	// RSA 密钥
	rsaPrivateKey *rsa.PrivateKey
	rsaPublicKey  *rsa.PublicKey
}

var (
	defaultClient *Client
	clientMu      sync.RWMutex
)

// Configure 配置默认客户端
func Configure(config *Config) error {
	if config == nil {
		return ErrMissingClient
	}

	if err := config.Validate(); err != nil {
		return err
	}

	config.normalize()

	clientMu.Lock()
	defer clientMu.Unlock()

	client := &Client{
		config: config,
	}

	if config.AESKey != "" {
		client.aesKey = []byte(config.AESKey)
	}

	// 解析 RSA 私钥
	if len(config.RSAPrivateKey) > 0 {
		priv, err := parseRSAPrivateKey(config.RSAPrivateKey)
		if err != nil {
			return err
		}
		client.rsaPrivateKey = priv
		client.rsaPublicKey = &priv.PublicKey
	}

	// 解析 RSA 公钥（如果单独提供了公钥）
	if len(config.RSAPublicKey) > 0 && client.rsaPublicKey == nil {
		pub, err := parseRSAPublicKey(config.RSAPublicKey)
		if err != nil {
			return err
		}
		client.rsaPublicKey = pub
	}

	defaultClient = client
	return nil
}

// NewClient 创建新的加密客户端（用于测试）
func NewClient(config *Config) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	client := &Client{
		config: config,
	}

	if config.AESKey != "" {
		client.aesKey = []byte(config.AESKey)
	}

	// 解析 RSA 私钥
	if len(config.RSAPrivateKey) > 0 {
		priv, err := parseRSAPrivateKey(config.RSAPrivateKey)
		if err != nil {
			return nil, err
		}
		client.rsaPrivateKey = priv
		client.rsaPublicKey = &priv.PublicKey
	}

	// 解析 RSA 公钥（如果单独提供了公钥）
	if len(config.RSAPublicKey) > 0 && client.rsaPublicKey == nil {
		pub, err := parseRSAPublicKey(config.RSAPublicKey)
		if err != nil {
			return nil, err
		}
		client.rsaPublicKey = pub
	}

	return client, nil
}

// EncryptAES 使用默认客户端加密
func (c *Client) EncryptAES(plaintext []byte) ([]byte, error) {
	if c == nil || c.aesKey == nil {
		return nil, ErrInvalidKey
	}
	return encryptAES(plaintext, c.aesKey)
}

// DecryptAES 使用默认客户端解密
func (c *Client) DecryptAES(ciphertext []byte) ([]byte, error) {
	if c == nil || c.aesKey == nil {
		return nil, ErrInvalidKey
	}
	return decryptAES(ciphertext, c.aesKey)
}

// EncryptRSA 使用 RSA-OAEP 加密
func (c *Client) EncryptRSA(plaintext []byte) ([]byte, error) {
	if c.rsaPublicKey == nil {
		return nil, ErrInvalidKey
	}
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, c.rsaPublicKey, plaintext, nil)
}

// DecryptRSA 使用 RSA-OAEP 解密
func (c *Client) DecryptRSA(ciphertext []byte) ([]byte, error) {
	if c.rsaPrivateKey == nil {
		return nil, ErrInvalidKey
	}
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, c.rsaPrivateKey, ciphertext, nil)
}

// SignRSA 使用 RSA-PSS 签名
func (c *Client) SignRSA(data []byte) ([]byte, error) {
	if c.rsaPrivateKey == nil {
		return nil, ErrInvalidKey
	}
	hash := sha256.Sum256(data)
	return rsa.SignPSS(rand.Reader, c.rsaPrivateKey, crypto.SHA256, hash[:], nil)
}

// VerifyRSA 使用 RSA-PSS 验证签名
func (c *Client) VerifyRSA(data, signature []byte) error {
	if c.rsaPublicKey == nil {
		return ErrInvalidKey
	}
	hash := sha256.Sum256(data)
	if err := rsa.VerifyPSS(c.rsaPublicKey, crypto.SHA256, hash[:], signature, nil); err != nil {
		return ErrInvalidSignature
	}
	return nil
}
