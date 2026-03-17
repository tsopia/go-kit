# Crypto 包

Crypto 包提供加解密与 JWT 能力，支持 AES-256-GCM 对称加密、RSA-OAEP/PSS 非对称加密和签名、以及 JWT HMAC-SHA256 签名验证。

## 功能列表

- **AES-256-GCM 对称加密** - 提供安全的对称加密解密功能
- **RSA-OAEP/PSS 非对称加密和签名** - 支持 RSA 公钥加密、私钥解密、私钥签名、公钥验证
- **JWT HMAC-SHA256 签名验证** - 支持 JWT Token 的生成与解析验证

## 安装

```bash
go get github.com/tsopia/go-kit/crypto
```

## 使用示例

### 配置全局客户端

```go
import "github.com/tsopia/go-kit/crypto"

// 配置全局客户端
err := crypto.Configure(&crypto.Config{
    AESKey:    "32-byte-key-for-aes-256-gcm!",
    JWTSecret: "your-jwt-secret-key",
    JWTExpiry: 24 * time.Hour,
})
if err != nil {
    log.Fatal(err)
}
```

### AES 加密解密示例

```go
// 加密
plaintext := []byte("secret data")
encrypted, err := crypto.EncryptAES(plaintext)
if err != nil {
    return err
}

// 解密
decrypted, err := crypto.DecryptAES(encrypted)
if err != nil {
    return err
}

// 字符串加密（自动 Base64 编码）
encryptedStr, err := crypto.EncryptAESString("secret data")
if err != nil {
    return err
}

// 字符串解密（自动 Base64 解码）
decryptedStr, err := crypto.DecryptAESString(encryptedStr)
if err != nil {
    return err
}
```

### RSA 加密解密和签名验证示例

```go
// 配置 RSA 密钥
crypto.Configure(&crypto.Config{
    RSAPrivateKey: []byte(privateKeyPEM),
    RSAPublicKey:  []byte(publicKeyPEM),
})

// RSA 加密
plaintext := []byte("secret data")
encrypted, err := crypto.EncryptRSA(plaintext)
if err != nil {
    return err
}

// RSA 解密
decrypted, err := crypto.DecryptRSA(encrypted)
if err != nil {
    return err
}

// RSA 签名
data := []byte("data to sign")
signature, err := crypto.SignRSA(data)
if err != nil {
    return err
}

// RSA 验证签名
err = crypto.VerifyRSA(data, signature)
if err != nil {
    return err // 签名无效
}
```

### JWT 签名验证示例

```go
// 配置 JWT
crypto.Configure(&crypto.Config{
    JWTSecret: "your-secret-key",
    JWTExpiry: 24 * time.Hour,
})

// 生成 JWT Token
claims := crypto.JWTClaims{
    RegisteredClaims: jwt.RegisteredClaims{
        Subject: "user123",
    },
    Custom: map[string]interface{}{
        "role": "admin",
    },
}

token, err := crypto.SignJWT(claims)
if err != nil {
    return err
}

// 使用指定算法签名（支持 HS256/HS384/HS512）
token, err := crypto.SignJWTWithAlg(claims, crypto.JWTAlgHS256)

// 解析 JWT Token
parsedClaims, err := crypto.ParseJWT(token)
if err != nil {
    return err
}

fmt.Println(parsedClaims.Subject) // "user123"
```

### 使用独立客户端

```go
// 创建独立客户端（适用于测试或多实例场景）
client, err := crypto.NewClient(&crypto.Config{
    AESKey:    "32-byte-key-for-aes-256-gcm!",
    JWTSecret: "your-jwt-secret-key",
})
if err != nil {
    return err
}

// 使用客户端方法
encrypted, err := client.EncryptAES([]byte("secret"))
decrypted, err := client.DecryptAES(encrypted)

token, err := client.SignJWT(claims)
parsedClaims, err := client.ParseJWT(token)
```

## API 速查表

### 全局函数

