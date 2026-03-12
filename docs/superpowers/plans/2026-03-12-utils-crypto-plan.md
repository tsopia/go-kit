# Utils 补充与 Crypto 包实现计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 utils 包注册到 capabilities.yaml，并创建新的 crypto 包提供 AES/RSA/JWT 加解密能力。

**Architecture:** 遵循 SDK 封装规范（Configure + Helper 模式），crypto 包包含 config/errors/aes/rsa/jwt/api 等模块，支持全局默认客户端和自定义客户端两种方式。

**Tech Stack:** Go 标准库（crypto/aes, crypto/rsa）+ github.com/golang-jwt/jwt/v5

---

## Chunk 1: 更新 capabilities.yaml（Utils 补充）

### Task 1.1: 读取现有 capabilities.yaml

**Files:**
- Read: `.ai/capabilities.yaml`

- [ ] **Step 1: 读取 capabilities.yaml 了解现有格式**

```bash
cat .ai/capabilities.yaml
```

Expected: 看到现有的 capabilities 列表，包括 kit、database、cfg 等包

---

### Task 1.2: 添加 utils 能力定义

**Files:**
- Modify: `.ai/capabilities.yaml`

- [ ] **Step 2: 在 capabilities 列表末尾添加 utils 定义**

参考 spec: `docs/superpowers/specs/2026-03-12-utils-crypto-design.md` Section 1.1

```yaml
  - name: utils
    description: 通用工具函数（字符串/数字/时间/文件/加密/验证/JSON/反射/切片/Map）
    import: github.com/tsopia/go-kit/utils
    scenarios:
      - name: 字符串工具
        snippet: utils.IsEmpty(str), utils.CamelToSnake(str), utils.MaskString(str, 0, 3, '*')
      - name: 加密工具
        snippet: utils.MD5Hash(data), utils.SHA256Hash(data), utils.GenerateSecureToken(32)
      - name: 验证工具
        snippet: utils.IsValidEmail(email), utils.IsStrongPassword(pwd)
      - name: 切片工具
        snippet: utils.ContainsString(slice, item), utils.UniqueStrings(slice)
    dependencies: []
```

---

### Task 1.3: 添加 crypto 能力定义（预留）

**Files:**
- Modify: `.ai/capabilities.yaml`

- [ ] **Step 3: 在 utils 定义后添加 crypto 定义**

```yaml
  - name: crypto
    description: 加解密与 JWT（AES/RSA 加解密、JWT 签名验证）
    import: github.com/tsopia/go-kit/crypto
    scenarios:
      - name: 配置 crypto
        snippet: |
          crypto.Configure(&crypto.Config{
              AESKey: "32-byte-key-for-aes-256-gcm!",
          })
      - name: AES 加密
        snippet: |
          encrypted, err := crypto.EncryptAES([]byte("secret"))
      - name: AES 解密
        snippet: |
          decrypted, err := crypto.DecryptAES(encrypted)
      - name: JWT 签名
        snippet: |
          token, err := crypto.SignJWT(crypto.JWTClaims{Subject: "user123"})
      - name: JWT 验证
        snippet: |
          claims, err := crypto.ParseJWT(token)
    dependencies: []
```

---

### Task 1.4: 验证 YAML 格式

**Files:**
- Read: `.ai/capabilities.yaml`

- [ ] **Step 4: 使用 yq 或在线工具验证 YAML 格式**

```bash
# 使用 yq 验证
yq '.' .ai/capabilities.yaml > /dev/null && echo "YAML valid"
```

Expected: 输出 "YAML valid"

Alternative: 复制到 https://www.yamllint.com/ 验证

---

### Task 1.5: 运行 gokit list 验证

**Files:**
- None

- [ ] **Step 5: 运行 gokit list 确认两个能力都显示**

```bash
go run ./cmd/gokit list
```

Expected: 输出列表中包含 `utils` 和 `crypto`

---

### Task 1.6: 提交

**Files:**
- Modify: `.ai/capabilities.yaml`

- [ ] **Step 6: Commit changes**

