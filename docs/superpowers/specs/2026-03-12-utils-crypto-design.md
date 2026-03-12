# Utils 补充与 Crypto 包设计文档

## 目标

为 go-kit 工具库补充完善：
1. **Utils 包注册** - 将现有的 utils 包补充到 capabilities.yaml
2. **Crypto 包新增** - 提供 JWT、AES、RSA 等加解密能力

---

## 1. Utils 包补充

### 1.1 能力定义

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

### 1.2 更新文件

- `.ai/capabilities.yaml` - 在 `capabilities` 列表中添加 utils 定义

---

## 2. Crypto 包设计

### 2.1 目录结构

```
crypto/
├── config.go       # 配置结构与默认化/校验
├── errors.go       # 包级错误定义
├── aes.go          # AES 对称加密
├── aes_test.go     # AES 测试
├── rsa.go          # RSA 非对称加密与签名
├── rsa_test.go     # RSA 测试
├── jwt.go          # JWT 生成与验证
├── jwt_test.go     # JWT 测试
├── api.go          # SDK 风格高层 API
├── doc.go          # 包文档
└── README.md       # 使用说明
```

### 2.2 架构设计

**SDK 封装模式**：遵循 Configure + Helper 规范

```go
// 1. 配置全局客户端
crypto.Configure(&crypto.Config{
    AESKey: "32-byte-key-for-aes-256-gcm",
})

// 2. 直接使用高层函数
encrypted, err := crypto.EncryptAES(plaintext)
decrypted, err := crypto.DecryptAES(ciphertext)

// 3. 或使用自定义客户端
c, _ := crypto.New(cfg)
encrypted, err := c.EncryptAES(plaintext)
```

### 2.3 配置结构

```go
package crypto

import "time"

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
```

### 2.4 API 设计

#### AES 对称加密

```go
// EncryptAES 使用 AES-256-GCM 加密
func EncryptAES(plaintext []byte) ([]byte, error)
func (c *Client) EncryptAES(plaintext []byte) ([]byte, error)

// DecryptAES 使用 AES-256-GCM 解密
func DecryptAES(ciphertext []byte) ([]byte, error)
func (c *Client) DecryptAES(ciphertext []byte) ([]byte, error)

// EncryptAESWithKey 使用指定密钥加密（多租户场景）
func EncryptAESWithKey(plaintext []byte, key string) ([]byte, error)

// DecryptAESWithKey 使用指定密钥解密（多租户场景）
func DecryptAESWithKey(ciphertext []byte, key string) ([]byte, error)
```

#### RSA 非对称加密与签名

```go
// EncryptRSA 使用 RSA-OAEP 加密
func EncryptRSA(plaintext []byte) ([]byte, error)
func (c *Client) EncryptRSA(plaintext []byte) ([]byte, error)

// EncryptRSAWithKey 使用指定公钥加密（多租户场景）
func EncryptRSAWithKey(plaintext []byte, publicKey []byte) ([]byte, error)

// DecryptRSA 使用 RSA-OAEP 解密
func DecryptRSA(ciphertext []byte) ([]byte, error)
func (c *Client) DecryptRSA(ciphertext []byte) ([]byte, error)

// DecryptRSAWithKey 使用指定私钥解密（多租户场景）
func DecryptRSAWithKey(ciphertext []byte, privateKey []byte) ([]byte, error)

// SignRSA 使用 RSA-PSS 签名
func SignRSA(data []byte) ([]byte, error)
func (c *Client) SignRSA(data []byte) ([]byte, error)

// SignRSAWithKey 使用指定私钥签名（多租户场景）
func SignRSAWithKey(data []byte, privateKey []byte) ([]byte, error)

// VerifyRSA 使用 RSA-PSS 验证签名
func VerifyRSA(data, signature []byte) error
func (c *Client) VerifyRSA(data, signature []byte) error

// VerifyRSAWithKey 使用指定公钥验证签名（多租户场景）
func VerifyRSAWithKey(data, signature []byte, publicKey []byte) error
```

#### JWT 生成与验证

```go
// JWTClaims 标准 Claims 结构
type JWTClaims struct {
    jwt.RegisteredClaims
    Custom map[string]interface{} `json:"custom,omitempty"`
}

// SignJWT 生成 JWT Token
func SignJWT(claims JWTClaims) (string, error)
func SignJWTWithAlg(claims JWTClaims, alg string) (string, error)
func (c *Client) SignJWT(claims JWTClaims) (string, error)
func (c *Client) SignJWTWithAlg(claims JWTClaims, alg string) (string, error)

// ParseJWT 解析并验证 JWT Token
func ParseJWT(token string) (*JWTClaims, error)
func (c *Client) ParseJWT(token string) (*JWTClaims, error)
```

