package cli

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const (
	rootCld        = "/root/.local/bin/cld"
	rootCompletion = "/root/.local/share/bash-completion/completions/cld"
	// readAsUser is the line by which root's file is read as root's own account, as asuser.Read
	// reads it.
	readAsUser = `env HOME=/root sh -c if [ -e "$1" ]; then printf y; cat "$1"; else printf n; fi sh `
)

// claudeProcess is a process as root with cld installed for root, whose completion script
// holds have.
func claudeProcess(values, have string) (*effecttest.Machine, func(args ...string) (int, string, string)) {
	p, m := valuesProcess(values)
	m.Files[rootCld] = nil
	m.Results["env HOME=/root "+rootCld+" completion bash"] = effecttest.Result{Out: []byte("complete -F _cld cld\n")}
	m.Results[readAsUser+rootCompletion] = effecttest.Result{Out: []byte("y" + have)}
	m.Results["env HOME=/root "+rootCld+" setup completion bash"] = effecttest.Result{}
	return m, func(args ...string) (int, string, string) {
		return runWs(p, "", append([]string{"server", "claude"}, args...)...)
	}
}

func TestClaudeCheckExitsOneWhereAChangeIsDue(t *testing.T) {
	m, ws := claudeProcess("---\n", "old\n")
	status, stdout, stderr := ws("completion", "--check")
	if status != 1 || stdout != "" || !strings.Contains(stderr, "to do: set up cld's completion in bash") {
		t.Errorf("status = %d, stdout %q, stderr %q", status, stdout, stderr)
	}
	for _, c := range m.Calls {
		if strings.Contains(c, "setup completion") {
			t.Errorf("--check ran %q", c)
		}
	}
	if status, _, stderr := ws("completion"); status != 0 || !strings.Contains(stderr, "set up cld's completion in bash") {
		t.Errorf("completion: status = %d, stderr %q", status, stderr)
	}
}

func TestClaudeCheckExitsZeroWhenDone(t *testing.T) {
	_, ws := claudeProcess("---\n", "complete -F _cld cld\n")
	if status, stdout, stderr := ws("completion", "--check"); status != 0 || stdout != "" || stderr != "" {
		t.Errorf("status = %d, stdout %q, stderr %q", status, stdout, stderr)
	}
}

func TestClaudeMarketplacesOfAnEmptyListAskClaudeNothing(t *testing.T) {
	m, ws := claudeProcess("---\n", "")
	m.Files["/root/.local/bin/claude"] = nil
	if status, _, stderr := ws("marketplaces"); status != 0 || stderr != "" {
		t.Errorf("status = %d, stderr %q", status, stderr)
	}
	for _, c := range m.Calls {
		if strings.Contains(c, "marketplace") {
			t.Errorf("an empty list ran %q", c)
		}
	}
}

func TestClaudeMarketplacesCheckNamesEachToAdd(t *testing.T) {
	_, ws := claudeProcess("---\nWS_CLAUDE_MARKETPLACES:\n  - name: acme\n    source: acme/plugins\n", "")
	status, _, stderr := ws("marketplaces", "--check")
	if status != 1 || !strings.Contains(stderr, "add the plugin marketplace acme from acme/plugins") {
		t.Errorf("status = %d, stderr %q", status, stderr)
	}
}

func TestClaudeRestoreSkipsWithoutSystemd(t *testing.T) {
	m, ws := claudeProcess("---\n", "")
	status, _, stderr := ws("restore", "--check")
	if status != 0 || !strings.Contains(stderr, "Claude restore: skipped, systemd is not running") {
		t.Errorf("status = %d, stderr %q", status, stderr)
	}
	for _, c := range m.Calls {
		if strings.Contains(c, "setup restore") || strings.Contains(c, "systemctl") {
			t.Errorf("restore ran %q without systemd", c)
		}
	}
}

func TestClaudeCommandsReadTheValues(t *testing.T) {
	p, _ := valuesProcess("---\nWS_NO_SUCH_KEY: 1\n")
	for _, name := range []string{"update", "completion", "settings", "telemetry", "marketplaces", "restore"} {
		if status, _, stderr := runWs(p, "", "server", "claude", name, "--check"); status != 1 ||
			!strings.Contains(stderr, "WS_NO_SUCH_KEY") {
			t.Errorf("%s: status = %d, stderr %q", name, status, stderr)
		}
	}
}
