package local_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/local"
)

// typed puts a file that holds text in place of os.Stdin, as a test's own is empty.
func typed(t *testing.T, text string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stdin")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path) //nolint:gosec // the test's own temporary file
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdin
	os.Stdin = f
	t.Cleanup(func() {
		os.Stdin = old
		_ = f.Close()
	})
}

func TestOSMachinePassStdin(t *testing.T) {
	typed(t, "typed\n")
	tests := []struct {
		name  string
		stdin io.Reader
		want  string
	}{
		{"nil keeps ws's own", nil, "typed\n"},
		{"a reader", strings.NewReader("given\n"), "given\n"},
		{"none ends at once", local.NoStdin, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			m := local.OSMachine{Stdin: tt.stdin, Stderr: &out}
			if err := m.Pass(t.Context(), false, "cat"); err != nil {
				t.Fatal(err)
			}
			if out.String() != tt.want {
				t.Errorf("cat read %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestOSMachineStdinStopsGitAndSSHAsking(t *testing.T) {
	t.Setenv("GIT_TERMINAL_PROMPT", "1")
	t.Setenv("SSH_ASKPASS", "own")
	t.Setenv("SSH_ASKPASS_REQUIRE", "never")
	const show, want = `echo "$GIT_TERMINAL_PROMPT $SSH_ASKPASS $SSH_ASKPASS_REQUIRE"`, "0 false force\n"
	if out, err := (local.OSMachine{}).Run(t.Context(), "sh", "-c", show); err != nil || string(out) != "1 own never\n" {
		t.Errorf("a plain run gave %q, %v; want ws's own", out, err)
	}
	var passed bytes.Buffer
	m := local.OSMachine{Stdin: local.NoStdin, Stderr: &passed}
	if out, err := m.Run(t.Context(), "sh", "-c", show); err != nil || string(out) != want {
		t.Errorf("Run gave %q, %v; want %q", out, err, want)
	}
	if err := m.Pass(t.Context(), false, "sh", "-c", show); err != nil || passed.String() != want {
		t.Errorf("Pass gave %q, %v; want %q", passed.String(), err, want)
	}
}

// A PIN that a command asks for comes from ws's own stdin, nil Stdin as the Mac's, and its prompt
// shows. The value it prints comes back, never shown and never in an error.
func TestOSMachineRunTerminalShowsThePromptAndKeepsTheOutput(t *testing.T) {
	typed(t, "1234\n")
	var shown bytes.Buffer
	m := local.OSMachine{Stderr: &shown}
	out, err := m.RunTerminal(t.Context(), "sh", "-c", `read -r pin; echo asked >&2; echo "value of $pin"`)
	if err != nil || string(out) != "value of 1234\n" || shown.String() != "asked\n" {
		t.Errorf("RunTerminal = %q, %v, and showed %q; want the value, and asked shown", out, err, shown.String())
	}
	_, err = m.RunTerminal(t.Context(), "sh", "-c", `echo "$1$1"; echo failed >&2; exit 3`, "sh", "ab")
	if err == nil || strings.Contains(err.Error(), "abab") || strings.Contains(shown.String(), "abab") {
		t.Errorf("a failed RunTerminal gave %v, and showed %q; want an error without its output", err, shown.String())
	}
}

// The input goes to the command that sudo runs, as a password that ikeward reads on stdin.
func TestOSMachineRunRootInputGivesTheCommandItsInput(t *testing.T) {
	m := local.OSMachine{Sudo: []string{"env", "WS_TEST=1"}}
	out, err := m.RunRootInput(t.Context(), []byte("pw\n"), "sh", "-c", `read -r p; echo "$WS_TEST $p"`)
	if err != nil || string(out) != "1 pw\n" {
		t.Errorf("RunRootInput = %q, %v; want 1 pw, as root with the input", out, err)
	}
}
