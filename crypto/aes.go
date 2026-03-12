package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

func EncryptAES(plaintext []byte) ([]byte, error) {
	if defaultClient == nil {
		return nil, ErrMissingClient
	}
	return defaultClient.EncryptAES(plaintext)
}

func DecryptAES(ciphertext []byte) ([]byte, error) {
	if defaultClient == nil {
		return nil, ErrMissingClient
	}
	return defaultClient.DecryptAES(ciphertext)
}

func EncryptAESWithKey(plaintext []byte, key string) ([]byte, error) {
	return encryptAES(plaintext, []byte(key))
}

func DecryptAESWithKey(ciphertext []byte, key string) ([]byte, error) {
	return decryptAES(ciphertext, []byte(key))
}

func EncryptAESString(plaintext string) (string, error) {
	encrypted, err := EncryptAES([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

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
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

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
	return gcm.Open(nil, nonce, ciphertext, nil)
}
