package cli

import (
	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/values"
)

// fakeProcess returns an Env that touches nothing of the test's own.
func fakeProcess() enginecli.Env { return clitest.Env() }

// withEnv returns p with the environment env, for Getenv, Lookup and Setenv.
func withEnv(p enginecli.Env, env map[string]string) enginecli.Env { return clitest.WithEnv(p, env) }

// runWs runs args on the root of ws macos, and returns the status, stdout and stderr.
func runWs(p enginecli.Env, args ...string) (status int, stdout, stderr string) {
	return clitest.Run(p, []*cobra.Command{Command(p)}, "", args...)
}

// oneServer gives the Mac the file of its one server, ws.
func oneServer(m *effecttest.Machine) {
	writeConfs(m, values.Text("WS_SLOT", "0"))
}

// writeConfs puts the server ws's file at its path in /etc/ws.
func writeConfs(m *effecttest.Machine, server string) {
	m.Files["/etc/ws/servers/ws/values.yml"] = []byte(server)
	m.Dirs["/etc/ws/servers"], m.Dirs["/etc/ws/servers/ws"] = 0o755, 0o755
}