```bash
git add .ai/capabilities.yaml
git commit -m "feat: register utils and crypto in capabilities.yaml

Add utils and crypto package definitions to capabilities.yaml.
Utils provides general utilities (strings, crypto helpers, validation).
Crypto provides AES/RSA encryption and JWT signing/verification.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Chunk 2: Crypto 包基础结构

### Task 2.1: 创建 crypto 目录

**Files:**
- Create: `crypto/` directory

- [ ] **Step 1: 创建 crypto 包目录**

```bash
mkdir -p crypto
```

---

### Task 2.2: 创建 config.go

**Files:**
- Create: `crypto/config.go`

- [ ] **Step 2: 编写 config.go - 配置结构、默认化、校验**

```go
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
```

---

### Task 2.3: 创建 errors.go

**Files:**
- Create: `crypto/errors.go`

- [ ] **Step 3: 编写 errors.go - 包级错误定义**

```go
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
```

---

### Task 2.4: 创建 doc.go

**Files:**
- Create: `crypto/doc.go`

- [ ] **Step 4: 编写 doc.go - 包文档**

```go
// Package crypto 提供加解密与 JWT 能力
//
// 使用示例：
//
//	// 配置全局客户端
//	crypto.Configure(&crypto.Config{
//	    AESKey: "32-byte-key-for-aes-256-gcm!",
//	})
//
//	// AES 加密
//	encrypted, err := crypto.EncryptAES([]byte("secret"))
//	if err != nil {
//	    return err
//	}
//
//	// AES 解密
//	decrypted, err := crypto.DecryptAES(encrypted)
//
package crypto
```

---

### Task 2.5: 创建测试

**Files:**
- Create: `crypto/config_test.go`

- [ ] **Step 5: 编写 config_test.go - 配置测试**

```go
package crypto

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.JWTExpiry != 24*time.Hour {
		t.Errorf("expected default JWTExpiry to be 24h, got %v", cfg.JWTExpiry)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "valid config",
			config:  &Config{AESKey: "test-key"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
```

---

### Task 2.6: 运行测试

**Files:**
- None

- [ ] **Step 6: 运行测试确保通过**

```bash
go test -v ./crypto/...
```

Expected: `ok      github.com/tsopia/go-kit/crypto`

---

### Task 2.7: 提交

**Files:**
- Create: `crypto/config.go`
- Create: `crypto/errors.go`
- Create: `crypto/doc.go`
- Create: `crypto/config_test.go`

- [ ] **Step 7: Commit changes**

```bash
git add crypto/
git commit -m "feat(crypto): add package foundation

- Add Config with AES/RSA/JWT settings
- Add package error definitions
- Add doc.go with package documentation
- Add config tests

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Chunk 3: AES 加解密实现

### Task 3.1: 创建 aes.go

**Files:**
- Create: `crypto/aes.go`

- [ ] **Step 1: 编写 aes.go - AES-256-GCM 实现**

```go
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// EncryptAES 使用全局客户端的 AES-256-GCM 加密
func EncryptAES(plaintext []byte) ([]byte, error) {
	if defaultClient == nil {
		return nil, ErrMissingClient
	}
	return defaultClient.EncryptAES(plaintext)
}

// DecryptAES 使用全局客户端的 AES-256-GCM 解密
func DecryptAES(ciphertext []byte) ([]byte, error) {
	if defaultClient == nil {
		return nil, ErrMissingClient
	}
	return defaultClient.DecryptAES(ciphertext)
}

// EncryptAESWithKey 使用指定密钥加密（多租户场景）
func EncryptAESWithKey(plaintext []byte, key string) ([]byte, error) {
	ciphertext, err := encryptAES(plaintext, []byte(key))
	if err != nil {
		return nil, fmt.Errorf("encrypt with key: %w", err)
	}
	return ciphertext, nil
}

// DecryptAESWithKey 使用指定密钥解密（多租户场景）
func DecryptAESWithKey(ciphertext []byte, key string) ([]byte, error) {
	plaintext, err := decryptAES(ciphertext, []byte(key))
	if err != nil {
		return nil, fmt.Errorf("decrypt with key: %w", err)
	}
	return plaintext, nil
}

// EncryptAESString 加密并返回 base64 字符串
func EncryptAESString(plaintext string) (string, error) {
	encrypted, err := EncryptAES([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptAESString 解密 base64 字符串
func DecryptAESString(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}
	decrypted, err := DecryptAES(data)
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}

// encryptAES 使用 AES-256-GCM 加密（内部实现）
func encryptAES(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decryptAES 使用 AES-256-GCM 解密（内部实现）
func decryptAES(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidCiphertext
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}
```

---

### Task 3.2: 创建 aes_test.go

**Files:**
- Create: `crypto/aes_test.go`

- [ ] **Step 2: 编写 aes_test.go - Table-driven 测试**

```go
package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptAES(t *testing.T) {
	// 设置测试用的全局客户端
	Configure(
		&Config{
			AESKey: "this-is-32-byte-key-for-test!!",
		})

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{
			name:      "正常文本",
			plaintext: []byte("hello world"),
		},
		{
			name:      "空字符串",
			plaintext: []byte(""),
		},
		{
			name:      "中文内容",
			plaintext: []byte("你好世界，这是测试内容"),
		},
		{
			name:      "二进制数据",
			plaintext: []byte{0x00, 0x01, 0x02, 0x03, 0xff, 0xfe},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := EncryptAES(tt.plaintext)
			if err != nil {
				t.Fatalf("EncryptAES failed: %v", err)
			}

			decrypted, err := DecryptAES(encrypted)
			if err != nil {
				t.Fatalf("DecryptAES failed: %v", err)
			}

			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("decrypted != plaintext, got %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptAESWithKey(t *testing.T) {
	key := "this-is-32-byte-key-for-test!!"
	plaintext := []byte("test data with custom key")

	encrypted, err := EncryptAESWithKey(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptAESWithKey failed: %v", err)
	}

	decrypted, err := DecryptAESWithKey(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptAESWithKey failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted != plaintext")
	}
}

func TestEncryptAES_MissingClient(t *testing.T) {
	// 清除全局客户端
	defaultClient = nil

	_, err := EncryptAES([]byte("test"))
	if err != ErrMissingClient {
		t.Errorf("expected ErrMissingClient, got %v", err)
	}
}

func TestEncryptDecryptAESString(t *testing.T) {
	Configure(
		&Config{
			AESKey: "this-is-32-byte-key-for-test!!",
		})

	plaintext := "Hello, World! 你好世界"

	encrypted, err := EncryptAESString(plaintext)
	if err != nil {
		t.Fatalf("EncryptAESString failed: %v", err)
	}

	decrypted, err := DecryptAESString(encrypted)
	if err != nil {
		t.Fatalf("DecryptAESString failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("decrypted != plaintext, got %s, want %s", decrypted, plaintext)
	}
}
```

---

### Task 3.3: 运行测试

**Files:**
- None

- [ ] **Step 3: 运行 AES 测试**

```bash
go test -v ./crypto/... -run TestEncrypt
```

Expected: 所有测试通过

---

### Task 3.4: 提交

**Files:**
- Create: `crypto/aes.go`
- Create: `crypto/aes_test.go`

- [ ] **Step 4: Commit changes**

```bash
git add crypto/aes.go crypto/aes_test.go
git commit -m "feat(crypto): add AES-256-GCM encryption

- Add EncryptAES/DecryptAES with global client
- Add EncryptAESWithKey/DecryptAESWithKey for multi-tenant
- Add EncryptAESString/DecryptAESString for string convenience
- Comprehensive table-driven tests

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Chunk 4: RSA 加解密与签名实现

### Task 4.1: 创建 rsa.go

**Files:**
- Create: `crypto/rsa.go`

- [ ] **Step 1: 编写 rsa.go - RSA-OAEP 加密与 RSA-PSS 签名**

```go
package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

// EncryptRSA 使用全局客户端的 RSA-OAEP 加密
func EncryptRSA(plaintext []byte) ([]byte, error) {
	if defaultClient == nil {
		return nil, ErrMissingClient
	}
	return defaultClient.EncryptRSA(plaintext)
}

// DecryptRSA 使用全局客户端的 RSA-OAEP 解密
func DecryptRSA(ciphertext []byte) ([]byte, error) {
	if defaultClient == nil {
		return nil, ErrMissingClient
	}
	return defaultClient.DecryptRSA(ciphertext)
}

// EncryptRSAWithKey 使用指定公钥加密（多租户场景）
func EncryptRSAWithKey(plaintext []byte, publicKey []byte) ([]byte, error) {
	pub, err := parseRSAPublicKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, plaintext, nil)
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}
	return ciphertext, nil
}

