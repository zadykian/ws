package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"github.com/zadykian/ws/engine/cli/clitest"
)

// TestNoRoleSetsAFramesKey keeps each role off the keys of the server's frame, which loads before
// the roles through the side's link.
func TestNoRoleSetsAFramesKey(t *testing.T) {
	clitest.FrameKeys(t, []string{treeRoot + "/mise.toml", treeRoot + "/mise.guest.toml"}, rolesDir+"/*/mise.*.toml")
}

// checkCommand returns the shell command of the frame's check against a silent skip, as mise's
// template gives it to sh, run in dir.
func checkCommand(t *testing.T, dir string) string {
	return frameCommand(t, "/etc/ws/units.verified", dir)
}

// frameCommand returns the shell command of the template that the frame renders at target, as
// mise gives it to sh. It names dir where the template names the run's directory.
func frameCommand(t *testing.T, target, dir string) string {
	text, err := os.ReadFile(treeRoot + "/mise.guest.toml")
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Bootstrap struct {
			Files map[string]struct {
				Content string `toml:"content"`
			} `toml:"files"`
		} `toml:"bootstrap"`
	}
	if err := toml.Unmarshal(text, &c); err != nil {
		t.Fatal(err)
	}
	content := c.Bootstrap.Files[target].Content
	command, prefixed := strings.CutPrefix(strings.TrimSpace(content), `{{ exec(command="`)
	command, suffixed := strings.CutSuffix(command, `") }}`)
	if !prefixed || !suffixed {
		t.Fatalf("%s's content is no exec of one command: %q", target, content)
	}
	return strings.ReplaceAll(strings.ReplaceAll(command, `" ~ cwd ~ "`, dir), `\"`, `"`)
}

// TestTheCheckNeedsTheSidesDirectory runs the frame's check with a stand-in for ws. In the side's
// run directory it passes, and elsewhere it fails and names the directory to run in. A run from
// the root, or from the Mac's run directory, loads no role of the server's, and would set up the
// frame alone with exit 0. A run with no ws fails and names install.sh.
func TestTheCheckNeedsTheSidesDirectory(t *testing.T) {
	bin := t.TempDir()
	ws := filepath.Join(bin, "ws")
	if err := os.WriteFile(ws, []byte("#!/bin/sh\necho 'ws units: none missing'\n"), 0o700); err != nil { //nolint:gosec // a stand-in for ws, which sh runs
		t.Fatal(err)
	}
	side, mac, root := t.TempDir(), t.TempDir(), t.TempDir()
	for _, file := range []string{
		filepath.Join(side, "mise", "conf.d", "frame", "mise.guest.toml"),
		filepath.Join(mac, "mise", "conf.d", "zed", "mise.macos.toml"),
	} {
		if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	refusal := ", which loads no role of the guest: run it as mise -C /opt/ws/ubuntu/guest"
	for _, c := range []struct {
		name, dir, ws, stdout, stderr string
		status                        int
	}{
		{"the side's directory", side, ws, "ws units: none missing\n", "", 0},
		{"the Mac's directory", mac, ws, "", "mise runs in " + mac + refusal, 1},
		{"the root", root, ws, "", "mise runs in " + root + refusal, 1},
		{"no ws", side, filepath.Join(bin, "missing"), "", "is missing: install.sh writes it", 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			command := strings.ReplaceAll(checkCommand(t, c.dir), "/usr/local/bin/ws", c.ws)
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
