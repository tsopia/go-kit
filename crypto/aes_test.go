package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptAESWithKey(t *testing.T) {
	// 32字节密钥用于 AES-256-GCM
	key := "this-is-a-32-byte-key-for-aes256"

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
			name:      "long text",
			plaintext: []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."),
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
			encrypted, err := EncryptAESWithKey(tt.plaintext, key)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncryptAESWithKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 解密
			decrypted, err := DecryptAESWithKey(encrypted, key)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecryptAESWithKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 验证解密后的数据与原文相同
			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("DecryptAESWithKey() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptDecryptAESStringWithKey(t *testing.T) {
	key := "this-is-a-32-byte-key-for-aes256"

	tests := []struct {
		name      string
		plaintext string
		wantErr   bool
	}{
		{
			name:      "simple string",
			plaintext: "hello world",
			wantErr:   false,
		},
		{
			name:      "empty string",
			plaintext: "",
			wantErr:   false,
		},
		{
			name:      "unicode string",
			plaintext: "你好世界 🌍 Привет мир",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 使用 WithKey 函数测试字符串加密解密
			encrypted, err := EncryptAESWithKey([]byte(tt.plaintext), key)
			if err != nil {
				t.Errorf("EncryptAESWithKey() error = %v", err)
				return
			}

			decrypted, err := DecryptAESWithKey(encrypted, key)
			if err != nil {
				t.Errorf("DecryptAESWithKey() error = %v", err)
				return
			}

			if string(decrypted) != tt.plaintext {
				t.Errorf("got %q, want %q", string(decrypted), tt.plaintext)
			}
		})
	}
}

func TestEncryptAESWithKey_InvalidKey(t *testing.T) {
	// 无效的密钥长度（非 16/24/32 字节）
	invalidKeys := []string{
		"short",                    // 5 字节
		"17-byte-key-aaaaa",        // 17 字节
		"25-byte-key--aaaaaaaaaa",  // 25 字节
		"33-byte-key--aaaaaaaaaaaaa", // 33 字节
	}

	plaintext := []byte("hello world")

	for _, key := range invalidKeys {
		_, err := EncryptAESWithKey(plaintext, key)
		if err == nil {
			t.Errorf("EncryptAESWithKey() with key %q should return error", key)
		}
	}
}

func TestDecryptAESWithKey_InvalidCiphertext(t *testing.T) {
	key := "this-is-a-32-byte-key-for-aes256"

	invalidCiphertexts := [][]byte{
		{},                         // 空密文
		{0x00},                     // 过短密文
		{0x00, 0x01, 0x02, 0x03},   // 仍然过短
	}

	for _, ct := range invalidCiphertexts {
		_, err := DecryptAESWithKey(ct, key)
		if err == nil {
			t.Errorf("DecryptAESWithKey() with ciphertext %v should return error", ct)
		}
	}
}

func TestEncryptDecryptAESWithKey_DifferentKeys(t *testing.T) {
	key1 := "this-is-a-32-byte-key-for-aes256"
	key2 := "different-32-byte-key-for-aes2"
	plaintext := []byte("secret message")

	// 使用 key1 加密
	encrypted, err := EncryptAESWithKey(plaintext, key1)
	if err != nil {
		t.Fatalf("EncryptAESWithKey() error = %v", err)
	}

	// 使用 key2 解密应该失败
	_, err = DecryptAESWithKey(encrypted, key2)
	if err == nil {
		t.Error("DecryptAESWithKey() with wrong key should return error")
	}

	// 使用 key1 解密应该成功
	decrypted, err := DecryptAESWithKey(encrypted, key1)
	if err != nil {
		t.Errorf("DecryptAESWithKey() with correct key error = %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("got %v, want %v", decrypted, plaintext)
	}
}

func TestEncryptDecryptAES_NoClient(t *testing.T) {
	// 保存当前默认客户端
	oldClient := defaultClient
	defaultClient = nil
	defer func() { defaultClient = oldClient }()

	plaintext := []byte("hello")

	_, err := EncryptAES(plaintext)
	if err != ErrMissingClient {
		t.Errorf("EncryptAES() error = %v, want ErrMissingClient", err)
	}

	_, err = DecryptAES([]byte("ciphertext"))
	if err != ErrMissingClient {
		t.Errorf("DecryptAES() error = %v, want ErrMissingClient", err)
	}
}

func TestEncryptAESString_NoClient(t *testing.T) {
	// 保存当前默认客户端
	oldClient := defaultClient
	defaultClient = nil
	defer func() { defaultClient = oldClient }()

	_, err := EncryptAESString("hello")
	if err != ErrMissingClient {
		t.Errorf("EncryptAESString() error = %v, want ErrMissingClient", err)
	}
}

func TestDecryptAESString_InvalidBase64(t *testing.T) {
	// 需要先设置一个客户端才能测试 DecryptAESString 的 base64 解码错误
	// 这里我们使用 WithKey 函数来避免依赖默认客户端
	// 实际上 DecryptAESString 依赖 DecryptAES，而 DecryptAES 依赖 defaultClient
	// 所以我们测试 base64 解码错误的情况

	// 无效的 base64 字符串
	invalidBase64 := "!!!not-valid-base64!!!"

	// 保存当前默认客户端
	oldClient := defaultClient
	defaultClient = &Client{aesKey: []byte("this-is-a-32-byte-key-for-aes256")}
	defer func() { defaultClient = oldClient }()

	_, err := DecryptAESString(invalidBase64)
	if err == nil {
		t.Error("DecryptAESString() with invalid base64 should return error")
	}
}

func TestEncryptDecryptAES_NonceUniqueness(t *testing.T) {
	key := "this-is-a-32-byte-key-for-aes256"
	plaintext := []byte("same plaintext")

	// 加密两次相同的明文
	encrypted1, err := EncryptAESWithKey(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptAESWithKey() error = %v", err)
	}

	encrypted2, err := EncryptAESWithKey(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptAESWithKey() error = %v", err)
	}

	// 由于 nonce 是随机生成的，两次加密结果应该不同
	if bytes.Equal(encrypted1, encrypted2) {
		t.Error("EncryptAESWithKey() should produce different ciphertexts for same plaintext due to random nonce")
	}

	// 但两次都应该能正确解密
	decrypted1, err := DecryptAESWithKey(encrypted1, key)
	if err != nil {
		t.Errorf("DecryptAESWithKey() error = %v", err)
	}

	decrypted2, err := DecryptAESWithKey(encrypted2, key)
	if err != nil {
		t.Errorf("DecryptAESWithKey() error = %v", err)
	}

	if !bytes.Equal(decrypted1, plaintext) || !bytes.Equal(decrypted2, plaintext) {
		t.Error("Decrypted plaintext should match original")
	}
}
