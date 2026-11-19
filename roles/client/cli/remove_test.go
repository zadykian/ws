package cli

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/values"
)

// rmMachine takes the files of a directory that rm -rf removes, which effecttest's Run only records.
type rmMachine struct{ *effecttest.Machine }

func (m rmMachine) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	out, err := m.Machine.Run(ctx, name, args...)
	m.take(name, args, err)
	return out, err
}

func (m rmMachine) RunRoot(ctx context.Context, name string, args ...string) ([]byte, error) {
	out, err := m.Machine.RunRoot(ctx, name, args...)
	m.take(name, args, err)
	return out, err
}

func (m rmMachine) take(name string, args []string, err error) {
	if name == "rm" && err == nil && len(args) == 2 {
		for f := range m.Files {
			if strings.HasPrefix(f, args[1]+"/") {
				delete(m.Files, f)
			}
		}
		delete(m.Dirs, args[1])
	}
}

// removeFixture is ws and lab on a Mac, whose known_hosts holds both, with the commands that a removal
// of lab runs. The Mac has no fnox key file, as where init never ran.
func removeFixture() (enginecli.Env, *effecttest.Machine) {
	m := effecttest.NewMachine()
	keyed(m)
	m.Files["/etc/ws/servers/lab/values.yml"] = []byte(values.Text("WS_USER", "ada", "WS_SLOT", "1"))
	m.Dirs["/etc/ws/servers/lab"] = 0o755
	m.Files["/h/.ssh/known_hosts"] = []byte("ws ssh-ed25519 AAAA\nlab ssh-ed25519 BBBB\n")
	m.Dirs["/Applications/Zed.app"] = 0o755
	m.Files["/h/Library/Fonts/JetBrainsMonoNerdFontMono-Regular.ttf"] = []byte("f")
	m.Results["mktemp -d"] = effecttest.Result{Out: []byte("/tmp/x\n")}
	m.Results["rm -rf /tmp/x"] = effecttest.Result{}
	for _, file := range []string{"new", "whole"} {
		m.Results["ssh -T -G -F /tmp/x/"+file+" ws"] = effecttest.Result{}
	}
	m.Results[findLab] = effecttest.Result{Out: []byte("lab ssh-ed25519 BBBB\n")}
	for _, line := range []string{"rm -rf /etc/ws/servers/lab", "ssh-keygen -R lab"} {
		m.Results[line] = effecttest.Result{}
	}
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.Machine = rmMachine{m}
	return p, m
}

func TestRemoveDeletesTheServersFilesAndItsHostKey(t *testing.T) {
	p, m := removeFixture()
	status, stdout, stderr := runWs(p, "client", "macos", "remove", "lab")
	if !strings.HasPrefix(stdout, "removed lab\n") {
		t.Errorf("status = %d, stdout = %q, stderr = %q; want removed lab first", status, stdout, stderr)
	}
	for _, want := range []string{"root rm -rf /etc/ws/servers/lab", "run ssh-keygen -R lab"} {
		if !slices.Contains(m.Calls, want) {
			t.Errorf("calls %q lack %q", m.Calls, want)
		}
	}
	if slices.Contains(m.Calls, "root rm -rf /etc/ws/servers/ws") {
		t.Error("remove deleted ws's files too")
	}
}

// TestRemoveKeepsTheServerWhereTheHostCannotBeForgotten checks that a failed ssh-keygen -R stops
// remove before it deletes the server's files, so that a second remove finds the server whole.
func TestRemoveKeepsTheServerWhereTheHostCannotBeForgotten(t *testing.T) {
	p, m := removeFixture()
	m.Results["ssh-keygen -R lab"] = effecttest.Result{Err: errors.New("exit 1")}
	status, stdout, stderr := runWs(p, "client", "macos", "remove", "lab")
	if status == 0 || !strings.Contains(stderr, "cannot drop lab from /h/.ssh/known_hosts") {
		t.Errorf("status = %d, stderr = %q; want the host's failure", status, stderr)
	}
	if strings.Contains(stdout, "removed") {
		t.Errorf("stdout = %q; want no removal", stdout)
	}
	if slices.Contains(m.Calls, "root rm -rf /etc/ws/servers/lab") {
		t.Error("remove deleted the server's directory before the host went")
	}
	if _, ok := m.Files["/etc/ws/servers/lab/values.yml"]; !ok {
		t.Error("the server's file is gone")
	}
}

func TestRemoveLeavesAKnownHostsThatIsALink(t *testing.T) {
	p, m := removeFixture()
	m.Links["/h/.ssh/known_hosts"] = "/d/known_hosts"
	m.Files["/d/known_hosts"] = []byte("lab ssh-ed25519 BBBB\n")
	_, stdout, _ := runWs(p, "client", "macos", "remove", "lab")
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
	if _, stdout, _ := runWs(p, "client", "macos", "remove", "lab"); !strings.HasPrefix(stdout, "removed lab\n") {
		t.Errorf("stdout = %q; want removed lab", stdout)
	}
}

