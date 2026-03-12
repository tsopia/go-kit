package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

// generateTestRSAKeys 生成测试用的 RSA 密钥对
func generateTestRSAKeys() (privateKeyPEM, publicKeyPEM []byte, err error) {
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

func TestEncryptRSAWithKey(t *testing.T) {
	privateKeyPEM, publicKeyPEM, err := generateTestRSAKeys()
	if err != nil {
		t.Fatalf("failed to generate test keys: %v", err)
	}

	plaintext := []byte("hello world")

	// 测试加密
	ciphertext, err := EncryptRSAWithKey(plaintext, publicKeyPEM)
	if err != nil {
		t.Fatalf("EncryptRSAWithKey failed: %v", err)
	}

	// 测试解密
	decrypted, err := DecryptRSAWithKey(ciphertext, privateKeyPEM)
	if err != nil {
		t.Fatalf("DecryptRSAWithKey failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypted text doesn't match: got %s, want %s", decrypted, plaintext)
	}
}

func TestDecryptRSAWithKey_InvalidCiphertext(t *testing.T) {
	privateKeyPEM, _, err := generateTestRSAKeys()
	if err != nil {
		t.Fatalf("failed to generate test keys: %v", err)
	}

	// 测试无效密文
	_, err = DecryptRSAWithKey([]byte("invalid ciphertext"), privateKeyPEM)
	if err == nil {
		t.Error("expected error for invalid ciphertext")
	}
}

func TestSignRSAWithKey(t *testing.T) {
	privateKeyPEM, publicKeyPEM, err := generateTestRSAKeys()
	if err != nil {
		t.Fatalf("failed to generate test keys: %v", err)
	}

	data := []byte("data to sign")

	// 测试签名
	signature, err := SignRSAWithKey(data, privateKeyPEM)
	if err != nil {
		t.Fatalf("SignRSAWithKey failed: %v", err)
	}

	// 测试验证
	err = VerifyRSAWithKey(data, signature, publicKeyPEM)
	if err != nil {
		t.Fatalf("VerifyRSAWithKey failed: %v", err)
	}
}

func TestVerifyRSAWithKey_InvalidSignature(t *testing.T) {
	privateKeyPEM, publicKeyPEM, err := generateTestRSAKeys()
	if err != nil {
		t.Fatalf("failed to generate test keys: %v", err)
	}

	data := []byte("data to sign")
	wrongData := []byte("wrong data")

	// 使用正确的私钥签名
	signature, err := SignRSAWithKey(data, privateKeyPEM)
	if err != nil {
		t.Fatalf("SignRSAWithKey failed: %v", err)
	}

	// 尝试用错误的数据验证
	err = VerifyRSAWithKey(wrongData, signature, publicKeyPEM)
	if err != ErrInvalidSignature {
		t.Errorf("expected ErrInvalidSignature, got: %v", err)
	}
}

func TestVerifyRSAWithKey_WrongKey(t *testing.T) {
	privateKeyPEM1, _, err := generateTestRSAKeys()
	if err != nil {
		t.Fatalf("failed to generate test keys: %v", err)
	}

	_, publicKeyPEM2, err := generateTestRSAKeys()
	if err != nil {
		t.Fatalf("failed to generate test keys: %v", err)
	}

	data := []byte("data to sign")

	// 使用第一个私钥签名
	signature, err := SignRSAWithKey(data, privateKeyPEM1)
	if err != nil {
		t.Fatalf("SignRSAWithKey failed: %v", err)
	}

	// 尝试用第二个公钥验证
	err = VerifyRSAWithKey(data, signature, publicKeyPEM2)
	if err != ErrInvalidSignature {
		t.Errorf("expected ErrInvalidSignature, got: %v", err)
	}
}

func TestParseRSAPrivateKey_InvalidPEM(t *testing.T) {
	_, err := parseRSAPrivateKey([]byte("invalid pem"))
	if err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey, got: %v", err)
	}
}

func TestParseRSAPublicKey_InvalidPEM(t *testing.T) {
	_, err := parseRSAPublicKey([]byte("invalid pem"))
	if err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey, got: %v", err)
	}
}

func TestEncryptRSA_MissingClient(t *testing.T) {
	// 保存并恢复默认客户端
	oldClient := defaultClient
	defaultClient = nil
	defer func() { defaultClient = oldClient }()

	_, err := EncryptRSA([]byte("test"))
	if err != ErrMissingClient {
		t.Errorf("expected ErrMissingClient, got: %v", err)
	}
}

func TestDecryptRSA_MissingClient(t *testing.T) {
	oldClient := defaultClient
	defaultClient = nil
	defer func() { defaultClient = oldClient }()

	_, err := DecryptRSA([]byte("test"))
	if err != ErrMissingClient {
		t.Errorf("expected ErrMissingClient, got: %v", err)
	}
}

func TestSignRSA_MissingClient(t *testing.T) {
	oldClient := defaultClient
	defaultClient = nil
	defer func() { defaultClient = oldClient }()

	_, err := SignRSA([]byte("test"))
	if err != ErrMissingClient {
		t.Errorf("expected ErrMissingClient, got: %v", err)
	}
}

