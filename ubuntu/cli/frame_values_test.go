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
// /usr/local/bin/ws. It passes the ws's line, refuses with the ws's message, and fails with no ws,
// naming install.sh.
func TestTheValuesCheckRunsTheFramesWs(t *testing.T) {
	for _, side := range []string{"guest", "host"} {
		t.Run(side, func(t *testing.T) { theValuesCheckRunsTheFramesWs(t, side) })
	}
}

// theValuesCheckRunsTheFramesWs is TestTheValuesCheckRunsTheFramesWs for one side.
func theValuesCheckRunsTheFramesWs(t *testing.T, side string) {
	dir := t.TempDir()
	stand := func(name, body string) string {
		file := filepath.Join(dir, name)
		if err := os.WriteFile(file, []byte("#!/bin/sh\n"+body), 0o700); err != nil { //nolint:gosec // a stand-in for ws, which sh runs
			t.Fatal(err)
		}
		return file
	}
	for _, c := range []struct {
		name, ws, stdout, stderr string
		status                   int
	}{
		{"a pass", stand("passes", "echo \"$*\"\n"), "ubuntu " + side + " values check\n", "", 0},
		{"a refusal", stand("fails", "echo 'WS_BASE_SWAPPINESS is 400' >&2\nexit 1\n"), "", "WS_BASE_SWAPPINESS is 400", 1},
		{"no ws", filepath.Join(dir, "missing"), "", "is missing: install.sh writes it", 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			command := strings.ReplaceAll(frameCommand(t, side, "/etc/ws/values.checked", dir), "/usr/local/bin/ws", c.ws)
			cmd := exec.Command("sh", "-o", "errexit", "-c", command) //nolint:gosec // the frame's own command, read from the tree
			cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
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
