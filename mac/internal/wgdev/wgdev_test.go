package wgdev

import (
	"io"
	"net/netip"
	"testing"

	"golang.zx2c4.com/wireguard/tun/tuntest"

	"github.com/zadykian/ws/mac/internal/tunnel"
	"github.com/zadykian/ws/mac/internal/uapi"
)

// The keys are 32 equal bytes in base64.
const (
	macKey  = "AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI="
	peerKey = "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="
)

func TestNewTakesUTunNamesAlone(t *testing.T) {
	for name, ok := range map[string]bool{"utun0": true, "utun12": true, "utun": false, "en0": false, "utun1234": false, "utun1x": false} {
		tn, err := New(tuntest.NewChannelTUN().TUN(), name, io.Discard)
		if (err == nil) != ok {
			t.Errorf("New(%q) err = %v, want ok = %v", name, err, ok)
		}
		if tn != nil {
			tn.Close()
		}
	}
}

func TestDeviceKeepsThePeerThatIpcSetGave(t *testing.T) {
	tn, err := New(tuntest.NewChannelTUN().TUN(), "utun7", io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	defer tn.Close()
	key, _ := uapi.SetPrivateKey(macKey)
	peer, _ := uapi.Peer{
		Key: peerKey, Endpoint: netip.MustParseAddrPort("10.77.0.1:51820"),
		Allowed: []netip.Prefix{netip.PrefixFrom(tunnel.Server, 32)}, Keepalive: 25,
	}.Add()
	for _, text := range []string{key, peer} {
		if err := tn.IpcSet(text); err != nil {
			t.Fatal(err)
		}
	}
	text, err := tn.IpcGet()
	if err != nil {
		t.Fatal(err)
	}
	info, err := uapi.ParseGet(text, peerKey)
	if err != nil || info.Endpoint.String() != "10.77.0.1:51820" {
		t.Errorf("endpoint = %v, err = %v, from %q", info.Endpoint, err, text)
	}
	if tn.Name() != "utun7" {
		t.Errorf("name = %q", tn.Name())
	}
}
