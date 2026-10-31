// Package uapi writes and reads wireguard-go's configuration protocol, the text of its IpcSet and
// IpcGet: key=value lines, with keys in hex. It is pure; the daemon and status carry the text.
package uapi

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/zadykian/ws/mac/internal/wgkey"
)

// SetPrivateKey is the text that gives the device its private key, in base64.
func SetPrivateKey(private string) (string, error) {
	h, err := wgkey.Hex(private)
	if err != nil {
		return "", err
	}
	return "private_key=" + h + "\n", nil
}

// Peer is ws, the tunnel's only peer.
type Peer struct {
	Key       string // base64
	Endpoint  netip.AddrPort
	Allowed   []netip.Prefix
	Keepalive int // seconds
}

// Add is the text that adds the peer, with its endpoint, its AllowedIPs and its keepalive. With a
// keepalive, wireguard-go handshakes at once with a new peer.
func (p Peer) Add() (string, error) {
	h, err := wgkey.Hex(p.Key)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("public_key=%s\nendpoint=%s\npersistent_keepalive_interval=%d\n%s",
		h, p.Endpoint, p.Keepalive, allowed(p.Allowed)), nil
}

// SetAllowed is the text that replaces the AllowedIPs of a peer the device has, and adds no peer.
func SetAllowed(key string, nets []netip.Prefix) (string, error) {
	h, err := wgkey.Hex(key)
	if err != nil {
		return "", err
	}
	return "public_key=" + h + "\nupdate_only=true\n" + allowed(nets), nil
}

// Remove is the text that removes the peer.
func Remove(key string) (string, error) {
	h, err := wgkey.Hex(key)
	if err != nil {
		return "", err
	}
	return "public_key=" + h + "\nremove=true\n", nil
}

func allowed(nets []netip.Prefix) string {
	var b strings.Builder
	b.WriteString("replace_allowed_ips=true\n")
	for _, n := range nets {
		fmt.Fprintf(&b, "allowed_ip=%s\n", n)
	}
	return b.String()
}
