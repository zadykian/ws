// Package wgkey makes WireGuard keys and turns them into the forms that the tunnel needs. A key is
// 32 bytes, shown as base64 in files and as hex in wireguard-go's UAPI.
package wgkey

import (
	"crypto/ecdh"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

const size = 32

// Generate makes a private key from random, as wg genkey does: 32 random bytes with the bits
// cleared and set that Curve25519 needs, so that the key is the same in every program. It returns
// the key in base64.
func Generate(random io.Reader) (string, error) {
	var key [size]byte
	if _, err := io.ReadFull(random, key[:]); err != nil {
		return "", fmt.Errorf("cannot make a key: %w", err)
	}
	key[0] &= 248
	key[31] = key[31]&127 | 64
	return base64.StdEncoding.EncodeToString(key[:]), nil
}

// Public returns the public key, in base64, of a private key in base64.
func Public(private string) (string, error) {
	raw, err := Decode(private)
	if err != nil {
		return "", err
	}
	key, err := ecdh.X25519().NewPrivateKey(raw)
	if err != nil {
		return "", fmt.Errorf("not a private key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key.PublicKey().Bytes()), nil
}

// Decode returns the 32 bytes of a key in base64.
func Decode(key string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(key)
	if err != nil || len(raw) != size {
		return nil, fmt.Errorf("not a WireGuard key: want %d bytes in base64", size)
	}
	return raw, nil
}

// Hex returns a key in base64 as the UAPI writes it, in hex.
func Hex(key string) (string, error) {
	raw, err := Decode(key)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}
