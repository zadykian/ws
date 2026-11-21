package cli

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/ubuntu/devtools"
)

// In the secrets task's run, gh's login takes the token from WS_GITHUB_TOKEN, which mise writes to
// the file before it, and a plan writes nowhere. Elsewhere a stale variable counts for nothing, and
// the file's token stands.
func TestDevtoolsGHLoginTakesTheTokenInputInTheSecretsTask(t *testing.T) {
	gh := "env -i -C /root HOME=/root PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin env -u " + strings.Join(secretInputs, " -u ") + " gh"
	for i, c := range []struct {
		env    map[string]string
		status int
	}{
		{map[string]string{"MISE_TASK_NAME": "secrets", "WS_GITHUB_TOKEN": "dummy-b"}, 1},
		{map[string]string{"MISE_TASK_NAME": "secrets", "WS_GITHUB_TOKEN": "dummy-a"}, 0},
		{map[string]string{"MISE_TASK_NAME": "secrets", "WS_GITHUB_TOKEN": ""}, 0},
		{map[string]string{"WS_GITHUB_TOKEN": "dummy-b"}, 0},
	} {
		p, m := valuesProcess("---\n")
		p = clitest.WithEnv(p, c.env)
		m.Files[devtools.TokenFile] = []byte("# Written by github.com/zadykian/ws\ndummy-a\n")
		m.Results[gh+" auth token -h github.com"] = effecttest.Result{Out: []byte("dummy-a\n")}
		status, stdout, stderr := runWs(p, "", "server", "devtools", "gh-login", "--check")
		if status != c.status || strings.Contains(stdout+stderr, "dummy") {
			t.Errorf("case %d: status = %d, stdout %q, stderr %q; want %d", i, status, stdout, stderr, c.status)
		}
		if len(m.Inputs) != 0 {
			t.Errorf("case %d: --check logged gh in", i)
		}
	}
}

// The login gives gh the input's token on its input, and names no value in a command or a line.
func TestDevtoolsGHLoginLogsInByTheInput(t *testing.T) {
	gh := "env -i -C /root HOME=/root PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin env -u " + strings.Join(secretInputs, " -u ") + " gh"
	p, m := valuesProcess("---\n")
	p = clitest.WithEnv(p, map[string]string{"MISE_TASK_NAME": "secrets", "WS_GITHUB_TOKEN": "dummy-b"})
	m.Results[gh+" auth login -h github.com -p https --insecure-storage --with-token"] = effecttest.Result{}
	status, stdout, stderr := runWs(p, "", "server", "devtools", "gh-login")
	if status != 0 || strings.Contains(stdout+stderr, "dummy") || !slices.Equal(m.Inputs, []string{"dummy-b\n"}) {
		t.Errorf("status = %d, stdout %q, stderr %q, inputs %d", status, stdout, stderr, len(m.Inputs))
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, "dummy") }) {
		t.Errorf("a command's arguments hold the token: %q", m.Calls)
	}
}