// DecryptRSAWithKey 使用指定私钥解密（多租户场景）
func DecryptRSAWithKey(ciphertext []byte, privateKey []byte) ([]byte, error) {
	priv, err := parseRSAPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plaintext, nil
}

// SignRSA 使用全局客户端的 RSA-PSS 签名
func SignRSA(data []byte) ([]byte, error) {
	if defaultClient == nil {
		return nil, ErrMissingClient
	}
	return defaultClient.SignRSA(data)
}

// VerifyRSA 使用全局客户端的 RSA-PSS 验证签名
func VerifyRSA(data, signature []byte) error {
	if defaultClient == nil {
		return ErrMissingClient
	}
	return defaultClient.VerifyRSA(data, signature)
}

// SignRSAWithKey 使用指定私钥签名（多租户场景）
func SignRSAWithKey(data []byte, privateKey []byte) ([]byte, error) {
	priv, err := parseRSAPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	hash := sha256.Sum256(data)
	signature, err := rsa.SignPSS(rand.Reader, priv, crypto.SHA256, hash[:], nil)
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}
	return signature, nil
}

// VerifyRSAWithKey 使用指定公钥验证签名（多租户场景）
func VerifyRSAWithKey(data, signature []byte, publicKey []byte) error {
	pub, err := parseRSAPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}

	hash := sha256.Sum256(data)
	err = rsa.VerifyPSS(pub, crypto.SHA256, hash[:], signature, nil)
	if err != nil {
		return ErrInvalidSignature
	}
	return nil
}

