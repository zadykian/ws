package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const (
	rootCld        = "/root/.local/bin/cld"
	rootCompletion = "/root/.local/share/bash-completion/completions/cld"
)

// claudeProcess is a process as root with cld installed for root, whose completion script
// holds have. Its units are a fake that has root's user manager running.
func claudeProcess(values, have string) (*effecttest.Machine, func(args ...string) (int, string, string)) {
	m, _, ws := claudeUnits(values, have)
	return m, ws
}

// claudeUnits is claudeProcess with its fake units returned, as the claude commands plan them
// there and never over D-Bus.
func claudeUnits(values, have string) (*effecttest.Machine, *effecttest.Units, func(args ...string) (int, string, string)) {
	p, m := valuesProcess(values)
	u := &effecttest.Units{States: map[string]effect.UnitState{"user@0.service": {Active: true}}}
	m.Files[rootCld] = nil
	m.Results["env -i -C /root HOME=/root PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin "+rootCld+" completion bash"] = effecttest.Result{Out: []byte("complete -F _cld cld\n")}
	m.Files[rootCompletion] = []byte(have)
	m.Results["env -i -C /root HOME=/root PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin "+rootCld+" setup completion bash"] = effecttest.Result{}
	return m, u, func(args ...string) (int, string, string) {
		root := enginecli.Group("ubuntu", "Ubuntu", enginecli.Group("guest", "the guest", newClaudeCmd(p, u)))
		return clitest.Run(p, []*cobra.Command{root}, "", append([]string{"ubuntu", "guest", "claude"}, args...)...)
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

func TestClaudeCommandsReadTheValues(t *testing.T) {
	p, _ := valuesProcess("---\nWS_NO_SUCH_KEY: 1\n")
	for _, name := range []string{"update", "completion", "settings", "telemetry", "marketplaces", "restore"} {
		if status, _, stderr := runWs(p, "", "ubuntu", "guest", "claude", name, "--check"); status != 1 ||
			!strings.Contains(stderr, "WS_NO_SUCH_KEY") {
			t.Errorf("%s: status = %d, stderr %q", name, status, stderr)
		}
	}
}
