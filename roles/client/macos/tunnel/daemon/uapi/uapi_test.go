package uapi

import (
	"bytes"
	"encoding/base64"
	"net/netip"
	"strings"
	"testing"
)

// key is 32 bytes of 0x2a in base64, and keyHex the same in hex.
var (
	keyHex = strings.Repeat("2a", 32)
	key    = base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x2a}, 32))
)

var nets = []netip.Prefix{netip.MustParsePrefix("10.99.0.1/32"), netip.MustParsePrefix("192.168.1.0/24")}

func TestText(t *testing.T) {
	endpoint := netip.MustParseAddrPort("192.0.2.7:51820")
	tests := []struct {
		name string
		got  func() (string, error)
		want string
	}{
		{"private key", func() (string, error) { return SetPrivateKey(key) }, "private_key=" + keyHex + "\n"},
		{"add", func() (string, error) { return Peer{key, endpoint, nets, 25}.Add() },
			"public_key=" + keyHex + "\nendpoint=192.0.2.7:51820\npersistent_keepalive_interval=25\n" +
				"replace_allowed_ips=true\nallowed_ip=10.99.0.1/32\nallowed_ip=192.168.1.0/24\n"},
		{"allowed", func() (string, error) { return SetAllowed(key, nets[:1]) },
			"public_key=" + keyHex + "\nupdate_only=true\nreplace_allowed_ips=true\nallowed_ip=10.99.0.1/32\n"},
		{"remove", func() (string, error) { return Remove(key) }, "public_key=" + keyHex + "\nremove=true\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.got()
			if err != nil || got != tt.want {
				t.Errorf("got %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestTextBadKey(t *testing.T) {
	for name, f := range map[string]func() (string, error){
		"private key": func() (string, error) { return SetPrivateKey("x") },
		"add":         func() (string, error) { return Peer{Key: "x"}.Add() },
		"allowed":     func() (string, error) { return SetAllowed("x", nil) },
		"remove":      func() (string, error) { return Remove("x") },
	} {
		if _, err := f(); err == nil || !strings.Contains(err.Error(), "WireGuard key") {
			t.Errorf("%s: err = %v, want the key's", name, err)
		}
	}
}
