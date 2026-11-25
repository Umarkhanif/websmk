package secure

import (
	"strings"
	"testing"
)

func TestCipherEncryptsAndDecrypts(t *testing.T) {
	cipher, err := NewCipher("kunci-uji-yang-panjang-dan-acak-123456")
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	first, err := cipher.Encrypt("GEZDGNBVGY3TQOJQ")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	second, err := cipher.Encrypt("GEZDGNBVGY3TQOJQ")
	if err != nil {
		t.Fatalf("encrypt second: %v", err)
	}
	if !IsEncrypted(first) || first == second {
		t.Fatal("each value must use a fresh nonce and version prefix")
	}
	if strings.Contains(first, "GEZDGNBVGY3TQOJQ") {
		t.Fatal("ciphertext must not contain the plaintext")
	}
	plain, err := cipher.Decrypt(first)
	if err != nil || plain != "GEZDGNBVGY3TQOJQ" {
		t.Fatalf("decrypt = %q, %v", plain, err)
	}
}

func TestCipherRejectsWrongKeyAndLeavesLegacyPlaintext(t *testing.T) {
	first, _ := NewCipher("kunci-pertama-yang-panjang-123456789")
	second, _ := NewCipher("kunci-kedua-yang-berbeda-123456789")
	encrypted, _ := first.Encrypt("secret")
	if _, err := second.Decrypt(encrypted); err == nil {
		t.Fatal("wrong key must fail closed")
	}
	legacy, err := first.Decrypt("PLAINTEXT-LEGACY")
	if err != nil || legacy != "PLAINTEXT-LEGACY" {
		t.Fatalf("legacy decrypt = %q, %v", legacy, err)
	}
}
