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
