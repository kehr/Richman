package llm

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// validTestKey is a deterministic 64-char hex string (32 bytes) used to build
// a real GCM cipher for unit tests. It is NOT a secret; it only exists inside
// the test binary.
const validTestKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestNewCryptoFromHex_ValidKey(t *testing.T) {
	c, err := NewCryptoFromHex(validTestKey)
	if err != nil {
		t.Fatalf("NewCryptoFromHex(valid) returned error: %v", err)
	}
	if c == nil {
		t.Fatal("NewCryptoFromHex(valid) returned nil Crypto")
	}
}

func TestNewCryptoFromHex_InvalidLength(t *testing.T) {
	cases := []struct {
		name string
		hex  string
	}{
		{"empty", ""},
		{"too short", "deadbeef"},
		{"63 chars", strings.Repeat("a", 63)},
		{"65 chars", strings.Repeat("a", 65)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewCryptoFromHex(tc.hex)
			if !errors.Is(err, ErrMasterKeyInvalid) {
				t.Fatalf("expected ErrMasterKeyInvalid, got %v", err)
			}
		})
	}
}

func TestNewCryptoFromHex_InvalidHexChars(t *testing.T) {
	// 64 chars but not valid hex (contains "z"). Length passes the first check,
	// so this exercises the hex.DecodeString error branch.
	bad := strings.Repeat("z", 64)
	_, err := NewCryptoFromHex(bad)
	if !errors.Is(err, ErrMasterKeyInvalid) {
		t.Fatalf("expected ErrMasterKeyInvalid for non-hex, got %v", err)
	}
}

func TestCrypto_EncryptDecryptRoundtrip(t *testing.T) {
	c, err := NewCryptoFromHex(validTestKey)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	plaintext := []byte("sk-ant-api03-example-plaintext-key")

	ct, nonce, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt returned error: %v", err)
	}
	if len(nonce) != NonceBytes {
		t.Fatalf("nonce length = %d, want %d", len(nonce), NonceBytes)
	}
	if bytes.Equal(ct, plaintext) {
		t.Fatal("ciphertext equals plaintext; encryption is a no-op")
	}

	got, err := c.Decrypt(ct, nonce)
	if err != nil {
		t.Fatalf("Decrypt returned error: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("decrypted = %q, want %q", got, plaintext)
	}
}

func TestCrypto_EncryptNonceUniqueness(t *testing.T) {
	// Two encryptions of the same plaintext must yield different nonces and
	// different ciphertexts. This is a GCM invariant; verifying it catches any
	// future regression that accidentally fixes the nonce.
	c, err := NewCryptoFromHex(validTestKey)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	plaintext := []byte("repeat-me")
	ct1, nonce1, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt #1: %v", err)
	}
	ct2, nonce2, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt #2: %v", err)
	}
	if bytes.Equal(nonce1, nonce2) {
		t.Fatal("two encryptions produced the same nonce")
	}
	if bytes.Equal(ct1, ct2) {
		t.Fatal("two encryptions produced the same ciphertext")
	}
}

func TestCrypto_DecryptWrongNonce(t *testing.T) {
	c, err := NewCryptoFromHex(validTestKey)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	ct, _, err := c.Encrypt([]byte("data"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	// Nonce of correct length but not the one used at Seal time.
	wrongNonce := bytes.Repeat([]byte{0x42}, NonceBytes)
	_, err = c.Decrypt(ct, wrongNonce)
	if !errors.Is(err, ErrDecryptFailed) {
		t.Fatalf("expected ErrDecryptFailed, got %v", err)
	}
}

func TestCrypto_DecryptShortNonce(t *testing.T) {
	c, err := NewCryptoFromHex(validTestKey)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err = c.Decrypt([]byte("anything"), []byte{0x00, 0x01})
	if !errors.Is(err, ErrDecryptFailed) {
		t.Fatalf("expected ErrDecryptFailed for short nonce, got %v", err)
	}
}

func TestCrypto_DecryptTamperedCiphertext(t *testing.T) {
	c, err := NewCryptoFromHex(validTestKey)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	ct, nonce, err := c.Encrypt([]byte("payload"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if len(ct) == 0 {
		t.Fatal("unexpected empty ciphertext")
	}
	// Flip a single bit; GCM tag verification must fail.
	ct[0] ^= 0xFF
	_, err = c.Decrypt(ct, nonce)
	if !errors.Is(err, ErrDecryptFailed) {
		t.Fatalf("expected ErrDecryptFailed for tampered ciphertext, got %v", err)
	}
}