// parseRSAPrivateKey 解析 PEM 格式的 RSA 私钥
func parseRSAPrivateKey(key []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, ErrInvalidKey
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// 尝试 PKCS8 格式
		keyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		var ok bool
		priv, ok = keyInterface.(*rsa.PrivateKey)
		if !ok {
			return nil, ErrInvalidKey
		}
	}
	return priv, nil
}

// parseRSAPublicKey 解析 PEM 格式的 RSA 公钥
func parseRSAPublicKey(key []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, ErrInvalidKey
	}

	pub, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		// 尝试 PKIX 格式
		keyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse public key: %w", err)
		}
		var ok bool
		pub, ok = keyInterface.(*rsa.PublicKey)
		if !ok {
			return nil, ErrInvalidKey
		}
	}
	return pub, nil
}
```

---

### Task 4.2: 创建 rsa_test.go

**Files:**
- Create: `crypto/rsa_test.go`

- [ ] **Step 2: 编写 rsa_test.go - Table-driven 测试**

```go
package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

// generateTestRSAKey 生成测试用的 RSA 密钥对
func generateTestRSAKey(t *testing.T) (privateKey, publicKey []byte) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	privKeyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(priv),
		},
	)

	pubKeyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(&priv.PublicKey),
		},
	)

	return privKeyPEM, pubKeyPEM
}

func TestEncryptDecryptRSAWithKey(t *testing.T) {
	privKey, pubKey := generateTestRSAKey(t)
	plaintext := []byte("hello rsa world")

	encrypted, err := EncryptRSAWithKey(plaintext, pubKey)
	if err != nil {
		t.Fatalf("EncryptRSAWithKey failed: %v", err)
	}

	decrypted, err := DecryptRSAWithKey(encrypted, privKey)
	if err != nil {
		t.Fatalf("DecryptRSAWithKey failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted != plaintext")
	}
}

func TestSignVerifyRSAWithKey(t *testing.T) {
	privKey, pubKey := generateTestRSAKey(t)
	data := []byte("data to be signed")

	signature, err := SignRSAWithKey(data, privKey)
	if err != nil {
		t.Fatalf("SignRSAWithKey failed: %v", err)
	}

	err = VerifyRSAWithKey(data, signature, pubKey)
	if err != nil {
		t.Fatalf("VerifyRSAWithKey failed: %v", err)
	}
}