### 2.5 支持的算法

| 功能 | 算法 | 说明 |
|------|------|------|
| 对称加密 | AES-256-GCM | 推荐，带认证标签（GCM 模式同时提供加密和认证） |
| 非对称加密 | RSA-OAEP-SHA256 | 用于密钥交换 |
| 数字签名 | RSA-PSS-SHA256 | 推荐签名方案 |
| JWT | HS256, HS384, HS512 | HMAC 签名 |
| JWT | RS256, RS384, RS512 | RSA 签名 |

**注**：V1 版本仅支持 AES-256-GCM。如后续需要兼容旧系统（AES-CBC），将在 V2 中通过 `EncryptAESWithMode()` 接口扩展。

### 2.6 错误定义

```go
var (
    ErrMissingClient      = errors.New("crypto: 客户端未配置，请先调用 Configure")
    ErrInvalidKey         = errors.New("crypto: 无效的密钥")
    ErrInvalidCiphertext  = errors.New("crypto: 无效的密文")
    ErrInvalidSignature   = errors.New("crypto: 签名验证失败")
    ErrInvalidToken       = errors.New("crypto: 无效的 JWT token")
    ErrTokenExpired       = errors.New("crypto: JWT token 已过期")
    ErrUnsupportedAlg     = errors.New("crypto: 不支持的算法")
)
```

### 2.7 使用示例

```go
package main

import (
    "fmt"
    "time"

    "github.com/tsopia/go-kit/crypto"
)

func main() {
    // 配置 crypto
    crypto.Configure(&crypto.Config{
        AESKey:    "this-is-32-byte-key-for-aes-256!",
        JWTSecret: "my-jwt-secret-key",
        JWTExpiry: 24 * time.Hour,
    })

    // AES 加密
    plaintext := []byte("Hello, World!")
    encrypted, err := crypto.EncryptAES(plaintext)
    if err != nil {
        panic(err)
    }

    decrypted, err := crypto.DecryptAES(encrypted)
    if err != nil {
        panic(err)
    }
    fmt.Println(string(decrypted)) // Hello, World!

    // JWT 签名
    token, err := crypto.SignJWT(crypto.JWTClaims{
        RegisteredClaims: jwt.RegisteredClaims{
            Subject:   "user123",
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
        },
    })
    if err != nil {
        panic(err)
    }

    // JWT 验证
    claims, err := crypto.ParseJWT(token)
    if err != nil {
        panic(err)
    }
    fmt.Println(claims.Subject) // user123
}
```

---

## 3. 任务分解

### Task 1: 更新 capabilities.yaml
- 添加 utils 能力定义（完整定义，包含使用场景）
- 添加 crypto 能力定义（完整定义，包含使用场景）
- 验证 YAML 格式正确
- 运行 `gokit list` 确认两个能力都显示正常

### Task 2: 创建 crypto 包基础结构
- config.go - 配置结构、默认化、校验
- errors.go - 包级错误定义
- doc.go - 包文档

### Task 3: 实现 AES 加解密
- aes.go - AES-256-GCM 实现
- aes_test.go - Table-driven 测试

### Task 4: 实现 RSA 加解密与签名
- rsa.go - RSA-OAEP 加密、RSA-PSS 签名
- rsa_test.go - Table-driven 测试

### Task 5: 实现 JWT 生成与验证
- jwt.go - JWT Sign/Parse 实现
- jwt_test.go - Table-driven 测试

### Task 6: SDK 风格 API 封装
- api.go - Configure、New、全局函数封装

### Task 7: 文档与验证
- README.md - 使用说明
- 运行 `go test ./crypto/...`
- 更新 AGENTS.md 能力速查表

---

## 4. 依赖分析

### 新增外部依赖

```go
// go.mod
require (
    github.com/golang-jwt/jwt/v5 v5.2.0
)
```

- `jwt/v5` - 业界标准 JWT 库，CloudWeGo/Echo 等框架都在用

### 内部依赖

- 无（crypto 包保持零依赖，仅依赖标准库 + jwt 库）

---

## 5. 测试策略

每个功能都需要 Table-driven 测试：

```go
func TestEncryptAES(t *testing.T) {
    tests := []struct {
        name      string
        plaintext []byte
        key       string
        wantErr   bool
    }{
        {
            name:      "正常加密",
            plaintext: []byte("hello world"),
            key:       "32-byte-key-for-aes-256-encrypt",
            wantErr:   false,
        },
        {
            name:      "错误密钥长度",
            plaintext: []byte("hello"),
            key:       "short-key",
            wantErr:   true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 测试代码
        })
    }
}
```
