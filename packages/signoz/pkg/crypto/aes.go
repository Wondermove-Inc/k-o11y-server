package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

const envKeyName = "K_O11Y_ENCRYPTION_KEY"

// GetEncryptionKey reads the AES-256 key from environment variable.
// Returns error if not set or invalid length.
func GetEncryptionKey() ([]byte, error) {
	keyHex := os.Getenv(envKeyName)
	if keyHex == "" {
		return nil, fmt.Errorf("encryption key not configured: set %s environment variable", envKeyName)
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid encryption key format: must be hex string")
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("invalid encryption key length: must be 32 bytes (64 hex chars), got %d bytes", len(key))
	}

	return key, nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns hex-encoded ciphertext (nonce + encrypted data).
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	key, err := GetEncryptionKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// Decrypt decrypts hex-encoded AES-256-GCM ciphertext.
func Decrypt(ciphertextHex string) (string, error) {
	if ciphertextHex == "" {
		return "", nil
	}

	key, err := GetEncryptionKey()
	if err != nil {
		return "", err
	}

	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", fmt.Errorf("invalid ciphertext format: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(plaintext), nil
}

// MaskSecret returns masked version of a secret string.
func MaskSecret(s string) string {
	if s == "" {
		return ""
	}
	return "****"
}
