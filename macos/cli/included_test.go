package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// runIncluded runs ws macos ROLE included with text on stdin.
func runIncluded(role, text string) (status int, stderr string) {
	p := fakeProcess()
	status, _, stderr = clitest.Run(p, []*cobra.Command{Command(p)}, text, "macos", role, "included")
	return status, stderr
}

func TestSSHIncludedSaysByItsStatus(t *testing.T) {
	for text, want := range map[string]int{
		"Include /etc/ws/ssh_config\n\nHost *\n    User me\n": 0,
		"# mine\ninclude \"/etc/ws/ssh_config\"\n":            0,
		"Host *\n    User me\n":                               1,
		"":                                                    1,
		"User me\nInclude /etc/ws/ssh_config\n":               2,
	} {
		status, stderr := runIncluded("ssh", text)
		if status != want || (want == 2) != strings.Contains(stderr, "line 2 of ~/.ssh/config") {
			t.Errorf("%q: status = %d, stderr = %q; want %d", text, status, stderr, want)
		}
	}
}

func TestGitIncludedSaysByItsStatus(t *testing.T) {
	p := fakeProcess()
	m := effecttest.NewMachine()
	p.Machine = m
	m.Results["git config --file - --type path --get-all include.path"] = effecttest.Result{Out: []byte("/etc/ws/gitconfig\n")}
	m.Results["git config --file - --name-only --list"] = effecttest.Result{Out: []byte("include.path\nuser.name\n")}
	run := func() (int, string) {
		status, _, stderr := clitest.Run(p, []*cobra.Command{Command(p)}, "[include]\n", "macos", "git", "included")
		return status, stderr
	}
	if status, stderr := run(); status != 0 || m.Inputs[0] != "[include]\n" {
		t.Errorf("first: status = %d, stderr = %q, stdin %q; want 0, git given the text", status, stderr, m.Inputs)
	}
	m.Results["git config --file - --name-only --list"] = effecttest.Result{Out: []byte("user.name\ninclude.path\n")}
	if status, stderr := run(); status != 2 || !strings.Contains(stderr, "~/.gitconfig") {
		t.Errorf("below: status = %d, stderr = %q; want 2 and the reason", status, stderr)
	}
	m.Results["git config --file - --type path --get-all include.path"] = effecttest.Result{}
	m.Results["git config --file - --name-only --list"] = effecttest.Result{Out: []byte("user.name\n")}
	if status, stderr := run(); status != 1 {
		t.Errorf("none: status = %d, stderr = %q; want 1", status, stderr)
	}
}
