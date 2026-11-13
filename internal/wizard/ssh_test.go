package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// fixedWays gives a way, or the reason that none is given.
type fixedWays struct {
	host, port string
	missing    error
}

func (f fixedWays) Direct(context.Context) (string, string, bool) {
	return f.host, f.port, f.host != ""
}

func (f fixedWays) Missing() error { return f.missing }

func TestSSHArgv(t *testing.T) {
	const shared = "-o ConnectTimeout=10 -o StrictHostKeyChecking=yes -o UpdateHostKeys=no -o CheckHostIP=no " +
		"-o ServerAliveInterval=15 -o PermitLocalCommand=no"
	way := fixedWays{host: "ws.example.org", port: "22022"}
	for name, tc := range map[string]struct {
		lock        string
		ways        Ways
		first, want string
	}{
		"a way": {"/t/l", way, "", "-o ControlMaster=auto -o ControlPath=/t/l/direct -o ControlPersist=30m " +
			shared + " -o HostName=ws.example.org -o Port=22022 -o AddressFamily=inet ws"},
		"first with a way": {"/t/l", way, "10.0.0.5", "-o ControlMaster=auto -o ControlPath=/t/l/direct.1 " +
			"-o ControlPersist=30m " + shared + " -o HostName=10.0.0.5 -o Port=22022 -o AddressFamily=inet ws"},
		"first with no way": {"/t/l", fixedWays{}, "10.0.0.5", "-o ControlMaster=auto -o ControlPath=/t/l/direct.1 " +
			"-o ControlPersist=30m " + shared + " -o HostName=10.0.0.5 -o AddressFamily=inet ws"},
		"neither": {"/t/l", nil, "", "-o ControlMaster=auto -o ControlPath=/t/l/direct -o ControlPersist=30m " +
			shared + " -o AddressFamily=inet ws"},
		"no room": {"/t/%l", way, "", shared + " -o HostName=ws.example.org -o Port=22022 " +
			"-o AddressFamily=inet ws"},
	} {
		m := effecttest.NewMachine()
		if tc.first != "" {
			m.Files[tc.lock+"/"+FirstFile] = []byte(tc.first + "\n")
		}
		got := strings.Join(SSH{M: m, Lock: tc.lock, Ways: tc.ways}.Argv(context.Background(), []string{"ws"}), " ")
		if got != tc.want {
			t.Errorf("%s: argv = %q; want %q", name, got, tc.want)
		}
	}
}

func TestSSHTunnelTakesNoWayOptions(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/t/l/"+FirstFile] = []byte("10.0.0.5\n")
	s := SSH{M: m, Lock: "/t/l", Ways: fixedWays{host: "ws.example.org", port: "22022"}}
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
