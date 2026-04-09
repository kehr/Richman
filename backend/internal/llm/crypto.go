package llm

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
)

// Master key and GCM nonce sizes for the per-user api_key encryption.
const (
	// MasterKeyBytes is the required length of the raw master key (AES-256).
	MasterKeyBytes = 32
	// NonceBytes is the standard GCM nonce length, 96 bits.
	NonceBytes = 12
)

// Crypto sentinel errors. Callers MUST check with errors.Is so we can keep
// the underlying details opaque and log-safe.
var (
	// ErrMasterKeyInvalid means the configured LLM_CONFIG_MASTER_KEY is not
	// a 64-char hex string decoding to exactly 32 bytes.
	ErrMasterKeyInvalid = errors.New("llm: master key must be 32 bytes (64 hex chars)")
	// ErrDecryptFailed is returned when GCM Open fails for any reason (wrong
	// nonce, tampered ciphertext, truncated input). The real error is never
	// wrapped so that callers cannot accidentally log secret material.
	ErrDecryptFailed = errors.New("llm: decrypt failed")
)

// Crypto wraps an AES-256-GCM AEAD constructed once from the master key at
// process startup. Instances are safe for concurrent use: the underlying AEAD
// is stateless and every Encrypt call supplies its own random nonce.
type Crypto struct {
	aead cipher.AEAD
}

// NewCryptoFromHex parses a 64-character hex master key into a Crypto. The
// process MUST fail fast (log.Fatal) at boot if this returns an error; the
// application cannot persist or decrypt user API keys without a valid key.
func NewCryptoFromHex(masterKeyHex string) (*Crypto, error) {
	if len(masterKeyHex) != MasterKeyBytes*2 {
		return nil, ErrMasterKeyInvalid
	}
	key, err := hex.DecodeString(masterKeyHex)
	if err != nil {
		return nil, ErrMasterKeyInvalid
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		// aes.NewCipher only fails on unsupported key length. Since we just
		// validated len(key) == 32 above, this branch is defensive: surface
		// as ErrMasterKeyInvalid to keep the public contract narrow.
		return nil, ErrMasterKeyInvalid
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ErrMasterKeyInvalid
	}
	return &Crypto{aead: aead}, nil
}

// Encrypt seals plaintext under a fresh random nonce and returns the
// ciphertext, the nonce, and any error. The nonce MUST be stored alongside
// the ciphertext so that a future Decrypt call can verify and open the
// payload. The plaintext slice is not modified and may be zeroized by the
// caller after return.
func (c *Crypto) Encrypt(plaintext []byte) (ciphertext, nonce []byte, err error) {
	nonce = make([]byte, NonceBytes)
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	ciphertext = c.aead.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// Decrypt opens ciphertext using the stored nonce. Any GCM verification
// failure (wrong nonce, truncated ciphertext, bit-flip) is collapsed into
// ErrDecryptFailed so callers cannot leak cipher-specific error details.
func (c *Crypto) Decrypt(ciphertext, nonce []byte) ([]byte, error) {
	if len(nonce) != NonceBytes {
		return nil, ErrDecryptFailed
	}
	pt, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptFailed
	}
	return pt, nil
}