// TestRemoveRunsTheBootstrap checks that the bootstrap takes the rest of the server off, and that
// its failure says to run it again.
func TestRemoveRunsTheBootstrap(t *testing.T) {
	p, _ := removeFixture()
	status, _, stderr := runWs(p, "client", "macos", "remove", "lab")
	if want := "MISE_LOCKED=1 mise -C /opt/ws/roles/client bootstrap: "; status != 1 || !strings.Contains(stderr, want) || !strings.Contains(stderr, "run it again to take lab's daemon, routes and host off") {
		t.Errorf("status = %d, stderr = %q; want 1 and %q", status, stderr, want)
	}
	p, m := removeFixture()
	m.Results["env MISE_LOCKED=1 mise -C /opt/ws/roles/client bootstrap"] = effecttest.Result{}
	if status, _, stderr := runWs(p, "client", "macos", "remove", "lab"); status != 0 || !slices.Contains(m.Calls, "run env MISE_LOCKED=1 mise -C /opt/ws/roles/client bootstrap") {
		t.Errorf("status = %d, stderr = %q, calls %q; want 0, the bootstrap run", status, stderr, m.Calls)
	}
}

func TestRemoveRefusesWhereAServerThatStaysHasABadFile(t *testing.T) {
	p, m := removeFixture()
	m.Files["/etc/ws/servers/abc/values.yml"] = []byte(values.Text("WS_USER", "ada", "WS_SLOT", "99"))
	m.Dirs["/etc/ws/servers/abc"] = 0o755
	status, _, stderr := runWs(p, "client", "macos", "remove", "lab")
	if want := ": fix its file, or ws client macos remove abc"; status != 1 || !strings.Contains(stderr, "server abc: ") ||
		!strings.Contains(stderr, want) {
		t.Errorf("status = %d, stderr = %q; want 1 and %q", status, stderr, want)
	}
	if slices.Contains(m.Calls, "root rm -rf /etc/ws/servers/lab") {
		t.Error("remove deleted lab's files though abc's steps would skip a server")
	}
}

// TestRemoveTakesAServerThatClashesWithOneThatStays checks that a clash with the gone server's file does
// not stop its removal, which is the way out of the clash.
func TestRemoveTakesAServerThatClashesWithOneThatStays(t *testing.T) {
	p, m := removeFixture()
	m.Files["/etc/ws/servers/lab/values.yml"] = []byte(values.Text("WS_USER", "ada", "WS_SLOT", "0"))
	if _, stdout, stderr := runWs(p, "client", "macos", "remove", "lab"); !strings.HasPrefix(stdout, "removed lab\n") {
		t.Errorf("stdout = %q, stderr = %q; want removed lab", stdout, stderr)
	}
}

func TestRemoveNamesTheCableOfTheGoneServer(t *testing.T) {
	p, m := removeFixture()
	m.Files["/etc/ws/servers/lab/values.yml"] = []byte(values.Text(
		"WS_USER", "ada",
		"WS_SLOT", "1",
		"WS_TARGET", "lan",
		"WS_CABLE_PORT", "eth1",
		"WS_CABLE_SERVICE", "USB 10/100/1000 LAN",
	))
	_, stdout, _ := runWs(p, "client", "macos", "remove", "lab")
	if want := "By hand: the cable's service USB 10/100/1000 LAN keeps 10.77.0.2"; !strings.Contains(stdout, want) {
		t.Errorf("stdout = %q; want it to hold %q", stdout, want)
	}
}

func TestRemoveRefusesTheLastServerAndAWrongName(t *testing.T) {
	for name, tc := range map[string]struct {
		arg, want string
		last      bool
	}{
		"last server":  {"ws", "ws is the Mac's one server: with none, the bootstrap prunes no daemon, so remove leaves it", true},
		"unknown name": {"nope", "no server named nope in /etc/ws/servers, which holds lab and ws", false},
		"bad name":     {"Bad_Name", "Bad_Name is not a server's name: " + values.NameForm, false},
		"path":         {"../ws", "../ws is not a server's name: " + values.NameForm, false},
	} {
		t.Run(name, func(t *testing.T) {
			p, m := removeFixture()
			if tc.last {
				delete(m.Files, "/etc/ws/servers/lab/values.yml")
				delete(m.Dirs, "/etc/ws/servers/lab")
			}
			status, _, stderr := runWs(p, "client", "macos", "remove", tc.arg)
			if status != 1 || !strings.Contains(stderr, tc.want) || strings.Contains(stderr, "--server") {
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
	status, _, stderr := runWs(p, "client", "macos", "--server", "ws", "remove", "lab")
	if status != 1 || !strings.Contains(stderr, "remove takes the server's name alone") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}
