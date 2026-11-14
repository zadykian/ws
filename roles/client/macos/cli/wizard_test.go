package cli

import (
	"context"
	"errors"
	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/client/macos/stages/host"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"slices"
	"strings"
	"testing"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/remote"
)

// testWizardProcess returns the process of a Mac whose Command Line Tools and Homebrew are there, and
// its environment.
func testWizardProcess(goos string, uid int) (enginecli.Env, map[string]string) {
	env := map[string]string{"TMPDIR": "/t/"}
	m := effecttest.NewMachine()
	m.Results["xcode-select -p"] = effecttest.Result{Out: []byte("/clt\n")}
	m.Files["/clt/usr/bin/git"] = nil
	m.Modes["/clt/usr/bin/git"] = 0o755
	m.Results["/opt/homebrew/bin/brew --version"] = effecttest.Result{}
	p := withEnv(fakeProcess(), env)
	p.GOOS, p.Euid, p.Machine = goos, uid, m
	return p, env
}

func TestWizardCommandRefuses(t *testing.T) {
	for name, tc := range map[string]struct {
		goos    string
		uid     int
		message string
	}{
		"linux": {"linux", 501, "ws client macos wizard: it sets up a Mac, not Linux\n"},
		"root":  {"darwin", 0, "ws client macos wizard: run it as yourself, not root: it asks sudo where it needs root\n"},
	} {
		p, _ := testWizardProcess(tc.goos, tc.uid)
		if status, _, stderr := runWs(p, "", "client", "macos", "wizard"); status != 1 || stderr != tc.message {
			t.Errorf("%s: status = %d, stderr = %q; want 1, %q", name, status, stderr, tc.message)
		}
	}
}

func TestWizardCommandRuns(t *testing.T) {
	p, _ := testWizardProcess("darwin", 501)
	if status, _, stderr := runWs(p, "", "client", "macos", "wizard", "--check", "--server", "ws"); status != 1 ||
		!strings.Contains(stderr, "ws client macos wizard: Answers: no terminal to ask WS_TARGET: set WS_TARGET") {
		t.Errorf("status = %d, stderr = %q; want the refusal to ask with no terminal", status, stderr)
	}
	// The server is named first, by a read of the directory of servers alone.
	calls := slices.DeleteFunc(slices.Clone(p.Machine.(*effecttest.Machine).Calls), func(c string) bool { return strings.HasPrefix(c, "readdir ") })
	if len(calls) == 0 || calls[0] != "mkdir /t/ws-wizard.lock 700" {
		t.Errorf("calls = %v; want the lock first", calls)
	}
}

func TestWizardPutsHomebrewFirstAndSetsWsConf(t *testing.T) {
	p, env := testWizardProcess("darwin", 501)
	env["PATH"], env["HOME"] = "/usr/bin", "/h"
	runWs(p, "", "client", "macos", "wizard", "--check")
	if env["PATH"] != "/opt/homebrew/bin:/usr/bin" || env["WS_CONF"] != "/h/.config/ws/mac.conf" {
		t.Errorf("PATH = %q, WS_CONF = %q", env["PATH"], env["WS_CONF"])
	}
}

func TestWizardCheckShowsTheAnswersDiff(t *testing.T) {
	p, env := testWizardProcess("darwin", 501)
	for k, v := range map[string]string{"WS_TARGET": "cloud", "WS_REMOTE": "r.example", "WS_REMOTE_PORT": "22", "WS_GIT_NAME": "",
		"WS_TUNNEL_NETWORKS": "", "WS_RESOLVERS": ""} {
		env[k] = v
	}
	m := p.Machine.(*effecttest.Machine)
	m.Results["mkdir -p /t/ws-wizard.lock"] = effecttest.Result{}
	m.Results["mkdir -p /t/ws-wizard.lock/servers/ws"] = effecttest.Result{}
	status, _, stderr := runWs(p, "", "client", "macos", "wizard", "--check", "--server", "ws")
	if status != 0 || !strings.Contains(stderr, "+WS_TARGET=cloud") {
		t.Errorf("status = %d, stderr = %q; want the diff of the server's file", status, stderr)
	}
	if got := string(m.Files["/t/ws-wizard.lock/servers/ws/server.conf"]); !strings.HasPrefix(got, "WS_SLOT=0\nWS_PORT=51820\nWS_TARGET=cloud\n") {
		t.Errorf("scratch server file = %q", got)
	}
	if got := string(m.Files["/t/ws-wizard.lock/mac.conf"]); got != "WS_GIT_NAME=\n" {
		t.Errorf("scratch mac.conf = %q", got)
	}
}