| 函数 | 说明 |
|------|------|
| `Configure(config *Config) error` | 配置全局默认客户端 |
| `GetClient() *Client` | 获取全局默认客户端 |
| `NewClient(config *Config) (*Client, error)` | 创建新的独立客户端 |

### AES 加密

| 函数 | 说明 |
|------|------|
| `EncryptAES(plaintext []byte) ([]byte, error)` | 使用全局客户端加密 |
| `DecryptAES(ciphertext []byte) ([]byte, error)` | 使用全局客户端解密 |
| `EncryptAESString(plaintext string) (string, error)` | 加密并返回 Base64 字符串 |
| `DecryptAESString(ciphertext string) (string, error)` | 解密 Base64 字符串 |
| `EncryptAESWithKey(plaintext []byte, key string) ([]byte, error)` | 使用指定密钥加密 |
| `DecryptAESWithKey(ciphertext []byte, key string) ([]byte, error)` | 使用指定密钥解密 |

### RSA 加密与签名

| 函数 | 说明 |
|------|------|
| `EncryptRSA(plaintext []byte) ([]byte, error)` | 使用全局客户端 RSA-OAEP 加密 |
| `DecryptRSA(ciphertext []byte) ([]byte, error)` | 使用全局客户端 RSA-OAEP 解密 |
| `EncryptRSAWithKey(plaintext []byte, publicKey []byte) ([]byte, error)` | 使用指定公钥加密 |
| `DecryptRSAWithKey(ciphertext []byte, privateKey []byte) ([]byte, error)` | 使用指定私钥解密 |
| `SignRSA(data []byte) ([]byte, error)` | 使用全局客户端 RSA-PSS 签名 |
| `VerifyRSA(data, signature []byte) error` | 使用全局客户端验证签名 |
| `SignRSAWithKey(data []byte, privateKey []byte) ([]byte, error)` | 使用指定私钥签名 |
| `VerifyRSAWithKey(data, signature []byte, publicKey []byte) error` | 使用指定公钥验证签名 |

### JWT 操作

| 函数 | 说明 |
|------|------|
| `SignJWT(claims JWTClaims) (string, error)` | 使用 HS256 签名 JWT |
| `SignJWTWithAlg(claims JWTClaims, alg string) (string, error)` | 使用指定算法签名 JWT |
| `ParseJWT(token string) (*JWTClaims, error)` | 解析并验证 JWT |

**算法常量：** `JWTAlgHS256`、`JWTAlgHS384`、`JWTAlgHS512`

### 客户端方法

| 方法 | 说明 |
|------|------|
| `(c *Client) EncryptAES(plaintext []byte) ([]byte, error)` | AES 加密 |
| `(c *Client) DecryptAES(ciphertext []byte) ([]byte, error)` | AES 解密 |
| `(c *Client) EncryptRSA(plaintext []byte) ([]byte, error)` | RSA-OAEP 加密 |
| `(c *Client) DecryptRSA(ciphertext []byte) ([]byte, error)` | RSA-OAEP 解密 |
| `(c *Client) SignRSA(data []byte) ([]byte, error)` | RSA-PSS 签名 |
| `(c *Client) VerifyRSA(data, signature []byte) error` | RSA-PSS 验证签名 |
| `(c *Client) SignJWT(claims JWTClaims) (string, error)` | JWT 签名（HS256） |
| `(c *Client) SignJWTWithAlg(claims JWTClaims, alg string) (string, error)` | JWT 签名（指定算法） |
| `(c *Client) ParseJWT(token string) (*JWTClaims, error)` | JWT 解析验证 |

### 配置结构

```go
type Config struct {
    AESKey        string        // 32字节用于 AES-256-GCM
    RSAPrivateKey []byte        // PEM 格式私钥
    RSAPublicKey  []byte        // PEM 格式公钥
    JWTSecret     string        // HMAC 密钥
    JWTPrivateKey []byte        // RSA 私钥（用于 RS256）
    JWTPublicKey  []byte        // RSA 公钥（用于 RS256）
    JWTExpiry     time.Duration // 默认过期时间（默认 24h）
}
```
