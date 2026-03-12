package crypto

import (
	"sync"
)

// Client 加密客户端
type Client struct {
	aesKey []byte
	config *Config
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

	defaultClient = &Client{
		config: config,
	}

	if config.AESKey != "" {
		defaultClient.aesKey = []byte(config.AESKey)
	}

	return nil
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
