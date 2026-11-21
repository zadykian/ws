package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestTheValuesCheckRunsTheFramesWs runs the command of /etc/ws/values.checked with stand-ins for
// ws. WS_SELF names the ws, and /usr/local/bin/ws is the default. The check passes with the line
// of a pass where there is no ws. It passes so too where the ws lists no check among the commands
// of values, as the pin's ws of a release before it.
func TestTheValuesCheckRunsTheFramesWs(t *testing.T) {
	dir := t.TempDir()
	stand := func(name, body string) string {
		file := filepath.Join(dir, name)
		if err := os.WriteFile(file, []byte("#!/bin/sh\n"+body), 0o700); err != nil { //nolint:gosec // a stand-in for ws, which sh runs
			t.Fatal(err)
		}
		return file
	}
	// The stand-ins list check as ws does, and answer it with their own line.
	listing := "if [ \"$3\" = --help ]; then printf 'Available Commands:\\n  check       refuse\\n  set         write\\n'; exit 0; fi\n"
	passes := stand("passes", listing+"echo \"$*\"\n")
	fails := stand("fails", listing+"echo 'WS_BASE_SWAPPINESS is 400' >&2\nexit 1\n")
	fallback := stand("fallback", listing+"echo default\n")
	oldWs := stand("old", "if [ \"$3\" = --help ]; then printf 'Available Commands:\\n  set         write\\n'; exit 0; fi\necho \"unknown command\" >&2\nexit 2\n")
	missing := filepath.Join(dir, "missing")
	for _, c := range []struct {
		name, self, stdout, stderr string
		status                     int
	}{
		{"WS_SELF passes", passes, "server values check\n", "", 0},
		{"WS_SELF refuses", fails, "", "WS_BASE_SWAPPINESS is 400", 1},
		{"no WS_SELF takes the default", "", "default\n", "", 0},
		{"WS_SELF names no file", missing, valuesChecked + "\n", "no ws that checks them yet", 0},
		{"WS_SELF lists no check", oldWs, valuesChecked + "\n", "no ws that checks them yet", 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			command := strings.ReplaceAll(frameCommand(t, "/etc/ws/values.checked", dir), "/usr/local/bin/ws", fallback)
			cmd := exec.Command("sh", "-o", "errexit", "-c", command) //nolint:gosec // the frame's own command, read from the tree
			cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
			if c.self != "" {
				cmd.Env = append(cmd.Env, "WS_SELF="+c.self)
			}
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			if err := cmd.Run(); cmd.ProcessState == nil {
				t.Fatal(err)
			}
			status := cmd.ProcessState.ExitCode()
			if status != c.status || stdout.String() != c.stdout || !strings.Contains(stderr.String(), c.stderr) {
				t.Errorf("exit %d, stdout %q, stderr %q, want %d, %q and %q", status, stdout.String(), stderr.String(), c.status, c.stdout, c.stderr)
			}
		})
	}
}
