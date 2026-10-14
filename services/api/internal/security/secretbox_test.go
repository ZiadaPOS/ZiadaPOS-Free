package security

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
)

func testSecretBox(t *testing.T) *SecretBox {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	box, err := NewSecretBox(base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatal(err)
	}
	return box
}

func TestSecretBoxEncryptDecrypt(t *testing.T) {
	box := testSecretBox(t)
	secret := "example-not-a-real-provider-token"
	ciphertext, err := box.Encrypt(secret)
	if err != nil {
		t.Fatal(err)
	}
	if !IsEncrypted(ciphertext) {
		t.Fatalf("ciphertext lacks version prefix: %q", ciphertext)
	}
	if strings.Contains(ciphertext, secret) {
		t.Fatal("ciphertext contains plaintext credential")
	}
	got, err := box.Decrypt(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if got != secret {
		t.Fatalf("Decrypt() = %q, want original value", got)
	}
}

func TestSecretBoxUsesFreshNonce(t *testing.T) {
	box := testSecretBox(t)
	first, err := box.Encrypt("same-value")
	if err != nil {
		t.Fatal(err)
	}
	second, err := box.Encrypt("same-value")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("equal plaintexts produced identical ciphertext")
	}
}

func TestSecretBoxRejectsTamperingAndPlaintext(t *testing.T) {
	box := testSecretBox(t)
	ciphertext, err := box.Encrypt("example-not-a-real-provider-token")
	if err != nil {
		t.Fatal(err)
	}
	altered := ciphertext[:len(ciphertext)-1] + "A"
	if _, err := box.Decrypt(altered); err == nil {
		t.Fatal("tampered ciphertext unexpectedly decrypted")
	}
	if _, err := box.Decrypt("plaintext-legacy-value"); err == nil {
		t.Fatal("plaintext value unexpectedly decrypted")
	}
}

func TestSecretBoxRequiresCorrectKeyMaterial(t *testing.T) {
	if _, err := NewSecretBox("not-base64!"); err == nil {
		t.Fatal("invalid encoding accepted")
	}
	if _, err := NewSecretBox(base64.StdEncoding.EncodeToString([]byte("short"))); err == nil {
		t.Fatal("short key accepted")
	}
	var unavailable *SecretBox
	if _, err := unavailable.Encrypt("secret"); err == nil {
		t.Fatal("encryption without a configured key succeeded")
	}
}
