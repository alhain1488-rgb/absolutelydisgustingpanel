package protocols

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

// RealityKeypair is an X25519 keypair encoded the way xray's `xray x25519`
// command emits them (base64 raw-URL).
type RealityKeypair struct {
	PrivateKey string
	PublicKey  string
}

// GenerateRealityKeypair produces a fresh X25519 keypair for REALITY.
func GenerateRealityKeypair() (RealityKeypair, error) {
	var priv [32]byte
	if _, err := rand.Read(priv[:]); err != nil {
		return RealityKeypair{}, err
	}
	// Clamp per X25519.
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64

	pub, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return RealityKeypair{}, err
	}
	enc := base64.RawURLEncoding
	return RealityKeypair{
		PrivateKey: enc.EncodeToString(priv[:]),
		PublicKey:  enc.EncodeToString(pub),
	}, nil
}

// GenerateShortID returns a random REALITY shortId (n bytes hex, n in 1..8).
func GenerateShortID(n int) (string, error) {
	if n <= 0 || n > 8 {
		n = 8
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GenerateShadowsocksPassword returns a base64 password sized for the cipher.
// For classic ciphers any string works; for aes-256-gcm we emit 32 bytes.
func GenerateShadowsocksPassword() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate ss password: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