func TestVerifyRSAWithKey_InvalidSignature(t *testing.T) {
	privKey, pubKey := generateTestRSAKey(t)
	data := []byte("data to be signed")
	wrongData := []byte("wrong data")

	signature, err := SignRSAWithKey(data, privKey)
	if err != nil {
		t.Fatalf("SignRSAWithKey failed: %v", err)
	}

	// 使用错误的数据验证应该失败
	err = VerifyRSAWithKey(wrongData, signature, pubKey)
	if err != ErrInvalidSignature {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestEncryptRSA_MissingClient(t *testing.T) {
	defaultClient = nil

	_, err := EncryptRSA([]byte("test"))
	if err != ErrMissingClient {
		t.Errorf("expected ErrMissingClient, got %v", err)
	}
}
```

---

### Task 4.3: 运行测试

**Files:**
- None

- [ ] **Step 3: 运行 RSA 测试**

```bash
go test -v ./crypto/... -run TestRSA
```

Expected: 所有测试通过

---

### Task 4.4: 提交

**Files:**
- Create: `crypto/rsa.go`
- Create: `crypto/rsa_test.go`

- [ ] **Step 4: Commit changes**

```bash
git add crypto/rsa.go crypto/rsa_test.go
git commit -m "feat(crypto): add RSA encryption and signing

- Add EncryptRSA/DecryptRSA with global client
- Add EncryptRSAWithKey/DecryptRSAWithKey for multi-tenant
- Add SignRSA/VerifyRSA with RSA-PSS
- Add SignRSAWithKey/VerifyRSAWithKey for multi-tenant
- Support PKCS1 and PKCS8 private key formats
- Support PKIX public key format
- Comprehensive table-driven tests

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Chunk 5: JWT 生成与验证实现

### Task 5.1: 添加 jwt 依赖

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: 添加 jwt/v5 依赖**

```bash
go get github.com/golang-jwt/jwt/v5
```

Expected: go.mod 中添加 `github.com/golang-jwt/jwt/v5 v5.2.0`

---

### Task 5.2: 创建 jwt.go

**Files:**
- Create: `crypto/jwt.go`

- [ ] **Step 2: 编写 jwt.go - JWT 签名与验证**

```go
package crypto

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims JWT Claims 结构
type JWTClaims struct {
	jwt.RegisteredClaims
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// SignJWT 使用全局客户端和默认算法生成 JWT Token
func SignJWT(claims JWTClaims) (string, error) {
	if defaultClient == nil {
		return "", ErrMissingClient
	}
	return defaultClient.SignJWT(claims)
}

// SignJWTWithAlg 使用全局客户端和指定算法生成 JWT Token
func SignJWTWithAlg(claims JWTClaims, alg string) (string, error) {
	if defaultClient == nil {
		return "", ErrMissingClient
	}
	return defaultClient.SignJWTWithAlg(claims, alg)
}

// ParseJWT 使用全局客户端解析并验证 JWT Token
func ParseJWT(token string) (*JWTClaims, error) {
	if defaultClient == nil {
		return nil, ErrMissingClient
	}
	return defaultClient.ParseJWT(token)
}

// signJWTWithSecret 使用 HMAC 密钥签名
func signJWTWithSecret(claims JWTClaims, secret string, method jwt.SigningMethod) (string, error) {
	token := jwt.NewWithClaims(method, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return tokenString, nil
}

// parseJWTWithSecret 使用 HMAC 密钥解析
func parseJWTWithSecret(tokenString string, secret string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrUnsupportedAlg
		}
		return []byte(secret), nil
	})

	if err != nil {
		if err == jwt.ErrTokenExpired {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, ErrInvalidToken
}

// setDefaultExpiry 设置默认过期时间
func setDefaultExpiry(claims *JWTClaims, expiry time.Duration) {
	if claims.ExpiresAt == nil {
		claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(expiry))
	}
}
```

---

### Task 5.3: 创建 jwt_test.go

**Files:**
- Create: `crypto/jwt_test.go`

- [ ] **Step 3: 编写 jwt_test.go - Table-driven 测试**

```go
package crypto

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestSignParseJWT(t *testing.T) {
	Configure(&Config{
		JWTSecret: "my-test-secret-key",
		JWTExpiry: time.Hour,
	})

	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "user123",
		},
		Custom: map[string]interface{}{
			"role": "admin",
		},
	}

	token, err := SignJWT(claims)
	if err != nil {
		t.Fatalf("SignJWT failed: %v", err)
	}

	parsed, err := ParseJWT(token)
	if err != nil {
		t.Fatalf("ParseJWT failed: %v", err)
	}

	if parsed.Subject != "user123" {
		t.Errorf("subject mismatch: got %s, want user123", parsed.Subject)
	}

	if parsed.Custom["role"] != "admin" {
		t.Errorf("custom role mismatch")
	}
}

func TestParseJWT_Expired(t *testing.T) {
	Configure(&Config{
		JWTSecret: "my-test-secret-key",
	})

	// 创建一个已过期的 token
	expiredClaims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}

	token, err := SignJWT(expiredClaims)
	if err != nil {
		t.Fatalf("SignJWT failed: %v", err)
	}

	_, err = ParseJWT(token)
	if err != ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestSignJWT_MissingClient(t *testing.T) {
	defaultClient = nil

	_, err := SignJWT(JWTClaims{})
	if err != ErrMissingClient {
		t.Errorf("expected ErrMissingClient, got %v", err)
	}
}

func TestParseJWT_MissingClient(t *testing.T) {
	defaultClient = nil

	_, err := ParseJWT("some.token.here")
	if err != ErrMissingClient {
		t.Errorf("expected ErrMissingClient, got %v", err)
	}
}

func TestSignJWTWithAlg_HS256(t *testing.T) {
	Configure(&Config{
		JWTSecret: "my-test-secret-key",
	})

	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "user456",
		},
	}

	token, err := SignJWTWithAlg(claims, "HS256")
	if err != nil {
		t.Fatalf("SignJWTWithAlg failed: %v", err)
	}

	parsed, err := ParseJWT(token)
	if err != nil {
		t.Fatalf("ParseJWT failed: %v", err)
	}

	if parsed.Subject != "user456" {
		t.Errorf("subject mismatch")
	}
}
```

---

### Task 5.4: 运行测试

**Files:**
- None

- [ ] **Step 4: 运行 JWT 测试**

```bash
go test -v ./crypto/... -run TestJWT
```

Expected: 所有测试通过

---

### Task 5.5: 提交

**Files:**
- Create: `crypto/jwt.go`
- Create: `crypto/jwt_test.go`

- [ ] **Step 5: Commit changes**

```bash
git add go.mod go.sum crypto/jwt.go crypto/jwt_test.go
git commit -m "feat(crypto): add JWT signing and verification

- Add SignJWT/SignJWTWithAlg for token generation
- Add ParseJWT for token verification
- Support HMAC-SHA256 (HS256)
- Auto-set default expiry from config
- Handle expired tokens with ErrTokenExpired
- Comprehensive table-driven tests

Dependencies:
- github.com/golang-jwt/jwt/v5 v5.2.0

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Chunk 6: SDK API 封装

### Task 6.1: 创建 api.go - Client 结构与方法

**Files:**
- Create: `crypto/api.go`

- [ ] **Step 1: 编写 api.go - SDK 封装**

```go
package crypto

import (
	"fmt"
	"sync"

	"github.com/golang-jwt/jwt/v5"
)

// Client 加密客户端
type Client struct {
	config *Config
	mu     sync.RWMutex
}

// 全局默认客户端
var (
	defaultClient *Client
	mu            sync.RWMutex
)

// Configure 配置全局默认客户端
func Configure(cfg *Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	mu.Lock()
	defer mu.Unlock()

	defaultClient = New(cfg)
	return nil
}

// New 创建新的加密客户端
func New(cfg *Config) *Client {
	cfgCopy := *cfg
	cfgCopy.normalize()
	return &Client{
		config: &cfgCopy,
	}
}

// GetClient 获取全局客户端
func GetClient() *Client {
	mu.RLock()
	defer mu.RUnlock()
	return defaultClient
}

// EncryptAES 使用客户端的 AES-256-GCM 加密
func (c *Client) EncryptAES(plaintext []byte) ([]byte, error) {
	if c.config.AESKey == "" {
		return nil, ErrInvalidKey
	}
	return encryptAES(plaintext, []byte(c.config.AESKey))
}

// DecryptAES 使用客户端的 AES-256-GCM 解密
func (c *Client) DecryptAES(ciphertext []byte) ([]byte, error) {
	if c.config.AESKey == "" {
		return nil, ErrInvalidKey
	}
	return decryptAES(ciphertext, []byte(c.config.AESKey))
}

// EncryptRSA 使用客户端的 RSA-OAEP 加密
func (c *Client) EncryptRSA(plaintext []byte) ([]byte, error) {
	return EncryptRSAWithKey(plaintext, c.config.RSAPublicKey)
}

// DecryptRSA 使用客户端的 RSA-OAEP 解密
func (c *Client) DecryptRSA(ciphertext []byte) ([]byte, error) {
	return DecryptRSAWithKey(ciphertext, c.config.RSAPrivateKey)
}

// SignRSA 使用客户端的 RSA-PSS 签名
func (c *Client) SignRSA(data []byte) ([]byte, error) {
	return SignRSAWithKey(data, c.config.RSAPrivateKey)
}

// VerifyRSA 使用客户端的 RSA-PSS 验证签名
func (c *Client) VerifyRSA(data, signature []byte) error {
	return VerifyRSAWithKey(data, signature, c.config.RSAPublicKey)
}

// SignJWT 使用客户端配置生成 JWT Token
func (c *Client) SignJWT(claims JWTClaims) (string, error) {
	setDefaultExpiry(&claims, c.config.JWTExpiry)
	return c.SignJWTWithAlg(claims, "HS256")
}

// SignJWTWithAlg 使用指定算法生成 JWT Token
func (c *Client) SignJWTWithAlg(claims JWTClaims, alg string) (string, error) {
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

	if c.config.JWTSecret == "" {
		return "", ErrInvalidKey
	}

	return signJWTWithSecret(claims, c.config.JWTSecret, method)
}

// ParseJWT 使用客户端配置解析 JWT Token
func (c *Client) ParseJWT(token string) (*JWTClaims, error) {
	if c.config.JWTSecret == "" {
		return nil, ErrInvalidKey
	}
	return parseJWTWithSecret(token, c.config.JWTSecret)
}
```

---

### Task 6.2: 创建 api_test.go

**Files:**
- Create: `crypto/api_test.go`

- [ ] **Step 2: 编写 api_test.go - SDK 测试**

```go
package crypto

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestConfigure(t *testing.T) {
	err := Configure(&Config{
		AESKey:    "this-is-32-byte-key-for-test!!",
		JWTSecret: "test-secret",
	})
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}

	client := GetClient()
	if client == nil {
		t.Fatal("GetClient returned nil")
	}

	// 测试全局客户端可用
	plaintext := []byte("test data")
	encrypted, err := EncryptAES(plaintext)
	if err != nil {
		t.Fatalf("EncryptAES with global client failed: %v", err)
	}

	decrypted, err := DecryptAES(encrypted)
	if err != nil {
		t.Fatalf("DecryptAES with global client failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypted != plaintext")
	}
}

func TestNewClient(t *testing.T) {
	client := New(&Config{
		AESKey:    "this-is-32-byte-key-for-test!!",
		JWTSecret: "test-secret",
		JWTExpiry: 2 * time.Hour,
	})

	// 测试客户端方法
	plaintext := []byte("client test")
	encrypted, err := client.EncryptAES(plaintext)
	if err != nil {
		t.Fatalf("client.EncryptAES failed: %v", err)
	}

	decrypted, err := client.DecryptAES(encrypted)
	if err != nil {
		t.Fatalf("client.DecryptAES failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypted != plaintext")
	}
}

func TestClient_JWT(t *testing.T) {
	client := New(&Config{
		JWTSecret: "my-secret-key",
		JWTExpiry: time.Hour,
	})

	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "user123",
		},
	}

	token, err := client.SignJWT(claims)
	if err != nil {
		t.Fatalf("client.SignJWT failed: %v", err)
	}

	parsed, err := client.ParseJWT(token)
	if err != nil {
		t.Fatalf("client.ParseJWT failed: %v", err)
	}

	if parsed.Subject != "user123" {
		t.Errorf("subject mismatch")
	}
}

