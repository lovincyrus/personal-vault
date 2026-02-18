package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

const nonceLen = 12 // 96-bit nonce for GCM

// Encrypt encrypts plaintext with AES-256-GCM using a random 12-byte nonce.
// Returns nonce || ciphertext+tag as a single byte slice.
func Encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, nonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	// nonce || ciphertext+tag
	result := make([]byte, nonceLen+len(ciphertext))
	copy(result, nonce)
	copy(result[nonceLen:], ciphertext)
	return result, nil
}

// Decrypt decrypts data produced by Encrypt. Expects nonce || ciphertext+tag.
func Decrypt(key, data []byte) ([]byte, error) {
	if len(data) < nonceLen+1 {
		return nil, errors.New("ciphertext too short")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	nonce := data[:nonceLen]
	ciphertext := data[nonceLen:]

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	return plaintext, nil
}

// EncryptToBase64 encrypts plaintext and returns base64-encoded ciphertext.
func EncryptToBase64(key, plaintext []byte) (string, error) {
	data, err := Encrypt(key, plaintext)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// DecryptFromBase64 decodes base64 and decrypts.
func DecryptFromBase64(key []byte, encoded string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding base64: %w", err)
	}
	return Decrypt(key, data)
}
