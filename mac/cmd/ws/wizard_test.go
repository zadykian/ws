package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

// testWizardEnv returns the environment of a Mac whose Command Line Tools and Homebrew are there.
func testWizardEnv(goos string, uid int, stderr *bytes.Buffer) wizardEnv {
	env := map[string]string{"TMPDIR": "/t/"}
	m := effecttest.NewMachine()
	m.Results["xcode-select -p"] = effecttest.Result{Out: []byte("/clt\n")}
	m.Files["/clt/usr/bin/git"] = nil
	m.Modes["/clt/usr/bin/git"] = 0o755
	m.Results["/opt/homebrew/bin/brew --version"] = effecttest.Result{}
	return wizardEnv{goos: goos, uid: uid, pid: 7, m: m,
		getenv: func(k string) string { return env[k] },
		lookup: func(k string) (string, bool) { v, ok := env[k]; return v, ok }, pwd: "/p", stdin: strings.NewReader(""), stderr: stderr}
}

func TestWizardCommandRefuses(t *testing.T) {
	for name, tc := range map[string]struct {
		args        []string
		goos        string
		uid, status int
		message     string
	}{
		"argument": {[]string{"--now"}, "darwin", 501, 1, "ws mac wizard: unknown argument '--now'; usage: ws mac wizard"},
		"linux":    {nil, "linux", 501, 1, "ws mac wizard: it sets up a Mac, not Linux"},
		"root":     {nil, "darwin", 0, 1, "run it as yourself, not root: it asks sudo where it needs root"},
	} {
		var stderr bytes.Buffer
		got := wizardCommand(context.Background(), tc.args, testWizardEnv(tc.goos, tc.uid, &stderr))
		if got != tc.status || !strings.Contains(stderr.String(), tc.message) {
			t.Errorf("%s: status = %d, stderr = %q; want %d, %q", name, got, stderr.String(), tc.status, tc.message)
		}
	}
}

func TestWizardCommandHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	e := testWizardEnv("darwin", 501, &stderr)
	e.stdout = &stdout
	if got := wizardCommand(context.Background(), []string{"-h"}, e); got != 0 ||
		stdout.String() != "usage: ws mac wizard [--check] [--update]\n" || stderr.Len() != 0 {
		t.Errorf("status = %d, stdout = %q, stderr = %q; want the usage on stdout", got, stdout.String(), stderr.String())
	}
}

func TestWizardCommandRuns(t *testing.T) {
	var stderr bytes.Buffer
	e := testWizardEnv("darwin", 501, &stderr)
	if got := wizardCommand(context.Background(), []string{"--check"}, e); got != 1 ||
		!strings.Contains(stderr.String(), "Answers: no terminal to ask WS_TARGET: set WS_TARGET") {
		t.Errorf("status = %d, stderr = %q; want the refusal to ask with no terminal", got, stderr.String())
	}
	if calls := e.m.(*effecttest.Machine).Calls; len(calls) == 0 || calls[0] != "mkdir /t/ws-wizard.lock 700" {
		t.Errorf("calls = %v; want the lock first", calls)
	}
}

func TestWizardCheckShowsTheAnswersDiff(t *testing.T) {
	var stderr bytes.Buffer
	e := testWizardEnv("darwin", 501, &stderr)
	env := map[string]string{"WS_TARGET": "cloud", "WS_REMOTE": "r.example", "WS_REMOTE_PORT": "22", "WS_GIT_NAME": "",
		"WS_TUNNEL_NETWORKS": "", "WS_RESOLVERS": "", "WS_HOOK": ""}
	e.lookup = func(k string) (string, bool) { v, ok := env[k]; return v, ok }
	if got := wizardCommand(context.Background(), []string{"--check"}, e); got != 0 || !strings.Contains(stderr.String(), "+WS_TARGET=cloud") {
		t.Errorf("status = %d, stderr = %q; want the diff of mac.conf", got, stderr.String())
	}
	if got := string(e.m.(*effecttest.Machine).Files["/t/ws-wizard.lock/mac.conf"]); !strings.HasPrefix(got, "WS_TARGET=cloud\n") {
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
	var stderr bytes.Buffer
	e := testWizardEnv("darwin", 501, &stderr)
	m := e.m.(*effecttest.Machine)
	m.Results["xcode-select -p"] = effecttest.Result{Err: errors.New("exit status 2")}
	m.Results["/opt/homebrew/bin/brew --version"] = effecttest.Result{Err: errors.New("no brew")}
	if got := wizardCommand(context.Background(), []string{"--check"}, e); got != 0 {
		t.Fatalf("status = %d, stderr = %q", got, stderr.String())
	}
	for _, want := range []string{"Tools: would run xcode-select --install", "Homebrew: after Tools", "Answers: after Homebrew",
		"gh on the Mac: after Homebrew", "ws mac apply: after Homebrew"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr = %q, want %q", stderr.String(), want)
		}
	}
}

func TestWizardUpdateRunsTheNewWs(t *testing.T) {
	var stderr bytes.Buffer
	e := testWizardEnv("darwin", 501, &stderr)
	m := e.m.(*effecttest.Machine)
	e.version, e.releases, e.argv = "v1.0.0", "http://127.0.0.1:22080", []string{"mac", "wizard", "--update"}
	env := map[string]string{"HOME": "/h"}
	e.getenv = func(k string) string { return env[k] }
	m.Results["curl -fsS -o /dev/null -w %{redirect_url} http://127.0.0.1:22080/latest"] =
		effecttest.Result{Out: []byte("http://127.0.0.1:22080/tag/v1.1.0")}
	m.Results["mktemp -d"] = effecttest.Result{Out: []byte("/x\n")}
	for _, c := range []string{
		"curl -fsSL --retry 3 --proto-redir =https -o /x/install.sh http://127.0.0.1:22080/download/v1.1.0/install.sh",
		"env WS_VERSION=v1.1.0 WS_RELEASES_URL=http://127.0.0.1:22080 sh /x/install.sh", "rm -rf /x"} {
		m.Results[c] = effecttest.Result{}
	}
	// The fake Exec returns, so the step finds its change again: the command fails, after the exec.
	got := wizardCommand(context.Background(), []string{"--update"}, e)
	calls := strings.Join(m.Calls, "\n")
	if got != 1 || !strings.Contains(calls, "exec /h/.local/bin/ws mac wizard --update") || strings.Contains(calls, "mkdir /t/ws-wizard.lock") {
		t.Errorf("status = %d, calls = %q; want the exec of the new ws, before the lock", got, calls)
	}
}

func TestWizardUpdateCheckInstallsNothing(t *testing.T) {
	var stderr bytes.Buffer
	e := testWizardEnv("darwin", 501, &stderr)
	m := e.m.(*effecttest.Machine)
	e.version, e.releases = "v1.0.0", "http://127.0.0.1:22080"
	m.Results["curl -fsS -o /dev/null -w %{redirect_url} http://127.0.0.1:22080/latest"] =
		effecttest.Result{Out: []byte("http://127.0.0.1:22080/tag/v1.1.0")}
	wizardCommand(context.Background(), []string{"--check", "--update"}, e)
	if !strings.Contains(stderr.String(), "ws on the Mac: would install ws v1.1.0 on the Mac") || strings.Contains(strings.Join(m.Calls, "\n"), "exec") {
		t.Errorf("stderr = %q, calls = %v; want the plan and no exec", stderr.String(), m.Calls)
	}
}
