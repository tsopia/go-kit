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

func EncryptRSA(plaintext []byte) ([]byte, error) {
	if defaultClient == nil {
		return nil, ErrMissingClient
	}
	return defaultClient.EncryptRSA(plaintext)
}

func DecryptRSA(ciphertext []byte) ([]byte, error) {
	if defaultClient == nil {
		return nil, ErrMissingClient
	}
	return defaultClient.DecryptRSA(ciphertext)
}

func EncryptRSAWithKey(plaintext []byte, publicKey []byte) ([]byte, error) {
	pub, err := parseRSAPublicKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, plaintext, nil)
}

func DecryptRSAWithKey(ciphertext []byte, privateKey []byte) ([]byte, error) {
	priv, err := parseRSAPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, ciphertext, nil)
}

func SignRSA(data []byte) ([]byte, error) {
	if defaultClient == nil {
		return nil, ErrMissingClient
	}
	return defaultClient.SignRSA(data)
}

func VerifyRSA(data, signature []byte) error {
	if defaultClient == nil {
		return ErrMissingClient
	}
	return defaultClient.VerifyRSA(data, signature)
}

func SignRSAWithKey(data []byte, privateKey []byte) ([]byte, error) {
	priv, err := parseRSAPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(data)
	return rsa.SignPSS(rand.Reader, priv, crypto.SHA256, hash[:], nil)
}

func VerifyRSAWithKey(data, signature []byte, publicKey []byte) error {
	pub, err := parseRSAPublicKey(publicKey)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(data)
	if err := rsa.VerifyPSS(pub, crypto.SHA256, hash[:], signature, nil); err != nil {
		return ErrInvalidSignature
	}
	return nil
}

func parseRSAPrivateKey(key []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, ErrInvalidKey
	}
	if priv, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return priv, nil
	}
	keyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	priv, ok := keyInterface.(*rsa.PrivateKey)
	if !ok {
		return nil, ErrInvalidKey
	}
	return priv, nil
}

func parseRSAPublicKey(key []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, ErrInvalidKey
	}
	if pub, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return pub, nil
	}
	keyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := keyInterface.(*rsa.PublicKey)
	if !ok {
		return nil, ErrInvalidKey
	}
	return pub, nil
}