func TestClient_SignJWT_InvalidKey(t *testing.T) {
	client := New(&Config{
		JWTSecret: "",
	})

	_, err := client.SignJWT(JWTClaims{})
	if err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey, got %v", err)
	}
}

func TestClient_ParseJWT_InvalidKey(t *testing.T) {
	client := New(&Config{
		JWTSecret: "",
	})

	_, err := client.ParseJWT("some.token")
	if err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey, got %v", err)
	}
}
```

---

### Task 6.3: 运行测试

**Files:**
- None

- [ ] **Step 3: 运行所有测试**

```bash
go test -v ./crypto/...
```

Expected: 所有测试通过

---

### Task 6.4: 提交

**Files:**
- Create: `crypto/api.go`
- Create: `crypto/api_test.go`

- [ ] **Step 4: Commit changes**

```bash
git add crypto/api.go crypto/api_test.go
git commit -m "feat(crypto): add SDK-style API with Client

- Add Client struct with Config
- Add Configure for global client setup
- Add New/GetClient for custom clients
- Implement Client methods: EncryptAES, DecryptAES
- Implement Client methods: EncryptRSA, DecryptRSA
- Implement Client methods: SignRSA, VerifyRSA
- Implement Client methods: SignJWT, ParseJWT
- Comprehensive SDK tests

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Chunk 7: 文档与最终验证

