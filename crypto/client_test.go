package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestRSAKey 生成测试用的 RSA 密钥对
func generateTestRSAKey() (privateKeyPEM, publicKeyPEM []byte, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	// 编码私钥为 PKCS#1 PEM
	privateKeyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// 编码公钥为 PKIX PEM
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	publicKeyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	return privateKeyPEM, publicKeyPEM, nil
}

func TestConfigure(t *testing.T) {
	privateKeyPEM, publicKeyPEM, err := generateTestRSAKey()
	require.NoError(t, err)

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config with AES key",
			config: &Config{
				AESKey: "this-is-a-32-byte-key-for-aes256",
			},
			wantErr: false,
		},
		{
			name: "valid config with RSA keys",
			config: &Config{
				RSAPrivateKey: privateKeyPEM,
				RSAPublicKey:  publicKeyPEM,
			},
			wantErr: false,
		},
		{
			name: "valid config with all keys",
			config: &Config{
				AESKey:        "this-is-a-32-byte-key-for-aes256",
				RSAPrivateKey: privateKeyPEM,
				RSAPublicKey:  publicKeyPEM,
				JWTSecret:     "my-test-secret-key",
				JWTExpiry:     time.Hour,
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "short AES key - allowed at config level, error at use",
			config: &Config{
				AESKey: "short-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 保存当前默认客户端
			oldClient := defaultClient
			defer func() { defaultClient = oldClient }()

			err := Configure(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, defaultClient)
		})
	}
}

func TestNewClient(t *testing.T) {
	privateKeyPEM, publicKeyPEM, err := generateTestRSAKey()
	require.NoError(t, err)

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid client with AES key",
			config: &Config{
				AESKey: "this-is-a-32-byte-key-for-aes256",
			},
			wantErr: false,
		},
		{
			name: "valid client with RSA keys",
			config: &Config{
				RSAPrivateKey: privateKeyPEM,
				RSAPublicKey:  publicKeyPEM,
			},
			wantErr: false,
		},
		{
			name: "valid client with JWT config",
			config: &Config{
				JWTSecret: "my-test-secret-key",
				JWTExpiry: time.Hour,
			},
			wantErr: false,
		},
		{
			name: "valid client with all config",
			config: &Config{
				AESKey:        "this-is-a-32-byte-key-for-aes256",
				RSAPrivateKey: privateKeyPEM,
				RSAPublicKey:  publicKeyPEM,
				JWTSecret:     "my-test-secret-key",
				JWTExpiry:     time.Hour,
			},
			wantErr: false,
		},
		{
			name:    "empty config - allowed, creates client with no keys",
			config:  &Config{},
			wantErr: false,
		},
		{
			name: "short AES key - allowed at config level, error at use",
			config: &Config{
				AESKey: "short",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, client)
		})
	}
}

func TestClient_AES(t *testing.T) {
	client, err := NewClient(&Config{
		AESKey: "this-is-a-32-byte-key-for-aes256",
	})
	require.NoError(t, err)

	tests := []struct {
		name      string
		plaintext []byte
		wantErr   bool
	}{
		{
			name:      "simple text",
			plaintext: []byte("hello world"),
			wantErr:   false,
		},
		{
			name:      "empty text",
			plaintext: []byte(""),
			wantErr:   false,
		},
		{
			name:      "unicode text",
			plaintext: []byte("你好世界 🌍 Привет мир"),
			wantErr:   false,
		},
		{
			name:      "binary data",
			plaintext: []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD, 0xFC},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 加密
			encrypted, err := client.EncryptAES(tt.plaintext)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, encrypted)

			// 解密
			decrypted, err := client.DecryptAES(encrypted)
			assert.NoError(t, err)
			assert.True(t, bytes.Equal(decrypted, tt.plaintext))
		})
	}
}

func TestClient_AES_NoKey(t *testing.T) {
	client, err := NewClient(&Config{
		JWTSecret: "my-test-secret-key",
	})
	require.NoError(t, err)

	_, err = client.EncryptAES([]byte("test"))
	assert.ErrorIs(t, err, ErrInvalidKey)

	_, err = client.DecryptAES([]byte("test"))
	assert.ErrorIs(t, err, ErrInvalidKey)
}

func TestClient_RSA(t *testing.T) {
	privateKeyPEM, publicKeyPEM, err := generateTestRSAKey()
	require.NoError(t, err)

	tests := []struct {
		name      string
		plaintext []byte
		wantErr   bool
	}{
		{
			name:      "simple text",
			plaintext: []byte("hello world"),
			wantErr:   false,
		},
		{
			name:      "empty text",
			plaintext: []byte(""),
			wantErr:   false,
		},
		{
			name:      "unicode text",
			plaintext: []byte("你好世界 🌍"),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(&Config{
				RSAPrivateKey: privateKeyPEM,
				RSAPublicKey:  publicKeyPEM,
			})
			require.NoError(t, err)

			// 测试加密/解密
			encrypted, err := client.EncryptRSA(tt.plaintext)
			assert.NoError(t, err)
			assert.NotNil(t, encrypted)

			decrypted, err := client.DecryptRSA(encrypted)
			assert.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)

			// 测试签名/验证
			signature, err := client.SignRSA(tt.plaintext)
			assert.NoError(t, err)
			assert.NotNil(t, signature)

			err = client.VerifyRSA(tt.plaintext, signature)
			assert.NoError(t, err)
		})
	}
}

