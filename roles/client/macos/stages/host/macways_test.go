package host

import (
	"context"
	"github.com/zadykian/ws/engine/core/remote"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/values"
)

func TestMacWaysFromAddresses(t *testing.T) {
	var lan = values.Text("WS_LAN", "192.168.1.0/24", "WS_LAN_ADDRESS", "192.168.1.5")
	for name, tc := range map[string]struct{ conf, ifconfig, want string }{
		"lan": {lan, "en0: flags=8863<UP> mtu 1500\n\tinet 192.168.1.20 netmask 0xffffff00\n",
			"-o HostName=192.168.1.5 -o Port=22 "},
		"cable": {lan + values.Text("WS_CABLE_SERVICE", "USB"), "en5: flags=8863<UP> mtu 1500\n\tinet 10.77.0.2 netmask 0xfffffffc\n",
			"-o HostName=10.77.0.1 -o Port=22 "},
		"remote": {wayConf, "", "-o HostName=ws.example.org -o Port=22022 "},
		"remote over a lan out of reach": {lan + wayConf, "en0: flags=8863<UP> mtu 1500\n\tinet 172.16.0.2 netmask 0xffff0000\n",
			"-o HostName=ws.example.org -o Port=22022 "},
	} {
		m := effecttest.NewMachine()
		m.Files["/c"] = []byte(values.Text("WS_TARGET", "lan") + tc.conf)
		m.Results["ifconfig"] = effecttest.Result{Out: []byte(tc.ifconfig)}
		got := strings.Join(remote.SSH{Host: "ws", M: m, Lock: "/t/l", Ways: MacWays{M: m, Conf: "/c"}}.Argv(context.Background(), []string{"ws"}), " ")
		if !strings.Contains(got, tc.want) {
			t.Errorf("%s: argv = %q; want %q in it", name, got, tc.want)
		}
	}
}

// A values.yml that is missing or refused says so, in place of ssh's want of a way.
func TestMacWaysSayWhyMacConfLeavesNone(t *testing.T) {
	m := effecttest.NewMachine()
	s := remote.SSH{Host: "ws", M: m, Lock: "/l", Home: "/h", Ways: MacWays{M: m, Conf: "/h/values.yml"}}
	if _, err := s.Way(context.Background()); err == nil || !strings.Contains(err.Error(), "no values.yml at /h/values.yml") {
		t.Errorf("Way with no values.yml = %v", err)
	}
	m.Files["/h/values.yml"] = []byte(values.Text("WS_TARGET", "moon"))
	if _, err := s.Way(context.Background()); err == nil || !strings.Contains(err.Error(), "/h/values.yml, line 1") {
		t.Errorf("Way with a refused values.yml = %v", err)
	}
	m.Root["/h/values.yml"] = effect.Attrs{}
	if _, err := s.Way(context.Background()); err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("Way with an unreadable values.yml = %v", err)
	}
}
