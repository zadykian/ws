package host

import (
	"context"
	"github.com/zadykian/ws/engine/core/remote"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestMacWaysFromAddresses(t *testing.T) {
	const lan = "WS_LAN=192.168.1.0/24\nWS_LAN_ADDRESS=192.168.1.5\n"
	for name, tc := range map[string]struct{ conf, ifconfig, want string }{
		"lan": {lan, "en0: flags=8863<UP> mtu 1500\n\tinet 192.168.1.20 netmask 0xffffff00\n",
			"-o HostName=192.168.1.5 -o Port=22 "},
		"cable": {lan, "en5: flags=8863<UP> mtu 1500\n\tinet 10.77.0.2 netmask 0xfffffffc\n",
			"-o HostName=10.77.0.1 -o Port=22 "},
		"remote": {wayConf, "", "-o HostName=ws.example.org -o Port=22022 "},
		"remote over a lan out of reach": {lan + wayConf, "en0: flags=8863<UP> mtu 1500\n\tinet 172.16.0.2 netmask 0xffff0000\n",
			"-o HostName=ws.example.org -o Port=22022 "},
	} {
		m := effecttest.NewMachine()
		m.Files["/c"] = []byte("WS_TARGET=lan\n" + tc.conf)
		m.Results["ifconfig"] = effecttest.Result{Out: []byte(tc.ifconfig)}
		got := strings.Join(remote.SSH{M: m, Lock: "/t/l", Ways: MacWays{M: m, Conf: "/c"}}.Argv(context.Background(), []string{"ws"}), " ")
		if !strings.Contains(got, tc.want) {
			t.Errorf("%s: argv = %q; want %q in it", name, got, tc.want)
		}
	}
}

// A mac.conf that is missing or refused says so, in place of ssh's want of a way.
func TestMacWaysSayWhyMacConfLeavesNone(t *testing.T) {
	m := effecttest.NewMachine()
	s := remote.SSH{M: m, Lock: "/l", Home: "/h", Ways: MacWays{M: m, Conf: "/h/mac.conf"}}
	if _, err := s.Way(context.Background()); err == nil || !strings.Contains(err.Error(), "no server.conf at /h/mac.conf") {
		t.Errorf("Way with no mac.conf = %v", err)
	}
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=moon\n")
	if _, err := s.Way(context.Background()); err == nil || !strings.Contains(err.Error(), "/h/mac.conf, line 1") {
		t.Errorf("Way with a refused mac.conf = %v", err)
	}
	m.Root["/h/mac.conf"] = effect.Attrs{}
	if _, err := s.Way(context.Background()); err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("Way with an unreadable mac.conf = %v", err)
	}
}