### Task 7.1: 创建 README.md

**Files:**
- Create: `crypto/README.md`

- [ ] **Step 1: 编写 crypto/README.md**

```markdown
# Crypto 包

加解密与 JWT 工具包，提供 AES/RSA 加解密和 JWT 签名验证能力。

## 功能

- **AES-256-GCM**: 对称加密，推荐用于数据加密
- **RSA-OAEP/PSS**: 非对称加密和签名
- **JWT**: HMAC-SHA256 签名验证

## 安装

```bash
go get github.com/tsopia/go-kit/crypto
```

## 使用

### 配置

```go
import "github.com/tsopia/go-kit/crypto"

// 配置全局客户端
crypto.Configure(&crypto.Config{
    AESKey:    "32-byte-key-for-aes-256-gcm!",
    JWTSecret: "my-jwt-secret",
    JWTExpiry: 24 * time.Hour,
})
```

### AES 加密

```go
// 加密
encrypted, err := crypto.EncryptAES([]byte("secret data"))
if err != nil {
    return err
}

// 解密
decrypted, err := crypto.DecryptAES(encrypted)
```

### JWT 签名

```go
// 签名
token, err := crypto.SignJWT(crypto.JWTClaims{
    RegisteredClaims: jwt.RegisteredClaims{
        Subject: "user123",
    },
    Custom: map[string]interface{}{
        "role": "admin",
    },
})

