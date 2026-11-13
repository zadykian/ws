package cmd

import (
	"testing"

	"filippo.io/age"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// wizardProcess returns a root process on a machine that holds vars as host_vars, booted or not,
// whose remote script reports status as host_vars' ws_user reads it.
func wizardProcess(vars string, booted bool, status string) (cli.Env, *effecttest.Machine) {
	p, m := valuesProcess(vars)
	p.GOOS = "linux"
	if booted {
		m.Dirs["/run/systemd/system"] = 0o755
	}
	v, _ := values.Load(m, serverVars)
	m.Results["env SUDO_USER="+v.User+" sh -s -- status"] = effecttest.Result{Out: []byte(status)}
	return p, m
}

// valuesProcess returns a root process on a machine that holds file as the values.
func valuesProcess(file string) (cli.Env, *effecttest.Machine) {
	p := fakeProcess()
	m := effecttest.NewMachine()
	p.Machine, p.Euid = m, 0
	m.Dirs["/etc/ws"] = 0o755
	m.Files[serverVars] = []byte(file)
	return p, m
}

// runServerSecrets runs ws server ubuntu secrets as euid on m.
func runServerSecrets(m *effecttest.Machine, euid int, stdin string, args ...string) (int, string, string) {
	p := fakeProcess()
	p.Machine, p.Euid = m, euid
	return runWs(p, stdin, append([]string{"server", "ubuntu", "secrets"}, args...)...)
}

const credDecrypt = "systemd-creds decrypt --name=ws-secrets " + secrets.CredstorePath + " -"

// credFixture is ws with its credential, whose identity it returns.
func credFixture() (*effecttest.Machine, *age.X25519Identity) {
	id, _ := age.GenerateX25519Identity()
	m := effecttest.NewMachine()
	m.Files[secrets.CredstorePath] = []byte("sealed")
	m.Results[credDecrypt] = effecttest.Result{Out: []byte(id.String() + "\n")}
	return m, id
}

// editFixture is ws's credential with an identity, and a file of its own TOML.
func editFixture(t *testing.T, toml string) (*effecttest.Machine, *age.X25519Identity) {
	m, id := credFixture()
	if toml != "" {
		armored, err := secrets.Encrypt([]byte(toml), id.Recipient())
		if err != nil {
			t.Fatal(err)
		}
		m.Files[secrets.FilePath] = armored
	}
	return m, id
}
