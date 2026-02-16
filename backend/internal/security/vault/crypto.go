package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// Key context constants for HKDF key derivation.
// Each context produces a distinct 256-bit key from the same master secret.
const (
	KeyContextVault = "vault-encryption"
	KeyContextTOTP  = "totp-encryption"
)

const (
	keyLength   = 32 // AES-256
	nonceLength = 12 // GCM standard nonce
)

var (
	ErrKeyTooShort  = errors.New("vault: derived key must be 32 bytes")
	ErrDecryptFail  = errors.New("vault: decryption failed (corrupted or wrong key)")
	ErrEmptyPayload = errors.New("vault: plaintext must not be empty")
	ErrInvalidData  = errors.New("vault: ciphertext too short or malformed")
)

// DeriveKey derives a 32-byte AES-256 key from a master secret using HKDF-SHA256.
// The context string distinguishes the key purpose (e.g., "vault-encryption", "totp-encryption"),
// ensuring the same master secret produces different keys for different uses.
func DeriveKey(masterSecret []byte, context string) ([]byte, error) {
	if len(masterSecret) == 0 {
		return nil, errors.New("vault: master secret must not be empty")
	}

	reader := hkdf.New(sha256.New, masterSecret, nil, []byte(context))
	key := make([]byte, keyLength)
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, fmt.Errorf("vault: HKDF key derivation failed: %w", err)
	}
	return key, nil
}

// Encrypt encrypts plaintext using AES-256-GCM with a random nonce.
// Returns a base64-encoded string containing nonce + ciphertext.
func Encrypt(plaintext []byte, key []byte) (string, error) {
	if len(key) != keyLength {
		return "", ErrKeyTooShort
	}
	if len(plaintext) == 0 {
		return "", ErrEmptyPayload
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("vault: AES cipher creation failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("vault: GCM creation failed: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("vault: nonce generation failed: %w", err)
	}

	// Seal prepends ciphertext+tag after nonce
	sealed := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt: base64 decode, split nonce, GCM open.
func Decrypt(encoded string, key []byte) ([]byte, error) {
	if len(key) != keyLength {
		return nil, ErrKeyTooShort
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("vault: base64 decode failed: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("vault: AES cipher creation failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("vault: GCM creation failed: %w", err)
	}

	if len(data) < gcm.NonceSize() {
		return nil, ErrInvalidData
	}

	nonce := data[:gcm.NonceSize()]
	ciphertext := data[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptFail
	}
	return plaintext, nil
}
