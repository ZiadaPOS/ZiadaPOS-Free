package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const integrationSecretPrefix = "enc:v1:"

var ErrSecretBoxUnavailable = errors.New("integration credential encryption is not configured")

// SecretBox encrypts provider credentials before they are stored in the database.
// The key must be a base64-encoded, randomly generated 32-byte value.
type SecretBox struct {
	aead cipher.AEAD
}

// NewSecretBox returns nil when no key is configured, allowing installations
// without tenant credentials to boot. Credential writes and use must fail closed
// until a valid key is supplied.
func NewSecretBox(encodedKey string) (*SecretBox, error) {
	encodedKey = strings.TrimSpace(encodedKey)
	if encodedKey == "" {
		return nil, nil
	}
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, errors.New("integration encryption key must be base64-encoded 32-byte data")
	}
	if len(key) != 32 {
		return nil, errors.New("integration encryption key must decode to exactly 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create integration credential cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create integration credential cipher: %w", err)
	}
	return &SecretBox{aead: aead}, nil
}

// IsEncrypted reports whether the value uses this SecretBox's versioned format.
func IsEncrypted(value string) bool { return strings.HasPrefix(value, integrationSecretPrefix) }

// Encrypt protects plaintext using a fresh random nonce. Empty values remain
// empty so owners can explicitly remove an integration credential.
func (b *SecretBox) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if b == nil || b.aead == nil {
		return "", ErrSecretBoxUnavailable
	}
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", errors.New("could not generate integration credential nonce")
	}
	sealed := b.aead.Seal(nil, nonce, []byte(plaintext), []byte(integrationSecretPrefix))
	payload := append(nonce, sealed...)
	return integrationSecretPrefix + base64.RawStdEncoding.EncodeToString(payload), nil
}

// Decrypt opens only values in the versioned encrypted format. Legacy plaintext
// values must be handled by the startup migration, never silently accepted here.
func (b *SecretBox) Decrypt(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if b == nil || b.aead == nil {
		return "", ErrSecretBoxUnavailable
	}
	if !IsEncrypted(value) {
		return "", errors.New("integration credential is not encrypted")
	}
	payload, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(value, integrationSecretPrefix))
	if err != nil || len(payload) < b.aead.NonceSize()+b.aead.Overhead() {
		return "", errors.New("integration credential could not be decrypted")
	}
	nonce, sealed := payload[:b.aead.NonceSize()], payload[b.aead.NonceSize():]
	plaintext, err := b.aead.Open(nil, nonce, sealed, []byte(integrationSecretPrefix))
	if err != nil {
		return "", errors.New("integration credential could not be decrypted")
	}
	return string(plaintext), nil
}
