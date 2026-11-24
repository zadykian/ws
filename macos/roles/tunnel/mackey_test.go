package tunnel_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/internal/wgkey"
	"github.com/zadykian/ws/macos/roles/tunnel"
)

func TestKeyWritesThePublicKey(t *testing.T) {
	m := effecttest.NewMachine()
	private, err := wgkey.Generate(bytes.NewReader(bytes.Repeat([]byte{7}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	m.Files["/etc/ws/tunnel/private.key"] = []byte(private + "\n")
	m.Modes["/etc/ws/tunnel/private.key"] = 0o644
	public, err := tunnel.MakeKey(context.Background(), m, bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	if want, _ := wgkey.Public(private); public != want || string(m.Files["/etc/ws/tunnel.pub"]) != want+"\n" {
		t.Errorf("public = %q, tunnel.pub = %q; want %q in both", public, m.Files["/etc/ws/tunnel.pub"], want)
	}
	if m.Modes["/etc/ws/tunnel.pub"] != 0o644 || m.Modes["/etc/ws/tunnel/private.key"] != 0o600 {
		t.Errorf("modes = %v, want tunnel.pub 644 and the key 600", m.Modes)
	}
	if m.Dirs["/etc/ws/tunnel"] != 0o755 {
		t.Errorf("/etc/ws/tunnel's mode = %o, want 755, so that the unit's check finds the key with no root", m.Dirs["/etc/ws/tunnel"])
	}
}
