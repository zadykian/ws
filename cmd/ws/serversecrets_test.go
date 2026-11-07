package main

import (
	"testing"

	"filippo.io/age"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/secrets"
)

const credDecrypt = "systemd-creds decrypt --name=ws-secrets " + secrets.CredstorePath + " -"

// credFixture is ws with its credential, whose identity it returns.
func credFixture() (*effecttest.Machine, *age.X25519Identity) {
	id, _ := age.GenerateX25519Identity()
	m := effecttest.NewMachine()
	m.Files[secrets.CredstorePath] = []byte("sealed")
	m.Results[credDecrypt] = effecttest.Result{Out: []byte(id.String() + "\n")}
	return m, id
}

func runServerSecrets(m *effecttest.Machine, euid int, stdin string, args ...string) (int, string, string) {
	p := fakeProcess()
	p.machine, p.euid = m, euid
	return runWs(p, stdin, append([]string{"server", "secrets"}, args...)...)
}

func TestServerSecretsEdits(t *testing.T) {
	m, id := credFixture()
	for _, a := range [][]string{{"set", "env", "A"}, {"set", "env", "B"}, {"set", "file", "vpn_sswan"}, {"unset", "env", "B"}} {
		if status, _, errs := runServerSecrets(m, 0, "dummy-value\n", a...); status != 0 {
			t.Fatalf("%q: status %d, stderr %q", a, status, errs)
		}
	}
	s, err := secrets.Load(m, serverSecrets, id)
	if err != nil || s.Env["A"] != "dummy-value" || len(s.Env) != 1 || string(s.File.VPNSswan) != "dummy-value\n" {
		t.Errorf("secrets = %+v, %v; want env A and the file", s, err)
	}
	if m.Modes[serverSecrets] != 0o600 || m.Owners[serverSecrets] != [2]int{} || m.Dirs["/etc/ws"] != 0o755 {
		t.Errorf("file %o, owner %v, directory %o; want root's 600 in 755", m.Modes[serverSecrets],
			m.Owners[serverSecrets], m.Dirs["/etc/ws"])
	}
	status, out, _ := runServerSecrets(m, 0, "", "keys")
	if status != 0 || out != "env A\nfile vpn_sswan\n" {
		t.Errorf("keys = %d %q", status, out)
	}
	if len(m.Inputs) != 0 {
		t.Error("a command took a value")
	}
}
