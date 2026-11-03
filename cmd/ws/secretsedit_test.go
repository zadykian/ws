package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/secrets"
)

// editFixture is a Keychain with an identity and a file of its own TOML.
func editFixture(t *testing.T, toml string) (*effecttest.Machine, *age.X25519Identity) {
	t.Helper()
	id, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	m := effecttest.NewMachine()
	m.Results[kcFind+" -w"] = effecttest.Result{Out: []byte(id.String() + "\n")}
	if toml != "" {
		armored, err := secrets.Encrypt([]byte(toml), id.Recipient())
		if err != nil {
			t.Fatal(err)
		}
		m.Files[file] = armored
	}
	return m, id
}

func runSecrets(m *effecttest.Machine, stdin string, args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	status := secretsCommand(context.Background(), args, m, env, strings.NewReader(stdin), &out, &errb)
	return status, out.String(), errb.String()
}

func TestSecretsKeysAndUnset(t *testing.T) {
	m, id := editFixture(t, "[token]\ngithub = \"dummy\"\n[env]\nB = \"dummy\"\nA = \"${token.github}\"\n")
	status, out, _ := runSecrets(m, "", "keys")
	if status != 0 || out != "env A\nenv B\ntoken github\n" {
		t.Errorf("keys = %d %q, want the entries sorted", status, out)
	}
	if strings.Contains(out, "dummy") {
		t.Error("keys printed a value")
	}
	if status, _, errs := runSecrets(m, "", "unset", "env", "B"); status != 0 {
		t.Fatalf("unset: status %d, stderr %q", status, errs)
	}
	s, err := secrets.LoadRaw(m, file, id)
	if err != nil || len(s.Env) != 1 || s.Env["A"] != "${token.github}" || m.Modes[file] != 0o600 {
		t.Errorf("after unset: %+v, %v, mode %o; want A with its reference, 600", s, err, m.Modes[file])
	}
	status, _, errs := runSecrets(m, "", "unset", "env", "B")
	if status != 1 || !strings.Contains(errs, "env B is not in the file") {
		t.Errorf("unset again: status %d, stderr %q", status, errs)
	}
}

func TestSecretsKeysOfNoFile(t *testing.T) {
	m, _ := editFixture(t, "")
	if status, out, _ := runSecrets(m, "", "keys"); status != 0 || out != "" {
		t.Errorf("keys = %d %q, want none", status, out)
	}
}

func TestSecretsUsageOfTheForms(t *testing.T) {
	for _, a := range [][]string{{"unset", "env"}, {"keys", "x"}} {
		m, _ := editFixture(t, "")
		if status, _, errs := runSecrets(m, "", a...); status != 2 || !strings.Contains(errs, "usage: ws mac secrets COMMAND") {
			t.Errorf("%q: status %d, stderr %q; want 2 and the usage", a, status, errs)
		}
	}
}

func TestSecretsWithoutTheItem(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[kcFind+" -w"] = effecttest.Result{Err: errors.New(kcGone)}
	for _, a := range [][]string{{"unset", "env", "A"}, {"keys"}} {
		if status, _, errs := runSecrets(m, "", a...); status != 1 || !strings.Contains(errs, "ws mac secrets init") {
			t.Errorf("%q: status %d, stderr %q; want 1 naming init", a, status, errs)
		}
	}
}
