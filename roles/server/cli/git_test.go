package cli

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/ubuntu/git"
)

// gitAnswers sets what git config answers of a text on stdin: its include paths and its names.
func gitAnswers(m *effecttest.Machine, paths, names string) {
	m.Results["git config --file - --type path --get-all include.path"] = effecttest.Result{Out: []byte(paths)}
	m.Results["git config --file - --name-only --list"] = effecttest.Result{Out: []byte(names)}
}

func TestGitIncludedExits(t *testing.T) {
	for _, c := range []struct {
		paths, names string
		status       int
		stderr       string
	}{
		{"/etc/ws/gitconfig\n", "include.path\nuser.name\n", 0, ""},
		{"", "user.name\n", 1, ""},
		{"/etc/ws/gitconfig\n", "user.name\ninclude.path\n", 2,
			"ws server ubuntu git included: ~/.gitconfig includes /etc/ws/gitconfig below its own settings, " +
				"which it then overrides: move the include to the top\n"},
	} {
		p, m := valuesProcess("")
		gitAnswers(m, c.paths, c.names)
		status, stdout, stderr := runWs(p, "[x]\n", "server", "ubuntu", "git", "included")
		if status != c.status || stdout != "" || stderr != c.stderr {
			t.Errorf("names %q: status = %d, stdout %q, stderr %q; want %d and %q", c.names, status, stdout, stderr, c.status, c.stderr)
		}
	}
}

func TestGitCheckPassesWithNothingToRefuse(t *testing.T) {
	p, _ := valuesProcess("---\nWS_USER: root\n")
	if status, stdout, stderr := runWs(p, "", "server", "ubuntu", "git", "check"); status != 0 || stdout != "" || stderr != "" {
		t.Errorf("status = %d, stdout %q, stderr %q", status, stdout, stderr)
	}
}

func TestGitCheckRefusesATokenInAClone(t *testing.T) {
	p, _ := valuesProcess("---\nWS_USER: root\nWS_GIT_CLONES:\n  - repo: https://me:hunter2@example.test/r.git\n    dest: r\n")
	status, _, stderr := runWs(p, "", "server", "ubuntu", "git", "check")
	if status != 1 || !strings.Contains(stderr, "each entry of WS_GIT_CLONES") || strings.Contains(stderr, "hunter2") {
		t.Errorf("status = %d, stderr %q", status, stderr)
	}
}

func TestGitTokensWithoutASecretsFileChangeNothing(t *testing.T) {
	p, m := valuesProcess("---\nWS_USER: root\n")
	if status, _, stderr := runWs(p, "", "server", "ubuntu", "git", "tokens", "--check"); status != 0 || stderr != "" {
		t.Errorf("status = %d, stderr %q", status, stderr)
	}
	if status, _, _ := runWs(p, "", "server", "ubuntu", "git", "tokens"); status != 0 {
		t.Errorf("tokens: status = %d", status)
	}
	if _, ok := m.Dirs["/etc/ws/tokens"]; ok {
		t.Error("tokens made /etc/ws/tokens with no secrets file")
	}
}

func TestGitClonesCheckFailsWhereACloneIsDue(t *testing.T) {
	p, m := valuesProcess("---\nWS_USER: root\nWS_GIT_CLONES:\n  - repo: /srv/r.git\n    dest: src/r\n")
	status, _, stderr := runWs(p, "", "server", "ubuntu", "git", "clones", "--check")
	if status != 1 || !strings.Contains(stderr, "to do: clone /srv/r.git into src/r") {
		t.Errorf("status = %d, stderr %q", status, stderr)
	}
	m.Dirs["/root/src/r"] = 0o755
	if status, _, stderr := runWs(p, "", "server", "ubuntu", "git", "clones", "--check"); status != 0 {
		t.Errorf("with the clone there: status = %d, stderr %q", status, stderr)
	}
}

// git runs the program as the account, which may not read values.yml: it reads the tokens alone.
func TestGitCredentialAnswersFromTheTokensFile(t *testing.T) {
	p, m := valuesProcess("")
	delete(m.Files, serverVars)
	m.Files[git.TokensFile] = []byte("[\"https://git.example.test\"]\ntoken = \"dummy-marker\"\n")
	status, stdout, stderr := runWs(p, "protocol=https\nhost=git.example.test\n\n", "server", "ubuntu", "git", "credential",
		"git.example.test", "get")
	if status != 0 || stdout != "username=token\npassword=dummy-marker\n" || stderr != "" {
		t.Errorf("get: status = %d, stdout %d bytes, stderr %q", status, len(stdout), stderr)
	}
	if status, stdout, _ := runWs(p, "", "server", "ubuntu", "git", "credential", "git.example.test", "erase"); status != 0 || stdout != "" {
		t.Errorf("erase: status = %d, stdout %d bytes; want 0 and nothing", status, len(stdout))
	}
	if status, _, _ := runWs(p, "", "server", "ubuntu", "git", "credential"); status != 2 {
		t.Errorf("without a key: status = %d, want a usage error", status)
	}
}
