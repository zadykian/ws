package fresh

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/step"
)

// setup says how run builds the wrapper's surroundings. env is the one variable the wrapper sees
// beside HOME and PATH. terminal drops the wrapper's test for a terminal (a test's stdin and stdout
// are pipes). fakeTmux puts a tmux stub in PATH, and ext is the extended-keys option it reports.
// Without noConfig, HOME holds ~/.config/fresh/config.json.
type setup struct {
	env                          string
	fakeTmux, terminal, noConfig bool
	ext                          string
}

// run runs the wrapper with a stub for Fresh, which notes its arguments, and a stub for tmux that
// logs a set alone. The home is "~" in the output.
func run(t *testing.T, s setup) (out, tmux string, code int) {
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	stub := filepath.Join(dir, "editor")
	write := func(path, body string) {
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil { //nolint:gosec // a test's stub runs
			t.Fatal(err)
		}
	}
	write(stub, "echo \"editor $*\"\nexit 7\n")
	if s.fakeTmux {
		write(filepath.Join(dir, "tmux"), "case \"$1\" in\nshow) echo "+s.ext+" ;;\n*) echo \"$*\" >>"+log+" ;;\nesac\n")
	}
	if !s.noConfig {
		fresh := filepath.Join(dir, ".config", "fresh")
		if err := os.MkdirAll(fresh, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(fresh, "config.json"), []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	script := strings.ReplaceAll(string(wrapper), "/usr/bin/fresh", stub)
	if s.terminal {
		script = strings.ReplaceAll(script, ` || [ ! -t 0 ] || [ ! -t 1 ]`, "")
	}
	path := filepath.Join(dir, "wrapper")
	write(path, script)
	cmd := exec.CommandContext(t.Context(), "/bin/sh", path, "a", "b c") //nolint:gosec // the test's own script
	cmd.Env = []string{"PATH=" + dir, "HOME=" + dir, s.env}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("run: %v", err)
	}
	logged, _ := os.ReadFile(log) //nolint:gosec // the test's own file; no tmux leaves none
	return strings.ReplaceAll(stdout.String(), dir, "~"), string(logged), ee.ExitCode()
}

const conf = "--config ~/.config/fresh/config.json "

func TestTheWrapperAsksInTmuxAndResets(t *testing.T) {
	out, tmux, code := run(t, setup{env: "TMUX=/tmp/s,1,0", fakeTmux: true, terminal: true, ext: "off"})
	if want := "\033[>4;2meditor " + conf + "a b c\n\033[>4m"; out != want {
		t.Errorf("out %q, want %q", out, want)
	}
	if tmux != "set -s extended-keys on\n" {
		t.Errorf("tmux got %q", tmux)
	}
	if code != 7 {
		t.Errorf("exit %d, want Fresh's", code)
	}
}

func TestTheWrapperLeavesExtendedKeysAtAlways(t *testing.T) {
	for _, ext := range []string{"always", "on"} {
		out, tmux, _ := run(t, setup{env: "TMUX=/tmp/s,1,0", fakeTmux: true, terminal: true, ext: ext})
		if tmux != "" {
			t.Errorf("at %s tmux got %q", ext, tmux)
		}
		if want := "\033[>4;2meditor " + conf + "a b c\n\033[>4m"; out != want {
			t.Errorf("at %s out %q, want %q", ext, out, want)
		}
	}
}

func TestTheWrapperPassesThroughOutsideTmux(t *testing.T) {
	out, tmux, code := run(t, setup{env: "TERM=xterm", fakeTmux: true, terminal: true, ext: "off"})
	if out != "editor "+conf+"a b c\n" || tmux != "" || code != 7 {
		t.Errorf("out %q, tmux %q, exit %d", out, tmux, code)
	}
}

func TestTheWrapperPassesAPipeThrough(t *testing.T) {
	out, tmux, code := run(t, setup{env: "TMUX=/tmp/s,1,0", fakeTmux: true, ext: "off"})
	if out != "editor "+conf+"a b c\n" || tmux != "" || code != 7 {
		t.Errorf("out %q, tmux %q, exit %d", out, tmux, code)
	}
}

func TestTheWrapperPassesNoConfigWhereThereIsNone(t *testing.T) {
	for _, terminal := range []bool{false, true} {
		out, _, _ := run(t, setup{env: "TMUX=/tmp/s,1,0", fakeTmux: true, terminal: terminal, noConfig: true, ext: "always"})
		if strings.Contains(out, "--config") {
			t.Errorf("terminal %v: out %q", terminal, out)
		}
	}
}

func TestTheWrapperRunsWithNoTmuxBinary(t *testing.T) {
	if _, _, code := run(t, setup{env: "TMUX=/tmp/s,1,0", terminal: true}); code != 7 {
		t.Errorf("exit %d, want Fresh's", code)
	}
}

func TestTheWrapperIsWrittenForEveryUser(t *testing.T) {
	m, v := userMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Packages = effecttest.Packages{"fresh-editor": {Version: version, Status: "ii"}}
	out := &effecttest.Out{}
	if _, err := step.Run(context.Background(), Steps(v, secrets.Secrets{}, m, out), out, false); err != nil {
		t.Fatal(err)
	}
	if got := m.Files[wrapperPath]; !bytes.Equal(got, wrapper) {
		t.Errorf("%s holds %q", wrapperPath, got)
	}
	if got := m.Modes[wrapperPath]; got != 0o755 {
		t.Errorf("mode %o", got)
	}
}
