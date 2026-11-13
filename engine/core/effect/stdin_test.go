package effect_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
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
		{"none ends at once", effect.NoStdin, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			m := effect.OSMachine{Stdin: tt.stdin, Stderr: &out}
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
	if out, err := (effect.OSMachine{}).Run(t.Context(), "sh", "-c", show); err != nil || string(out) != "1 own never\n" {
		t.Errorf("a plain run gave %q, %v; want ws's own", out, err)
	}
	var passed bytes.Buffer
	m := effect.OSMachine{Stdin: effect.NoStdin, Stderr: &passed}
	if out, err := m.Run(t.Context(), "sh", "-c", show); err != nil || string(out) != want {
		t.Errorf("Run gave %q, %v; want %q", out, err, want)
	}
	if err := m.Pass(t.Context(), false, "sh", "-c", show); err != nil || passed.String() != want {
		t.Errorf("Pass gave %q, %v; want %q", passed.String(), err, want)
	}
}
