package lazygit

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// wrapped says how run builds the wrapper's surroundings: terminal drops the test for a terminal,
// as a test's stdin and stdout are pipes.
type wrapped struct {
	dir      string
	terminal bool
}

// run runs the wrapper in w.dir, with a stub for lazygit that prints its arguments and exits 3,
// and returns the output and the exit status.
func run(t *testing.T, w wrapped) (out string, code int) {
	t.Helper()
	bin := t.TempDir()
	write := func(path, body string) {
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil { //nolint:gosec // a test's stub runs
			t.Fatal(err)
		}
	}
	stub := filepath.Join(bin, "lazygit")
	write(stub, "echo \"lazygit $*\"\nexit 3\n")
	env := []string{"PATH=" + bin + ":/usr/bin:/bin", "HOME=" + bin}
	script := strings.ReplaceAll(string(wrapper), "/usr/bin/lazygit", stub)
	if w.terminal {
		script = strings.ReplaceAll(script, "[ ! -t 0 ] || [ ! -t 1 ]", "false")
	}
	path := filepath.Join(bin, "wrapper")
	write(path, script)
	cmd := exec.CommandContext(t.Context(), "/bin/sh", path, "a", "b c") //nolint:gosec // the test's own script
	cmd.Dir, cmd.Env = w.dir, env
	cmd.WaitDelay = 5 * time.Second
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("run: %v", err)
	}
	return stdout.String(), ee.ExitCode()
}

func TestTheWrapperPassesAPipeThrough(t *testing.T) {
	out, code := run(t, wrapped{dir: t.TempDir()})
	if out != "lazygit a b c\n" || code != 3 {
		t.Errorf("out %q, exit %d", out, code)
	}
}
