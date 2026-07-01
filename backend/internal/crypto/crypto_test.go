package crypto

import (
	"bytes"
	"testing"
)

func newTestCipher(t *testing.T) *Cipher {
	t.Helper()
	c, err := NewCipher(make([]byte, 32))
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	return c
}

func TestNewCipher_BadKey(t *testing.T) {
	if _, err := NewCipher(make([]byte, 16)); err == nil {
		t.Fatal("expected error for 16-byte key")
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	c := newTestCipher(t)
	for _, pt := range []string{"", "hello", "ssh-private-key-\x00\x01\x02", "totp-secret"} {
		enc, err := c.EncryptString(pt)
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		got, err := c.DecryptString(enc)
		if err != nil {
			t.Fatalf("decrypt: %v", err)
		}
		if got != pt {
			t.Fatalf("round-trip mismatch: got %q want %q", got, pt)
		}
	}
}

func TestEncrypt_NonDeterministic(t *testing.T) {
	c := newTestCipher(t)
	a, _ := c.EncryptString("same")
	b, _ := c.EncryptString("same")
	if a == b {
		t.Fatal("expected different ciphertexts due to random nonce")
	}
}

func TestDecrypt_Tampered(t *testing.T) {
	c := newTestCipher(t)
	enc, _ := c.Encrypt([]byte("secret"))
	// Corrupt one byte of the base64 payload.
	raw := []byte(enc)
	raw[len(raw)-2] ^= 0xFF
	if _, err := c.Decrypt(string(raw)); err == nil {
		t.Fatal("expected decrypt error for tampered ciphertext")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	c1 := newTestCipher(t)
	key2 := make([]byte, 32)
	key2[0] = 1
	c2, _ := NewCipher(key2)
	enc, _ := c1.EncryptString("secret")
	if _, err := c2.DecryptString(enc); err == nil {
		t.Fatal("expected decrypt failure with wrong key")
	}
}

func TestBcrypt(t *testing.T) {
	hash, err := HashPassword("hunter2")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !VerifyPassword(hash, "hunter2") {
		t.Fatal("verify should succeed for correct password")
	}
	if VerifyPassword(hash, "wrong") {
		t.Fatal("verify should fail for wrong password")
	}
	if bytes.Equal([]byte(hash), []byte("hunter2")) {
		t.Fatal("hash must not equal plaintext")
	}
}
