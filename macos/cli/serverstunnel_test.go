package cli

import (
	"errors"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestTunnelOfSaysWhereTheDaemonStands(t *testing.T) {
	const (
		plist  = "/Library/LaunchDaemons/com.github.zadykian.ws-tunnel.ws.plist"
		loaded = "launchctl print system/com.github.zadykian.ws-tunnel.ws"
		state  = "/var/run/ws-tunnel.ws.state"
	)
	tests := []struct {
		name  string
		setup func(m *effecttest.Machine)
		want  string
	}{
		{"no plist", func(*effecttest.Machine) {}, "absent"},
		{"not loaded", func(m *effecttest.Machine) {
			m.Files[plist] = []byte("plist")
			m.Results[loaded] = effecttest.Result{Err: errors.New("not loaded")}
		}, "stopped"},
		{"no state yet", func(m *effecttest.Machine) { m.Files[plist] = []byte("plist") }, "starting"},
		{"on the LAN", func(m *effecttest.Machine) {
			m.Files[plist] = []byte("plist")
			m.Files[state] = []byte("tier=lan\nendpoint=10.88.0.10:51820\ninterface=utun5\nsince=1\nroutes=\nleft_out=\n")
		}, "lan 10.88.0.10:51820"},
		{"no way", func(m *effecttest.Machine) {
			m.Files[plist] = []byte("plist")
			m.Files[state] = []byte("tier=none\nendpoint=\ninterface=utun5\nsince=1\nroutes=\nleft_out=\n")
		}, "none"},
		{"a state it cannot read", func(m *effecttest.Machine) {
			m.Files[plist] = []byte("plist")
			m.Files[state] = []byte("since=soon\n")
		}, "unreadable"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := effecttest.NewMachine()
			m.Results[loaded] = effecttest.Result{}
			tc.setup(m)
			if got := tunnelOf(t.Context(), m, "ws"); got != tc.want {
				t.Errorf("tunnelOf = %q, want %q", got, tc.want)
			}
		})
	}
}
