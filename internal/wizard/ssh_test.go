package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestSSHArgv(t *testing.T) {
	for name, tc := range map[string]struct {
		lock, conf, first, want string
		notWant                 string
	}{
		"shared":  {"/t/l", "", "", "ControlPath=/t/l/direct ", "HostName"},
		"first":   {"/t/l", "", "10.0.0.5", "ControlPath=/t/l/direct.1 -o ControlPersist=30m", ""},
		"no room": {"/t/%l", "", "", "ConnectTimeout=10", "ControlMaster"},
		"a way": {"/t/l", "WS_REMOTE=ws.example.org\nWS_REMOTE_PORT=22022\n", "",
			"-o HostName=ws.example.org -o Port=22022 -o AddressFamily=inet ws-ssh", ""},
		"first over a way": {"/t/l", "WS_REMOTE=ws.example.org\nWS_REMOTE_PORT=22022\n", "10.0.0.5",
			"-o HostName=10.0.0.5 -o Port=22022 -o AddressFamily=inet ws-ssh", ""},
	} {
		m := effecttest.NewMachine()
		m.Files["/c"] = []byte("WS_TARGET=lan\n" + tc.conf)
		if tc.first != "" {
			m.Files[tc.lock+"/"+FirstFile] = []byte(tc.first + "\n")
		}
		got := strings.Join(SSH{M: m, Lock: tc.lock, Conf: "/c"}.argv(context.Background(), []string{"ws-ssh"}), " ") + " "
		if !strings.Contains(got, tc.want) || (tc.notWant != "" && strings.Contains(got, tc.notWant)) {
			t.Errorf("%s: argv = %q; want %q, not %q", name, got, tc.want, tc.notWant)
		}
	}
}

// wayCall is the ssh -G call that Way makes.
func wayCall(s SSH) string {
	return "ssh " + strings.Join(append([]string{"-G"}, s.argv(context.Background(), []string{"ws-ssh"})...), " ")
}

// tunnelCall is the call that Tunnel makes.
func tunnelCall(s SSH, args ...string) string {
	return "ssh " + strings.Join(append(s.shared("%n"), args...), " ")
}

func TestSSHWayFromAddresses(t *testing.T) {
	const lan = "WS_LAN=192.168.1.0/24\nWS_LAN_ADDRESS=192.168.1.5\n"
	const remote = "WS_REMOTE=ws.example.org\nWS_REMOTE_PORT=22022\n"
	for name, tc := range map[string]struct{ conf, ifconfig, host, port string }{
		"lan": {lan, "en0: flags=8863<UP> mtu 1500\n\tinet 192.168.1.20 netmask 0xffffff00\n",
			"192.168.1.5", "22"},
		"cable": {lan, "en5: flags=8863<UP> mtu 1500\n\tinet 10.77.0.2 netmask 0xfffffffc\n",
			"10.77.0.1", "22"},
		"remote over a lan out of reach": {lan + remote, "en0: flags=8863<UP> mtu 1500\n\tinet 172.16.0.2 netmask 0xffff0000\n",
			"ws.example.org", "22022"},
	} {
		m := effecttest.NewMachine()
		m.Files["/c"] = []byte("WS_TARGET=lan\n" + tc.conf)
		m.Results["ifconfig"] = effecttest.Result{Out: []byte(tc.ifconfig)}
		d, found := SSH{M: m, Lock: "/t/l", Conf: "/c"}.way(context.Background())
		if !found || d.Host != tc.host || d.Port != tc.port {
			t.Errorf("%s: way = %+v, %v; want %s:%s", name, d, found, tc.host, tc.port)
		}
	}
}