func TestWizardCheckOnANewMacStopsAtHomebrew(t *testing.T) {
	p, _ := testWizardProcess("darwin", 501)
	m := p.Machine.(*effecttest.Machine)
	m.Results["xcode-select -p"] = effecttest.Result{Err: errors.New("exit status 2")}
	m.Results["/opt/homebrew/bin/brew --version"] = effecttest.Result{Err: errors.New("no brew")}
	status, _, stderr := runWs(p, "", "client", "macos", "wizard", "--check", "--server", "ws")
	if status != 0 {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	for _, want := range []string{"Tools: would run xcode-select --install", "Homebrew: after Tools", "Answers: after Homebrew",
		"gh on the Mac: after Homebrew", "ws client macos apply: after Homebrew"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr = %q, want %q", stderr, want)
		}
	}
}

func TestWizardUpdateRunsTheNewWs(t *testing.T) {
	p, env := testWizardProcess("darwin", 501)
	m := p.Machine.(*effecttest.Machine)
	p.Version, p.Releases, p.Argv = "v1.0.0", "http://127.0.0.1:22080", []string{"client", "macos", "wizard", "--update"}
	env["HOME"] = "/h"
	m.Results["curl -fsS -o /dev/null -w %{redirect_url} http://127.0.0.1:22080/latest"] =
		effecttest.Result{Out: []byte("http://127.0.0.1:22080/tag/v1.1.0")}
	m.Results["mktemp -d"] = effecttest.Result{Out: []byte("/x\n")}
	for _, c := range []string{
		"curl -fsSL --retry 3 --proto-redir =https -o /x/install.sh http://127.0.0.1:22080/download/v1.1.0/install.sh",
		"env WS_VERSION=v1.1.0 WS_RELEASES_URL=http://127.0.0.1:22080 sh /x/install.sh", "rm -rf /x"} {
		m.Results[c] = effecttest.Result{}
	}
	// The fake Exec returns, so the step finds its change again: the command fails, after the exec.
	status, _, _ := runWs(p, "", "client", "macos", "wizard", "--update")
	calls := strings.Join(m.Calls, "\n")
	if status != 1 || !strings.Contains(calls, "exec /h/.local/bin/ws client macos wizard --update") || strings.Contains(calls, "mkdir /t/ws-wizard.lock") {
		t.Errorf("status = %d, calls = %q; want the exec of the new ws, before the lock", status, calls)
	}
}

func TestWizardUpdateCheckInstallsNothing(t *testing.T) {
	p, _ := testWizardProcess("darwin", 501)
	m := p.Machine.(*effecttest.Machine)
	p.Version, p.Releases = "v1.0.0", "http://127.0.0.1:22080"
	m.Results["curl -fsS -o /dev/null -w %{redirect_url} http://127.0.0.1:22080/latest"] =
		effecttest.Result{Out: []byte("http://127.0.0.1:22080/tag/v1.1.0")}
	_, _, stderr := runWs(p, "", "client", "macos", "wizard", "--check", "--update")
	if !strings.Contains(stderr, "ws on the Mac: would install ws v1.1.0 on the Mac") || strings.Contains(strings.Join(m.Calls, "\n"), "exec") {
		t.Errorf("stderr = %q, calls = %v; want the plan and no exec", stderr, m.Calls)
	}
}

// TestWizardStagesStateTheirSides puts the lists of the two machines on firm ground: each stage
// that the wizard runs names the machine it acts on.
func TestWizardStagesStateTheirSides(t *testing.T) {
	var out strings.Builder
	stages := wizardStages(wizardEffects{p: fakeProcess(), prog: "ws client macos wizard", text: &out, hand: &wizard.Hand{},
		sudo: &host.Sudo{}}, engine.WizardOptions{}, "ws")
	onWs := 0
	for _, s := range stages {
		m, ok := wizard.SideOf(s)
		if !ok {
			t.Errorf("%s states no side", s.Name())
		}
		if m == session.Ws {
			onWs++
		}
	}
	if len(stages) != 15 || onWs != 4 {
		t.Errorf("%d stages, %d on ws; want 15 and 4", len(stages), onWs)
	}
}

// wsOverSSH answers the remote script's ops as a ws reached by ssh: state with its report, and any
// other with its status.
type wsOverSSH struct {
	*effecttest.Machine
	status, report []byte
}

// RunInput answers by the op, the last of ssh's arguments.
func (m wsOverSSH) RunInput(_ context.Context, _ []byte, _ string, args ...string) ([]byte, error) {
	if args[len(args)-1] == "state" {
		return m.report, nil
	}
	return m.status, nil
}

func TestWizardReadsWhatServerStatePrints(t *testing.T) {
	p, _ := serverTerminal()
	status, stdout, _ := runWs(p, "", "server", "ubuntu", "state", "--json", "--only", "tmux")
	if status != 0 {
		t.Fatalf("status = %d", status)
	}
	m := wsOverSSH{effecttest.NewMachine(), []byte("ws " + p.Version + "\n"), []byte(stdout)}
	ws := session.New(host.MacReach{SSH: remote.SSH{Host: "ws", M: m, Lock: "/t/l"}},
		session.Scripts{Remote: session.RemoteScript})
	roles, err := ws.State(context.Background(), p.Version)
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 2 || roles[1].Role != "tmux" || len(roles[1].Steps) == 0 {
		t.Errorf("roles = %+v", roles)
	}
}
