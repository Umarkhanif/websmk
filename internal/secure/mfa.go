// Package secure provides small crypto helpers for data that must not sit in
// the database as plaintext. TOTP secrets use AES-256-GCM with a random nonce
// per value and a version prefix so future formats can be rotated safely.
package secure

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

const (
	// MFAPrefix marks values produced by the current encryption format.
	MFAPrefix = "v1:"

	mfaAAD = "wesmk:totp-secret:v1"
)

var ErrNoKey = errors.New("MFA encryption key is empty")

// Cipher encrypts and decrypts short secrets. It is safe for concurrent use.
type Cipher struct {
	aead cipher.AEAD
}

// NewCipher derives a 256-bit key from the configured passphrase. Operators
// should supply at least 32 random characters; the hash removes length limits.
func NewCipher(secret string) (*Cipher, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, ErrNoKey
	}
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("init mfa cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("init mfa gcm: %w", err)
	}
	return &Cipher{aead: aead}, nil
}

// Encrypt returns a versioned base64 payload. The nonce is stored in front of
// the ciphertext; it is not secret.
func (c *Cipher) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate mfa nonce: %w", err)
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), []byte(mfaAAD))
	return MFAPrefix + base64.RawStdEncoding.EncodeToString(sealed), nil
}

// Decrypt accepts the versioned format. Legacy plaintext values are returned
// unchanged so existing accounts keep working until the startup re-encrypt pass
// rewrites them.
func (c *Cipher) Decrypt(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if !strings.HasPrefix(value, MFAPrefix) {
		return value, nil
	}
	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(value, MFAPrefix))
	if err != nil {
		return "", fmt.Errorf("decode mfa secret: %w", err)
	}
	nonceSize := c.aead.NonceSize()
	if len(raw) < nonceSize {
		return "", errors.New("mfa secret payload is too short")
	}
	plaintext, err := c.aead.Open(nil, raw[:nonceSize], raw[nonceSize:], []byte(mfaAAD))
	if err != nil {
		return "", errors.New("mfa secret cannot be decrypted with the configured key")
	}
	return string(plaintext), nil
}

// IsEncrypted reports whether a stored value already uses the current format.
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, MFAPrefix)
}
