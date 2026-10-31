package uapi

import (
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/mac/internal/tunnel"
)

var other = "public_key=" + "00" + keyHex + "\nendpoint=198.51.100.1:1\nrx_bytes=1\n"

func TestParseGet(t *testing.T) {
	device := "private_key=aa\nlisten_port=0\n"
	peer := "public_key=" + keyHex + "\nprotocol_version=1\nendpoint=192.0.2.7:51820\n" +
		"last_handshake_time_sec=1700000000\nlast_handshake_time_nsec=5\ntx_bytes=20\nrx_bytes=10\n" +
		"persistent_keepalive_interval=25\nallowed_ip=10.99.0.1/32\n"
	tests := []struct {
		name string
		text string
		want Info
	}{
		{"peer", device + other + peer, Info{
			tunnel.Stats{Handshake: time.Unix(1700000000, 5), Received: 10, Sent: 20, Known: true},
			netip.MustParseAddrPort("192.0.2.7:51820")}},
		{"no handshake", device + "public_key=" + keyHex + "\nlast_handshake_time_sec=0\n" +
			"last_handshake_time_nsec=0\n", Info{Stats: tunnel.Stats{Known: true}}},
		{"no peer", device + other, Info{}},
		{"empty", "", Info{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGet(tt.text, key)
			if err != nil || got != tt.want {
				t.Errorf("got %+v, %v; want %+v", got, err, tt.want)
			}
		})
	}
}

func TestParseGetErrors(t *testing.T) {
	head := "public_key=" + keyHex + "\n"
	for _, line := range []string{"endpoint=x", "rx_bytes=-1", "last_handshake_time_sec=a"} {
		if _, err := ParseGet(head+line+"\n", key); err == nil || !strings.Contains(err.Error(), "uapi:") {
			t.Errorf("%s: err = %v", line, err)
		}
	}
	if _, err := ParseGet(head, "x"); err == nil {
		t.Error("a bad key was taken")
	}
}
