package cli

import (
	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/values"
)

// fakeProcess returns an Env that touches nothing of the test's own.
func fakeProcess() enginecli.Env { return clitest.Env() }

// withEnv returns p with the environment env, for Getenv, Lookup and Setenv.
func withEnv(p enginecli.Env, env map[string]string) enginecli.Env { return clitest.WithEnv(p, env) }

// runWs runs args on the root of ws client, and returns the status, stdout and stderr.
func runWs(p enginecli.Env, args ...string) (status int, stdout, stderr string) {
	return clitest.Run(p, []*cobra.Command{Command(p)}, "", args...)
}

// keyed gives the Mac a values.yml that names the file key it has for ws's login, as ssh's step needs.
func keyed(m *effecttest.Machine) {
	writeConfs(m, values.Text("WS_IDENTITY_PUB_KEY", "~/.ssh/id_ed25519.pub"), values.Text("WS_USER", "ada"))
	m.Files["/h/.ssh/id_ed25519.pub"] = []byte("ssh-ed25519 AAAA ada\n")
	m.Results["ssh-keygen -l -f /h/.ssh/id_ed25519.pub"] = effecttest.Result{}
}

// writeConfs puts the Mac's values.yml and the server's file at their paths in the home.
func writeConfs(m *effecttest.Machine, mac, server string) {
	m.Files["/etc/ws/values.yml"] = []byte(mac)
	m.Files["/etc/ws/servers/ws/values.yml"] = []byte(server)
	m.Dirs["/etc/ws/servers"], m.Dirs["/etc/ws/servers/ws"] = 0o755, 0o755
}