func TestVerifyRSA_MissingClient(t *testing.T) {
	oldClient := defaultClient
	defaultClient = nil
	defer func() { defaultClient = oldClient }()

	err := VerifyRSA([]byte("test"), []byte("sig"))
	if err != ErrMissingClient {
		t.Errorf("expected ErrMissingClient, got: %v", err)
	}
}

func TestClient_EncryptRSA(t *testing.T) {
	privateKeyPEM, publicKeyPEM, err := generateTestRSAKeys()
	if err != nil {
		t.Fatalf("failed to generate test keys: %v", err)
	}

	client, err := NewClient(&Config{
		RSAPrivateKey: privateKeyPEM,
		RSAPublicKey:  publicKeyPEM,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	plaintext := []byte("test data")
	ciphertext, err := client.EncryptRSA(plaintext)
	if err != nil {
		t.Fatalf("EncryptRSA failed: %v", err)
	}

	decrypted, err := client.DecryptRSA(ciphertext)
	if err != nil {
		t.Fatalf("DecryptRSA failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypted text doesn't match: got %s, want %s", decrypted, plaintext)
	}
}

func TestClient_SignRSA(t *testing.T) {
	privateKeyPEM, publicKeyPEM, err := generateTestRSAKeys()
	if err != nil {
		t.Fatalf("failed to generate test keys: %v", err)
	}

	client, err := NewClient(&Config{
		RSAPrivateKey: privateKeyPEM,
		RSAPublicKey:  publicKeyPEM,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	data := []byte("test data")
	signature, err := client.SignRSA(data)
	if err != nil {
		t.Fatalf("SignRSA failed: %v", err)
	}

	err = client.VerifyRSA(data, signature)
	if err != nil {
		t.Fatalf("VerifyRSA failed: %v", err)
	}
}

func TestClient_EncryptRSA_NoPublicKey(t *testing.T) {
	privateKeyPEM, _, err := generateTestRSAKeys()
	if err != nil {
		t.Fatalf("failed to generate test keys: %v", err)
	}

	client, err := NewClient(&Config{
		RSAPrivateKey: privateKeyPEM,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// 清除公钥（模拟只有私钥的情况）
	client.rsaPublicKey = nil

	_, err = client.EncryptRSA([]byte("test"))
	if err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey, got: %v", err)
	}
}

func TestClient_DecryptRSA_NoPrivateKey(t *testing.T) {
	_, publicKeyPEM, err := generateTestRSAKeys()
	if err != nil {
		t.Fatalf("failed to generate test keys: %v", err)
	}

	client, err := NewClient(&Config{
		RSAPublicKey: publicKeyPEM,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.DecryptRSA([]byte("test"))
	if err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey, got: %v", err)
	}
}

func TestClient_SignRSA_NoPrivateKey(t *testing.T) {
	_, publicKeyPEM, err := generateTestRSAKeys()
	if err != nil {
		t.Fatalf("failed to generate test keys: %v", err)
	}

	client, err := NewClient(&Config{
		RSAPublicKey: publicKeyPEM,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.SignRSA([]byte("test"))
	if err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey, got: %v", err)
	}
}

func TestClient_VerifyRSA_NoPublicKey(t *testing.T) {
	privateKeyPEM, _, err := generateTestRSAKeys()
	if err != nil {
		t.Fatalf("failed to generate test keys: %v", err)
	}

	client, err := NewClient(&Config{
		RSAPrivateKey: privateKeyPEM,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// 清除公钥
	client.rsaPublicKey = nil

	err = client.VerifyRSA([]byte("test"), []byte("sig"))
	if err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey, got: %v", err)
	}
}

func TestParseRSAPrivateKey_PKCS8(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// 编码为 PKCS#8
	pkcs8Bytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("failed to marshal PKCS#8: %v", err)
	}

	pkcs8PEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8Bytes,
	})

	parsedKey, err := parseRSAPrivateKey(pkcs8PEM)
	if err != nil {
		t.Fatalf("parseRSAPrivateKey failed for PKCS#8: %v", err)
	}

	if parsedKey.N.Cmp(privateKey.N) != 0 {
		t.Error("parsed key doesn't match original")
	}
}

func TestParseRSAPublicKey_PKCS1(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// 编码为 PKCS#1
	pkcs1PEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&privateKey.PublicKey),
	})

	parsedKey, err := parseRSAPublicKey(pkcs1PEM)
	if err != nil {
		t.Fatalf("parseRSAPublicKey failed for PKCS#1: %v", err)
	}

	if parsedKey.N.Cmp(privateKey.N) != 0 {
		t.Error("parsed key doesn't match original")
	}
}

func TestParseRSAPrivateKey_NonRSAKey(t *testing.T) {
	// 创建一个无效的 PEM，类型正确但内容不是 RSA 密钥
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: []byte("invalid key data"),
	}
	invalidPEM := pem.EncodeToMemory(block)

	_, err := parseRSAPrivateKey(invalidPEM)
	if err == nil {
		t.Error("expected error for non-RSA key")
	}
}

func TestParseRSAPublicKey_NonRSAKey(t *testing.T) {
	// 创建一个 ECDSA 密钥来测试
	curve := elliptic.P256()
	ecPrivKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate EC key: %v", err)
	}

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&ecPrivKey.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal EC public key: %v", err)
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	}
	ecPEM := pem.EncodeToMemory(block)

	_, err = parseRSAPublicKey(ecPEM)
	if err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey, got: %v", err)
	}
}

