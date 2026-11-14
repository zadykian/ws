package cli

import (
	"context"
	"slices"
	"strings"
	"testing"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// rmMachine takes the files of a directory that rm -rf removes, which effecttest's Run only records.
type rmMachine struct{ *effecttest.Machine }

func (m rmMachine) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	out, err := m.Machine.Run(ctx, name, args...)
	if name == "rm" && err == nil && len(args) == 2 {
		for f := range m.Files {
			if strings.HasPrefix(f, args[1]+"/") {
				delete(m.Files, f)
			}
		}
		delete(m.Dirs, args[1])
	}
	return out, err
}

// removeFixture is ws and lab on a Mac, whose known_hosts holds both, with the commands that a removal
// of lab runs.
func removeFixture() (enginecli.Env, *effecttest.Machine) {
	m := effecttest.NewMachine()
	keyed(m)
	m.Files["/h/.config/ws/servers/lab/server.conf"] = []byte("WS_USER=ada\nWS_SLOT=1\n")
	m.Dirs["/h/.config/ws/servers/lab"] = 0o755
	m.Files["/h/.ssh/known_hosts"] = []byte("ws ssh-ed25519 AAAA\nlab ssh-ed25519 BBBB\n")
	m.Dirs["/Applications/Zed.app"] = 0o755
	m.Results["mv -f /h/.local/bin/ws.new /h/.local/bin/ws"] = effecttest.Result{}
	m.Results["/usr/local/libexec/ws-tunnel/ws client macos tunnel key"] = effecttest.Result{Out: []byte("PUB\n")}
	m.Results["ls -1 /etc/ws-tunnel"] = effecttest.Result{}
	m.Files["/h/.local/bin/ws"] = []byte("ws")
	m.Files["/h/Library/Fonts/JetBrainsMonoNerdFontMono-Regular.ttf"] = []byte("f")
	m.Results["mktemp -d"] = effecttest.Result{Out: []byte("/tmp/x\n")}
	m.Results["rm -rf /tmp/x"] = effecttest.Result{}
	for _, file := range []string{"new", "whole"} {
		m.Results["ssh -G -F /tmp/x/"+file+" ws"] = effecttest.Result{}
	}
	for _, line := range []string{"rm -rf /h/.config/ws/servers/lab", "rm -rf /tmp/ws-wizard.lock", "ssh-keygen -R lab"} {
		m.Results[line] = effecttest.Result{}
	}
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.Machine = rmMachine{m}
	return p, m
}

func TestRemoveDeletesTheServersFilesAndItsHostKey(t *testing.T) {
	p, m := removeFixture()
	status, stdout, stderr := runWs(p, "", "client", "macos", "remove", "lab")
	if !strings.HasPrefix(stdout, "removed lab\n") {
		t.Errorf("status = %d, stdout = %q, stderr = %q; want removed lab first", status, stdout, stderr)
	}
	for _, want := range []string{"run rm -rf /h/.config/ws/servers/lab", "run ssh-keygen -R lab"} {
		if !slices.Contains(m.Calls, want) {
			t.Errorf("calls %q lack %q", m.Calls, want)
		}
	}
	if slices.Contains(m.Calls, "run rm -rf /h/.config/ws/servers/ws") {
		t.Error("remove deleted ws's files too")
	}
}

// TestRemoveRunsTheStepsForTheServerThatStays checks that ssh, Zed and the tunnel run, in that order, and
// stop at the tunnel's copy of ws, which the fake machine cannot hold.
func TestRemoveLeavesAKnownHostsThatIsALink(t *testing.T) {
	p, m := removeFixture()
	m.Links["/h/.ssh/known_hosts"] = "/d/known_hosts"
	m.Files["/d/known_hosts"] = []byte("lab ssh-ed25519 BBBB\n")
	_, stdout, _ := runWs(p, "", "client", "macos", "remove", "lab")
	if want := "By hand: /h/.ssh/known_hosts is a link to /d/known_hosts: delete the lines of lab there\nremoved lab\n"; !strings.HasPrefix(stdout, want) {
		t.Errorf("stdout = %q, want it to start with %q", stdout, want)
	}
	if slices.Contains(m.Calls, "run ssh-keygen -R lab") {
		t.Error("remove ran ssh-keygen over a link")
	}
}

func TestRemoveWithNoKnownHostsHasNoKeyToDrop(t *testing.T) {
	p, m := removeFixture()
	delete(m.Files, "/h/.ssh/known_hosts")
	if _, stdout, _ := runWs(p, "", "client", "macos", "remove", "lab"); !strings.HasPrefix(stdout, "removed lab\n") {
		t.Errorf("stdout = %q; want removed lab", stdout)
	}
}

