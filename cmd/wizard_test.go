package cmd

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/roles/client/macos/mac"
)

// testWizardProcess returns the process of a Mac whose Command Line Tools and Homebrew are there, and
// its environment.
func testWizardProcess(goos string, uid int) (process, map[string]string) {
	env := map[string]string{"TMPDIR": "/t/"}
	m := effecttest.NewMachine()
	m.Results["xcode-select -p"] = effecttest.Result{Out: []byte("/clt\n")}
	m.Files["/clt/usr/bin/git"] = nil
	m.Modes["/clt/usr/bin/git"] = 0o755
	m.Results["/opt/homebrew/bin/brew --version"] = effecttest.Result{}
	p := withEnv(fakeProcess(), env)
	p.goos, p.euid, p.machine = goos, uid, m
	return p, env
}

func TestWizardCommandRefuses(t *testing.T) {
	for name, tc := range map[string]struct {
		goos    string
		uid     int
		message string
	}{
		"linux": {"linux", 501, "ws mac wizard: it sets up a Mac, not Linux\n"},
		"root":  {"darwin", 0, "ws mac wizard: run it as yourself, not root: it asks sudo where it needs root\n"},
	} {
		p, _ := testWizardProcess(tc.goos, tc.uid)
		if status, _, stderr := runWs(p, "", "mac", "wizard"); status != 1 || stderr != tc.message {
			t.Errorf("%s: status = %d, stderr = %q; want 1, %q", name, status, stderr, tc.message)
		}
	}
}

func TestWizardCommandRuns(t *testing.T) {
	p, _ := testWizardProcess("darwin", 501)
	if status, _, stderr := runWs(p, "", "mac", "wizard", "--check"); status != 1 ||
		!strings.Contains(stderr, "ws mac wizard: Answers: no terminal to ask WS_TARGET: set WS_TARGET") {
		t.Errorf("status = %d, stderr = %q; want the refusal to ask with no terminal", status, stderr)
	}
	if calls := p.machine.(*effecttest.Machine).Calls; len(calls) == 0 || calls[0] != "mkdir /t/ws-wizard.lock 700" {
		t.Errorf("calls = %v; want the lock first", calls)
	}
}

func TestWizardPutsHomebrewFirstAndSetsWsConf(t *testing.T) {
	p, env := testWizardProcess("darwin", 501)
	env["PATH"], env["HOME"] = "/usr/bin", "/h"
	runWs(p, "", "mac", "wizard", "--check")
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
	status, _, stderr := runWs(p, "", "mac", "wizard", "--check")
	if status != 0 || !strings.Contains(stderr, "+WS_TARGET=cloud") {
		t.Errorf("status = %d, stderr = %q; want the diff of mac.conf", status, stderr)
	}
	if got := string(p.machine.(*effecttest.Machine).Files["/t/ws-wizard.lock/mac.conf"]); !strings.HasPrefix(got, "WS_TARGET=cloud\n") {
		t.Errorf("scratch mac.conf = %q", got)
	}
}

func TestIsTerminalIsNotDevNull(t *testing.T) {
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = null.Close() }()
	if isTerminal(null) {
		t.Error("isTerminal(/dev/null) = true")
	}
}

func TestWizardCheckOnANewMacStopsAtHomebrew(t *testing.T) {
	p, _ := testWizardProcess("darwin", 501)
	m := p.machine.(*effecttest.Machine)
	m.Results["xcode-select -p"] = effecttest.Result{Err: errors.New("exit status 2")}
	m.Results["/opt/homebrew/bin/brew --version"] = effecttest.Result{Err: errors.New("no brew")}
	status, _, stderr := runWs(p, "", "mac", "wizard", "--check")
	if status != 0 {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	for _, want := range []string{"Tools: would run xcode-select --install", "Homebrew: after Tools", "Answers: after Homebrew",
		"gh on the Mac: after Homebrew", "ws mac apply: after Homebrew"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr = %q, want %q", stderr, want)
		}
	}
}

func TestWizardUpdateRunsTheNewWs(t *testing.T) {
	p, env := testWizardProcess("darwin", 501)
	m := p.machine.(*effecttest.Machine)
	p.version, p.releases, p.argv = "v1.0.0", "http://127.0.0.1:22080", []string{"mac", "wizard", "--update"}
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
	status, _, _ := runWs(p, "", "mac", "wizard", "--update")
	calls := strings.Join(m.Calls, "\n")
	if status != 1 || !strings.Contains(calls, "exec /h/.local/bin/ws mac wizard --update") || strings.Contains(calls, "mkdir /t/ws-wizard.lock") {
		t.Errorf("status = %d, calls = %q; want the exec of the new ws, before the lock", status, calls)
	}
}

func TestWizardUpdateCheckInstallsNothing(t *testing.T) {
	p, _ := testWizardProcess("darwin", 501)
	m := p.machine.(*effecttest.Machine)
	p.version, p.releases = "v1.0.0", "http://127.0.0.1:22080"
	m.Results["curl -fsS -o /dev/null -w %{redirect_url} http://127.0.0.1:22080/latest"] =
		effecttest.Result{Out: []byte("http://127.0.0.1:22080/tag/v1.1.0")}
	_, _, stderr := runWs(p, "", "mac", "wizard", "--check", "--update")
	if !strings.Contains(stderr, "ws on the Mac: would install ws v1.1.0 on the Mac") || strings.Contains(strings.Join(m.Calls, "\n"), "exec") {
		t.Errorf("stderr = %q, calls = %v; want the plan and no exec", stderr, m.Calls)
	}
}

// TestWizardStagesNameTheirMachines puts the lists of the two machines on firm ground: each stage
// that the wizard runs names the machine it acts on.
func TestWizardStagesNameTheirMachines(t *testing.T) {
	var out strings.Builder
	stages := wizardStages(wizardEffects{p: fakeProcess(), prog: "ws mac wizard", text: &out, hand: &wizard.Hand{},
		sudo: &wizard.Sudo{}}, wizardOptions{})
	onWs := 0
	for _, s := range stages {
		m, ok := wizard.MachineOf(s)
		if !ok {
			t.Errorf("%s names no machine", s.Name())
		}
		if m == wizard.OnWs {
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
	m := wsOverSSH{effecttest.NewMachine(), []byte("ws " + p.version + "\n"), []byte(stdout)}
	ws := wizard.NewSession(wizard.MacReach{SSH: wizard.SSH{M: m, Lock: "/t/l"}},
		wizard.Scripts{Remote: mac.RemoteScript})
	roles, err := ws.State(context.Background(), p.version)
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 2 || roles[1].Role != "tmux" || len(roles[1].Steps) == 0 {
		t.Errorf("roles = %+v", roles)
	}
}
