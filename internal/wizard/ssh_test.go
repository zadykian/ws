package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestSSHArgv(t *testing.T) {
	const shared = "-o ConnectTimeout=10 -o StrictHostKeyChecking=yes -o UpdateHostKeys=no -o CheckHostIP=no " +
		"-o ServerAliveInterval=15 -o PermitLocalCommand=no"
	for name, tc := range map[string]struct {
		lock, conf, first, want string
	}{
		"a way": {"/t/l", wayConf, "", "-o ControlMaster=auto -o ControlPath=/t/l/direct -o ControlPersist=30m " +
			shared + " -o HostName=ws.example.org -o Port=22022 -o AddressFamily=inet ws"},
		"first with a way": {"/t/l", wayConf, "10.0.0.5", "-o ControlMaster=auto -o ControlPath=/t/l/direct.1 " +
			"-o ControlPersist=30m " + shared + " -o HostName=10.0.0.5 -o Port=22022 -o AddressFamily=inet ws"},
		"first with no way": {"/t/l", "", "10.0.0.5", "-o ControlMaster=auto -o ControlPath=/t/l/direct.1 " +
			"-o ControlPersist=30m " + shared + " -o HostName=10.0.0.5 -o AddressFamily=inet ws"},
		"neither": {"/t/l", "", "", "-o ControlMaster=auto -o ControlPath=/t/l/direct -o ControlPersist=30m " +
			shared + " -o AddressFamily=inet ws"},
		"no room": {"/t/%l", wayConf, "", shared + " -o HostName=ws.example.org -o Port=22022 " +
			"-o AddressFamily=inet ws"},
	} {
		m := effecttest.NewMachine()
		m.Files["/c"] = []byte("WS_TARGET=lan\n" + tc.conf)
		if tc.first != "" {
			m.Files[tc.lock+"/"+FirstFile] = []byte(tc.first + "\n")
		}
		got := strings.Join(SSH{M: m, Lock: tc.lock, Ways: MacWays{M: m, Conf: "/c"}}.Argv(context.Background(), []string{"ws"}), " ")
		if got != tc.want {
			t.Errorf("%s: argv = %q; want %q", name, got, tc.want)
		}
	}
}

func TestSSHTunnelTakesNoWayOptions(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/c"] = []byte("WS_TARGET=lan\n" + wayConf)
	m.Files["/t/l/"+FirstFile] = []byte("10.0.0.5\n")
	s := SSH{M: m, Lock: "/t/l", Ways: MacWays{M: m, Conf: "/c"}}
	m.Results[tunnelCall(s, "ws", "true")] = effecttest.Result{}
	if _, err := s.Tunnel(context.Background(), "ws", "true"); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(m.Calls, "\n")
	for _, no := range []string{"HostName", "Port", "AddressFamily", "direct"} {
		if strings.Contains(got, no) {
			t.Errorf("Tunnel call %q holds %s", got, no)
		}
	}
	if !strings.Contains(got, "ControlPath=/t/l/%n ") {
		t.Errorf("Tunnel call %q lacks the socket /t/l/%%n", got)
	}
}

// wayCall is the ssh -G call that Way makes.
func wayCall(s SSH) string {
	return "ssh " + strings.Join(append([]string{"-G"}, s.Argv(context.Background(), []string{"ws"})...), " ")
}

// tunnelCall is the call that Tunnel makes.
func tunnelCall(s SSH, args ...string) string {
	return "ssh " + strings.Join(append(s.Shared("%n"), args...), " ")
}

func TestSSHWayFromAddresses(t *testing.T) {
	const lan = "WS_LAN=192.168.1.0/24\nWS_LAN_ADDRESS=192.168.1.5\n"
	for name, tc := range map[string]struct{ conf, ifconfig, want string }{
		"lan": {lan, "en0: flags=8863<UP> mtu 1500\n\tinet 192.168.1.20 netmask 0xffffff00\n",
			"-o HostName=192.168.1.5 -o Port=22 "},
		"cable": {lan, "en5: flags=8863<UP> mtu 1500\n\tinet 10.77.0.2 netmask 0xfffffffc\n",
			"-o HostName=10.77.0.1 -o Port=22 "},
		"remote over a lan out of reach": {lan + wayConf, "en0: flags=8863<UP> mtu 1500\n\tinet 172.16.0.2 netmask 0xffff0000\n",
			"-o HostName=ws.example.org -o Port=22022 "},
	} {
		m := effecttest.NewMachine()
		m.Files["/c"] = []byte("WS_TARGET=lan\n" + tc.conf)
		m.Results["ifconfig"] = effecttest.Result{Out: []byte(tc.ifconfig)}
		got := strings.Join(SSH{M: m, Lock: "/t/l", Ways: MacWays{M: m, Conf: "/c"}}.Argv(context.Background(), []string{"ws"}), " ")
		if !strings.Contains(got, tc.want) {
			t.Errorf("%s: argv = %q; want %q in it", name, got, tc.want)
		}
	}
}