func TestClient_RSA_InvalidSignature(t *testing.T) {
	privateKeyPEM, publicKeyPEM, err := generateTestRSAKey()
	require.NoError(t, err)

	client, err := NewClient(&Config{
		RSAPrivateKey: privateKeyPEM,
		RSAPublicKey:  publicKeyPEM,
	})
	require.NoError(t, err)

	data := []byte("data to sign")
	wrongData := []byte("wrong data")

	// 签名
	signature, err := client.SignRSA(data)
	require.NoError(t, err)

	// 使用错误的数据验证
	err = client.VerifyRSA(wrongData, signature)
	assert.ErrorIs(t, err, ErrInvalidSignature)
}

func TestClient_RSA_NoKeys(t *testing.T) {
	// 只创建 AES 客户端，没有 RSA 密钥
	client, err := NewClient(&Config{
		AESKey: "this-is-a-32-byte-key-for-aes256",
	})
	require.NoError(t, err)

	// 加密应该失败
	_, err = client.EncryptRSA([]byte("test"))
	assert.ErrorIs(t, err, ErrInvalidKey)

	// 解密应该失败
	_, err = client.DecryptRSA([]byte("test"))
	assert.ErrorIs(t, err, ErrInvalidKey)

	// 签名应该失败
	_, err = client.SignRSA([]byte("test"))
	assert.ErrorIs(t, err, ErrInvalidKey)

	// 验证应该失败
	err = client.VerifyRSA([]byte("test"), []byte("sig"))
	assert.ErrorIs(t, err, ErrInvalidKey)
}

func TestClient_JWT(t *testing.T) {
	client, err := NewClient(&Config{
		JWTSecret: "my-test-secret-key",
		JWTExpiry: time.Hour,
	})
	require.NoError(t, err)

	tests := []struct {
		name    string
		claims  JWTClaims
		wantErr bool
	}{
		{
			name: "valid claims with subject",
			claims: JWTClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "user123",
				},
			},
			wantErr: false,
		},
		{
			name: "valid claims with custom fields",
			claims: JWTClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:  "user456",
					Issuer:   "test-issuer",
					Audience: jwt.ClaimStrings{"test-audience"},
				},
				Custom: map[string]interface{}{
					"role": "admin",
					"org":  "acme",
				},
			},
			wantErr: false,
		},
		{
			name: "valid claims with explicit expiry",
			claims: JWTClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "user789",
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * time.Hour)),
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := client.SignJWT(tt.claims)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotEmpty(t, token)

			// 解析并验证
			parsed, err := client.ParseJWT(token)
			assert.NoError(t, err)
			assert.Equal(t, tt.claims.Subject, parsed.Subject)
			assert.Equal(t, tt.claims.Issuer, parsed.Issuer)
			assert.NotNil(t, parsed.ExpiresAt)

			// 检查自定义字段
			if tt.claims.Custom != nil {
				for k, v := range tt.claims.Custom {
					assert.Equal(t, v, parsed.Custom[k])
				}
			}
		})
	}
}

func TestClient_JWT_InvalidToken(t *testing.T) {
	client, err := NewClient(&Config{
		JWTSecret: "my-test-secret-key",
		JWTExpiry: time.Hour,
	})
	require.NoError(t, err)

	tests := []struct {
		name  string
		token string
		errIs error
	}{
		{
			name:  "empty token",
			token: "",
			errIs: ErrInvalidToken,
		},
		{
			name:  "malformed token",
			token: "invalid.token.here",
			errIs: ErrInvalidToken,
		},
		{
			name:  "token with wrong signature",
			token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyMTIzIn0.wrongsignature",
			errIs: ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.ParseJWT(tt.token)
			assert.ErrorIs(t, err, tt.errIs)
		})
	}
}

func TestClient_JWT_ExpiredToken(t *testing.T) {
	client, err := NewClient(&Config{
		JWTSecret: "my-test-secret-key",
		JWTExpiry: time.Hour,
	})
	require.NoError(t, err)

	// 创建过期的 token
	expiredClaims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}

	token, err := client.SignJWT(expiredClaims)
	require.NoError(t, err)

	_, err = client.ParseJWT(token)
	assert.ErrorIs(t, err, ErrTokenExpired)
}

func TestClient_JWT_NoSecret(t *testing.T) {
	client, err := NewClient(&Config{
		AESKey: "this-is-a-32-byte-key-for-aes256",
	})
	require.NoError(t, err)

	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "user123",
		},
	}

	_, err = client.SignJWT(claims)
	assert.ErrorIs(t, err, ErrInvalidKey)

	_, err = client.ParseJWT("some.token.here")
	assert.ErrorIs(t, err, ErrInvalidKey)
}

func TestGetClient(t *testing.T) {
	// 保存当前默认客户端
	oldClient := defaultClient
	defer func() { defaultClient = oldClient }()

	// 测试 nil 客户端
	defaultClient = nil
	client := GetClient()
	assert.Nil(t, client)

	// 配置客户端
	privateKeyPEM, publicKeyPEM, err := generateTestRSAKey()
	require.NoError(t, err)

	config := &Config{
		AESKey:        "this-is-a-32-byte-key-for-aes256",
		RSAPrivateKey: privateKeyPEM,
		RSAPublicKey:  publicKeyPEM,
		JWTSecret:     "my-test-secret-key",
		JWTExpiry:     time.Hour,
	}

	err = Configure(config)
	require.NoError(t, err)

	// 获取客户端
	client = GetClient()
	assert.NotNil(t, client)

	// 验证客户端可以正常工作
	plaintext := []byte("test data")
	encrypted, err := client.EncryptAES(plaintext)
	assert.NoError(t, err)

	decrypted, err := client.DecryptAES(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}
