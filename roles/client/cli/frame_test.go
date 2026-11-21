package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// valuesCheckFile is the frame's file that runs the values check, in the user's home.
const valuesCheckFile = "~/Library/Caches/ws-values.checked"

// valuesCheckPart reads the frame's file of the values check from the Mac's root config.
func valuesCheckPart(t *testing.T) map[string]any {
	text, err := os.ReadFile(macTreeRoot + "/" + macRoleFile)
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Bootstrap struct {
			Files map[string]map[string]any `toml:"files"`
		} `toml:"bootstrap"`
	}
	if err := toml.Unmarshal(text, &c); err != nil {
		t.Fatal(err)
	}
	part, ok := c.Bootstrap.Files[valuesCheckFile]
	if !ok {
		t.Fatalf("%s holds no [bootstrap.files.%q]: the Mac's values go unchecked until a unit reads one", macRoleFile, valuesCheckFile)
	}
	return part
}

// TestValuesCheckFileIsTheUsersTemplate keeps the file that runs the check in the user's home with
// no owner or group, as mise reads and writes it as the user. A template renders before the
// packages' phase, where a unit would run.
func TestValuesCheckFileIsTheUsersTemplate(t *testing.T) {
	part := valuesCheckPart(t)
	if part["template"] != true {
		t.Errorf("template = %v, want true", part["template"])
	}
	for _, k := range []string{"owner", "group", "mode"} {
		if v, ok := part[k]; ok {
			t.Errorf("%s = %v: a file with no owner or group is the user's", k, v)
		}
	}
}

// TestValuesCheckRunsTheCommand runs the file's command against a stand-in for /usr/local/bin/ws.
// A pass prints the command's own line, so that the next plan finds the file unchanged. A bad value
// fails the run with the command's words, and a run with no ws fails naming install.sh.
func TestValuesCheckRunsTheCommand(t *testing.T) {
	content, _ := valuesCheckPart(t)["content"].(string)
	command, prefixed := strings.CutPrefix(strings.TrimSpace(content), `{{ exec(command="`)
	command, suffixed := strings.CutSuffix(command, `") }}`)
	if !prefixed || !suffixed {
		t.Fatalf("the file's content is no exec of one command: %q", content)
	}
	command = strings.ReplaceAll(command, `\"`, `"`)
	for _, c := range []struct {
		name, ws, stdout, stderr string
		status                   int
	}{
		{"no ws", "", "", "is missing: install.sh writes it", 1},
		{"a pass", "echo " + valuesChecked, valuesChecked + "\n", "", 0},
		{"a bad key", "echo '/etc/ws/values.yml, line 1: WS_SLOT=9 is bad' >&2; exit 1", "", "WS_SLOT=9 is bad", 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			ws := filepath.Join(t.TempDir(), "ws")
			if c.ws != "" {
				script := "#!/bin/sh\n[ \"$*\" = 'client values check' ] || exit 2\n" + c.ws + "\n"
				if err := os.WriteFile(ws, []byte(script), 0o700); err != nil { //nolint:gosec // a stand-in that sh runs
					t.Fatal(err)
				}
			}
			cmd := exec.Command("sh", "-o", "errexit", "-c", strings.ReplaceAll(command, "/usr/local/bin/ws", ws)) //nolint:gosec // the frame's own command, read from the tree
			cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			if err := cmd.Run(); cmd.ProcessState == nil {
				t.Fatal(err)
			}
			if status := cmd.ProcessState.ExitCode(); status != c.status || stdout.String() != c.stdout || !strings.Contains(stderr.String(), c.stderr) {
				t.Errorf("exit %d, stdout %q, stderr %q; want %d, %q and %q", status, stdout.String(), stderr.String(), c.status, c.stdout, c.stderr)
			}
		})
	}
}

// wsBinaryUnit reads the frame's unit that keeps /usr/local/bin/ws, whose spec holds no template.
func wsBinaryUnit(t *testing.T) (command, check []string, user string) {
	text, err := os.ReadFile(macRolesDir + "/frame/units.toml.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	var s struct {
		State map[string]struct {
			Run struct {
				User    string   `toml:"user"`
				Command []string `toml:"command"`
				Check   []string `toml:"check"`
			} `toml:"run"`
		} `toml:"state"`
	}
	if err := toml.Unmarshal(text, &s); err != nil {
		t.Fatal(err)
	}
	u, ok := s.State["ws-binary"]
	if !ok {
		t.Fatal("the frame's spec holds no state ws-binary")
	}
	return u.Run.Command, u.Run.Check, u.Run.User
}

// TestWsBinaryUnitKeepsTheFileWithNoCopy holds the unit to the owner and the mode of the file that
// install.sh writes: root runs it, and nothing in it copies ws.
func TestWsBinaryUnitKeepsTheFileWithNoCopy(t *testing.T) {
	command, check, user := wsBinaryUnit(t)
	if user != "root" {
		t.Errorf("user = %q, want root: the owner is set by root", user)
	}
	cmd, chk := strings.Join(command, " "), strings.Join(check, " ")
	for _, want := range []string{"chown 0:0 /usr/local/bin/ws", "chmod 0755 /usr/local/bin/ws"} {
		if !strings.Contains(cmd, want) {
			t.Errorf("the command lacks %q: %s", want, cmd)
		}
	}
	if !strings.Contains(chk, "stat -f %u:%g:%Lp /usr/local/bin/ws") || !strings.Contains(chk, "0:0:755") {
		t.Errorf("the check does not read the owner and the mode: %s", chk)
	}
	for _, text := range []string{cmd, chk} {
		for _, copying := range []string{"cmp", "install -", "mv ", "cp ", "{{"} {
			if strings.Contains(text, copying) {
				t.Errorf("%q holds %q: the unit keeps the file and copies none", text, copying)
			}
		}
	}
}

// TestWsBinaryUnitFailsWithNoFile runs the command where /usr/local/bin/ws is missing: it fails
// before any change and names install.sh.
func TestWsBinaryUnitFailsWithNoFile(t *testing.T) {
	command, _, _ := wsBinaryUnit(t)
	missing := filepath.Join(t.TempDir(), "ws")
	script := strings.ReplaceAll(command[2], "/usr/local/bin/ws", missing)
	var stderr bytes.Buffer
	cmd := exec.Command("sh", "-c", script) //nolint:gosec // the frame's own command, read from the tree
	cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Fatal("the command passed with no file")
	}
	if !strings.Contains(stderr.String(), "install.sh writes it") {
		t.Errorf("stderr %q does not name install.sh", stderr.String())
	}
}
