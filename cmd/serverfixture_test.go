package cmd

import (
	"context"
	"testing"

	"filippo.io/age"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
)

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

const (
	// enterScreen and leaveScreen are how a terminal's alternate screen starts and ends.
	enterScreen = "\x1b[?1049h"
	leaveScreen = "\x1b[?1049l"
)

// serverTerminal returns a process with a terminal, as root on the one system that ws server ubuntu
// apply takes, with ca-certificates and tmux installed.
func serverTerminal() (cli.Env, *effecttest.Machine) {
	p := withEnv(fakeProcess(), map[string]string{"TERM": "xterm"})
	p.TTY, p.ErrTTY, p.Euid = true, true, 0
	m := p.Machine.(*effecttest.Machine)
	m.Files["/etc/os-release"] = []byte("ID=ubuntu\nVERSION_ID=\"26.04\"\n")
	m.Packages = effecttest.Packages{"ca-certificates": {Version: "1", Status: "ii"}, "tmux": {Version: "3.5", Status: "ii"}}
	return p, m
}

// change is what a stepOf plans.
type change string

func (c change) String() string { return string(c) }

// stepOf is a step whose Plan runs plan and finds a change, so that --check names its role.
type stepOf struct {
	name string
	plan func(ctx context.Context) error
}

func (s stepOf) Name() string    { return s.name }
func (s stepOf) WaitsOn() string { return "" }
func (s stepOf) Plan(ctx context.Context) (step.Plan, error) {
	if err := s.plan(ctx); err != nil {
		return step.Plan{}, err
	}
	return step.Plan{Changes: []step.Change{change("x")}}, nil
}
func (stepOf) Apply(context.Context, step.Plan) error { return nil }
