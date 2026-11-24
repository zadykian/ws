package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The hook's script stands for ws: it logs its arguments and its standard input, and exits as the
// test says, with 0 for the commit's hook.
func TestGitHookRunsWsPrePushOnAPush(t *testing.T) {
	skipBelowHooks(t)
	r := newRepo(t)
	m, e := machine(t)
	if err := write(t, e, units(t, e, "file/git-config")); err != nil {
		t.Fatal(err)
	}
	home := filepath.Dir(r.dir)
	log := filepath.Join(home, "log")
	script := filepath.Join(home, "ws-stub")
	body := "#!/bin/sh\necho \"$@\" >" + log + "\ncat >>" + log + "\nexit \"${STATUS-0}\"\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil { //nolint:gosec // the test's own script
		t.Fatal(err)
	}
	config := strings.ReplaceAll(string(m.Files["/etc/ws/gitconfig"]), "/usr/local/bin/ws", script)
	if strings.Contains(config, "/etc/ws/tokens.gitconfig") {
		config = strings.ReplaceAll(config, "/etc/ws/tokens.gitconfig", filepath.Join(home, "none"))
	}
	wsConfig := filepath.Join(home, "ws.gitconfig")
	if err := os.WriteFile(wsConfig, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	r.git("config", "--file", r.env[2][len("GIT_CONFIG_GLOBAL="):], "include.path", wsConfig)
	r.git("config", "commit.gpgsign", "false")
	r.git("commit", "-q", "--allow-empty", "-m", "x")
	r.git("remote", "add", "origin", r.remote)
	sha := r.git("rev-parse", "HEAD")
	r.env = append(r.env, "STATUS=1")
	cmd := exec.CommandContext(t.Context(), "git", "push", "-q", "origin", "main")
	cmd.Dir, cmd.Env = r.dir, r.env
	if out, err := cmd.CombinedOutput(); err == nil {
		t.Errorf("a failing hook did not refuse the push: %s", out)
	}
	got, err := os.ReadFile(log) //nolint:gosec // the test's own path
	if err != nil {
		t.Fatal(err)
	}
	if want := "ubuntu guest git pre-push origin " + r.remote + "\nrefs/heads/main " + sha + " refs/heads/main " + zeros + "\n"; string(got) != want {
		t.Errorf("hook saw %q, want %q", got, want)
	}
}