// TestRemoveRunsTheStepsForTheServerThatStays checks that ssh, Zed and the tunnel run, in that order, and
// stop at the tunnel's copy of ws, which the fake machine cannot hold.
func TestRemoveRunsTheStepsForTheServerThatStays(t *testing.T) {
	p, _ := removeFixture()
	_, _, stderr := runWs(p, "", "client", "macos", "remove", "lab")
	last := -1
	for _, want := range []string{"wrote /h/.ssh/config.d/ws", "wrote /h/.config/zed/settings.json", "made the Mac's WireGuard key"} {
		at := strings.Index(stderr, want)
		if at <= last {
			t.Errorf("stderr = %q; want %q after the step before it", stderr, want)
		}
		last = at
	}
	if want := "then ws client macos apply --server ws takes lab's daemon, hosts and routes off"; !strings.Contains(stderr, want) {
		t.Errorf("stderr = %q; want it to name apply: %q", stderr, want)
	}
}

func TestRemoveRefusesWhereAServerThatStaysHasABadFile(t *testing.T) {
	p, m := removeFixture()
	m.Files["/h/.config/ws/servers/abc/server.conf"] = []byte("WS_USER=ada\nWS_SLOT=99\n")
	m.Dirs["/h/.config/ws/servers/abc"] = 0o755
	status, _, stderr := runWs(p, "", "client", "macos", "remove", "lab")
	if want := ": fix its file, or ws client macos remove abc"; status != 1 || !strings.Contains(stderr, "server abc: ") ||
		!strings.Contains(stderr, want) {
		t.Errorf("status = %d, stderr = %q; want 1 and %q", status, stderr, want)
	}
	if slices.Contains(m.Calls, "run rm -rf /h/.config/ws/servers/lab") {
		t.Error("remove deleted lab's files though abc's steps would skip a server")
	}
}

// TestRemoveTakesAServerThatClashesWithOneThatStays checks that a clash with the gone server's file does
// not stop its removal, which is the way out of the clash.
func TestRemoveTakesAServerThatClashesWithOneThatStays(t *testing.T) {
	p, m := removeFixture()
	m.Files["/h/.config/ws/servers/lab/server.conf"] = []byte("WS_USER=ada\nWS_SLOT=0\n")
	if _, stdout, stderr := runWs(p, "", "client", "macos", "remove", "lab"); !strings.HasPrefix(stdout, "removed lab\n") {
		t.Errorf("stdout = %q, stderr = %q; want removed lab", stdout, stderr)
	}
}

func TestRemoveNamesTheCableOfTheGoneServer(t *testing.T) {
	p, m := removeFixture()
	m.Files["/h/.config/ws/servers/lab/server.conf"] = []byte("WS_USER=ada\nWS_SLOT=1\nWS_TARGET=lan\nWS_CABLE_PORT=eth1\nWS_CABLE_SERVICE=\"USB 10/100/1000 LAN\"\n")
	_, stdout, _ := runWs(p, "", "client", "macos", "remove", "lab")
	if want := "By hand: the cable's service USB 10/100/1000 LAN keeps 10.77.0.2"; !strings.Contains(stdout, want) {
		t.Errorf("stdout = %q; want it to hold %q", stdout, want)
	}
}

func TestRemoveRefusesTheLastServerAndAWrongName(t *testing.T) {
	for name, tc := range map[string]struct {
		arg, want string
		last      bool
	}{
		"last server":  {"ws", "ws is the Mac's one server: apply needs one, so remove leaves it", true},
		"unknown name": {"nope", "no server named nope", false},
		"bad name":     {"Bad_Name", "is not a server's name", false},
	} {
		t.Run(name, func(t *testing.T) {
			p, m := removeFixture()
			if tc.last {
				delete(m.Files, "/h/.config/ws/servers/lab/server.conf")
				delete(m.Dirs, "/h/.config/ws/servers/lab")
			}
			status, _, stderr := runWs(p, "", "client", "macos", "remove", tc.arg)
			if status != 1 || !strings.Contains(stderr, tc.want) {
				t.Errorf("status = %d, stderr = %q; want 1 and %q", status, stderr, tc.want)
			}
			for _, call := range m.Calls {
				if strings.HasPrefix(call, "run rm") {
					t.Errorf("a refused remove ran %q", call)
				}
			}
		})
	}
}

func TestRemoveRefusesAServerFlag(t *testing.T) {
	p, _ := removeFixture()
	status, _, stderr := runWs(p, "", "client", "macos", "--server", "ws", "remove", "lab")
	if status != 1 || !strings.Contains(stderr, "remove takes the server's name alone") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}
