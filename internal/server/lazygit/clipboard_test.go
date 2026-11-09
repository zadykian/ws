package lazygit

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// command returns the config's command at key, with placeholder filled as lazygit quotes text.
func command(t *testing.T, key, placeholder, text string) string {
	t.Helper()
	v, _ := leaves(t)[key].(string)
	if v == "" {
		t.Fatalf("no %s", key)
	}
	quoted := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`, "`", "\\`").Replace(text)
	return strings.ReplaceAll(v, placeholder, `"`+quoted+`"`)
}

// In tmux, a copy goes to tmux's buffer, whose -w sends it on to the client's terminal.
func TestACopyGoesThroughTmuxInTmux(t *testing.T) {
	bin := t.TempDir()
	log := filepath.Join(bin, "log")
	stub := "#!/bin/sh\necho \"$*\" >>" + log + "\ncat >>" + log + "\n"
	if err := os.WriteFile(filepath.Join(bin, "tmux"), []byte(stub), 0o755); err != nil { //nolint:gosec // a test's stub runs
		t.Fatal(err)
	}
	text := "a \"b\" $c `d` \\e\nf"
	script := command(t, "os.copyToClipboardCmd", "{{text}}", text)
	cmd := exec.CommandContext(t.Context(), "/bin/sh", "-c", script) //nolint:gosec // the config's own command
	cmd.Env = []string{"PATH=" + bin + ":/usr/bin:/bin", "TMUX=/tmp/s,1,0"}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	logged, err := os.ReadFile(log) //nolint:gosec // the test's own log
	if err != nil {
		t.Fatal(err)
	}
	if want := "load-buffer -w -\n" + text; string(logged) != want {
		t.Errorf("tmux got %q, want %q", logged, want)
	}
}
