package main

import (
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/secrets"
)

const credDecrypt = "systemd-creds decrypt --name=ws-secrets " + secrets.CredstorePath + " -"

const credEncrypt = "systemd-creds encrypt --with-key=host --name=ws-secrets - -" //nolint:gosec // G101: a command line, no credential

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
	s, err := secrets.Load(m, secrets.FilePath, id)
	if err != nil || s.Env["A"] != "dummy-value" || len(s.Env) != 1 || string(s.File.VPNSswan) != "dummy-value\n" {
		t.Errorf("secrets = %+v, %v; want env A and the file", s, err)
	}
	status, out, _ := runServerSecrets(m, 0, "", "keys")
	if status != 0 || out != "env A\nfile vpn_sswan\n" {
		t.Errorf("keys = %d %q", status, out)
	}
	if len(m.Inputs) != 0 {
		t.Error("a command took a value")
	}
}

func TestServerSecretsInitMakesBoth(t *testing.T) {
	m, id := credFixture()
	delete(m.Files, secrets.CredstorePath)
	m.Results[credEncrypt] = effecttest.Result{Out: []byte("sealed")}
	status, out, errs := runServerSecrets(m, 0, "", "init")
	if status != 0 || len(m.Inputs) != 1 || string(m.Files[secrets.CredstorePath]) != "sealed" {
		t.Fatalf("status %d, stderr %q; want the credential made", status, errs)
	}
	// The fake's credential gives id back, as systemd-creds gives the identity that it took.
	if _, err := secrets.Load(m, secrets.FilePath, id); err != nil || m.Modes[secrets.FilePath] != 0o600 {
		t.Errorf("Load = %v, mode %o; want an empty file that the identity opens, 600", err, m.Modes[secrets.FilePath])
	}
	if strings.Count(out, ": made ") != 2 || strings.Contains(out+errs, "AGE-SECRET-KEY") {
		t.Errorf("stdout %q; want two lines of what init made, and no key", out)
	}
}

func TestServerSecretsInitKeepsWhatIsThere(t *testing.T) {
	m, id := credFixture()
	if status, _, errs := runServerSecrets(m, 0, "", "init"); status != 0 {
		t.Fatalf("status %d, stderr %q", status, errs)
	}
	if _, err := secrets.Load(m, secrets.FilePath, id); err != nil {
		t.Errorf("Load = %v; want the file made with the credential's identity", err)
	}
	m.Files[secrets.FilePath] = []byte("kept")
	status, out, _ := runServerSecrets(m, 0, "", "init")
	if status != 0 || string(m.Files[secrets.FilePath]) != "kept" || strings.Count(out, "stays") != 2 || len(m.Inputs) != 0 {
		t.Errorf("status %d, stdout %q; want both kept", status, out)
	}
}

func TestServerSecretsAsRootAlone(t *testing.T) {
	m, _ := credFixture()
	for _, a := range [][]string{{"init"}, {"set", "env", "A"}, {"unset", "env", "A"}, {"keys"}} {
		status, _, errs := runServerSecrets(m, 1000, "dummy\n", a...)
		if status != 1 || !strings.Contains(errs, "run it as root") {
			t.Errorf("%q: status %d, stderr %q; want 1, as root", a, status, errs)
		}
	}
	if len(m.Calls) != 0 {
		t.Errorf("calls %q, want none", m.Calls)
	}
}

func TestServerSecretsWithoutTheCredential(t *testing.T) {
	status, _, errs := runServerSecrets(effecttest.NewMachine(), 0, "dummy\n", "set", "env", "A")
	if status != 1 || !strings.Contains(errs, "ws server secrets init") {
		t.Errorf("status %d, stderr %q; want 1 naming init", status, errs)
	}
}
