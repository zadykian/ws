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

// TestValuesCheckRunsTheCommandAndPassesWithNoWs runs the file's command against stand-ins for mise
// and ws. A pass prints the command's own line, and so does a run that mise has no ws for yet, so
// that the next plan finds the file unchanged. A bad value fails the run with the command's words.
func TestValuesCheckRunsTheCommandAndPassesWithNoWs(t *testing.T) {
	content, _ := valuesCheckPart(t)["content"].(string)
	command, prefixed := strings.CutPrefix(strings.TrimSpace(content), `{{ exec(command="`)
	command, suffixed := strings.CutSuffix(command, `") }}`)
	if !prefixed || !suffixed {
		t.Fatalf("the file's content is no exec of one command: %q", content)
	}
	command = strings.ReplaceAll(command, `\"`, `"`)
	for _, c := range []struct {
		name, mise, ws, stdout, stderr string
		status                         int
	}{
		{"no ws yet", "exit 1", "", valuesChecked + "\n", "", 0},
		{"a pass", "echo $0", "echo " + valuesChecked, valuesChecked + "\n", "", 0},
		{"a bad key", "echo $0", "echo '/etc/ws/values.yml, line 1: WS_SLOT=9 is bad' >&2; exit 1", "", "WS_SLOT=9 is bad", 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			bin := t.TempDir()
			ws := filepath.Join(bin, "ws")
			stands := map[string]string{
				"mise": "#!/bin/sh\n" + strings.ReplaceAll(c.mise, "$0", ws) + "\n",
				"ws":   "#!/bin/sh\n[ \"$*\" = 'client values check' ] || exit 2\n" + c.ws + "\n",
			}
			for name, script := range stands {
				if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0o700); err != nil { //nolint:gosec // stand-ins that sh runs
					t.Fatal(err)
				}
			}
			cmd := exec.Command("sh", "-o", "errexit", "-c", command) //nolint:gosec // the frame's own command, read from the tree
			cmd.Env = []string{"PATH=" + bin}
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