// 验证
claims, err := crypto.ParseJWT(token)
```

### 自定义客户端

```go
client := crypto.New(&crypto.Config{
    AESKey: "custom-key",
})

encrypted, err := client.EncryptAES(data)
```
```

---

### Task 7.2: 更新 AGENTS.md 能力速查表

**Files:**
- Modify: `AGENTS.md`

- [ ] **Step 2: 在 AGENTS.md 的"库能力速查"表格中添加 crypto 行**

找到表格：
```markdown
| 场景 | 使用包 | 典型调用 |
|------|--------|----------|
| 打印日志 | `kit` | `kit.Info(ctx, "msg", fields...)` |
...
| 对象存储 | `storage` | `storage.Upload(ctx, "file", reader)` |
```

在最后一行后添加：
```markdown
| 加解密/JWT | `crypto` | `crypto.EncryptAES(data)`, `crypto.SignJWT(claims)` |
```

---

### Task 7.3: 运行所有测试

**Files:**
- None

- [ ] **Step 3: 运行完整测试套件**

```bash
go test -v ./crypto/...
```

Expected: `ok      github.com/tsopia/go-kit/crypto`

---

### Task 7.4: 运行代码检查

**Files:**
- None

- [ ] **Step 4: 运行 golangci-lint**

```bash
golangci-lint run ./crypto/...
```

Expected: 无错误或警告

---

### Task 7.5: 提交文档更新

**Files:**
- Create: `crypto/README.md`
- Modify: `AGENTS.md`

- [ ] **Step 5: Commit changes**

```bash
git add crypto/README.md AGENTS.md
git commit -m "docs(crypto): add README and update AGENTS.md

- Add comprehensive README with usage examples
- Update AGENTS.md capability table with crypto package
- Complete crypto package documentation

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

### Task 7.6: 最终验证

**Files:**
- None

- [ ] **Step 6: 验证所有组件工作正常**

```bash
# 1. 运行全部测试
go test ./crypto/...

# 2. 构建检查
go build ./crypto/...

# 3. 检查 gokit list
go run ./cmd/gokit list | grep -E "(utils|crypto)"
```

Expected:
- 测试全部通过
- 构建成功
- gokit list 显示 utils 和 crypto

---

## 完成总结

实现完成后的文件结构：

```
crypto/
├── config.go       # 配置结构
├── config_test.go  # 配置测试
├── errors.go       # 错误定义
├── doc.go          # 包文档
├── aes.go          # AES 实现
├── aes_test.go     # AES 测试
├── rsa.go          # RSA 实现
├── rsa_test.go     # RSA 测试
├── jwt.go          # JWT 实现
├── jwt_test.go     # JWT 测试
├── api.go          # SDK API
├── api_test.go     # SDK 测试
└── README.md       # 使用文档
```

---


