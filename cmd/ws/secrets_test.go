package main

import (
	"errors"
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/secrets"
)

const (
	kcFind = "security find-generic-password -s ws-secrets"
	kcGone = "The specified item could not be found in the keychain."
	home   = "/Users/me"
	file   = home + "/.config/ws/secrets.age"
)

func runMacSecrets(m *effecttest.Machine, args ...string) (int, string, string) {
	p := withEnv(fakeProcess(), map[string]string{"HOME": home, "USER": "me"})
	p.machine = m
	return runWs(p, "", append([]string{"mac", "secrets"}, args...)...)
}

func TestSecretsInitMakesTheItemAndTheFile(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	m := effecttest.NewMachine()
	m.Results[kcFind] = effecttest.Result{Err: errors.New(kcGone)}
	m.Results["security -i"] = effecttest.Result{}
	m.Results[kcFind+" -w"] = effecttest.Result{Out: []byte(id.String() + "\n")}
	got, out, errb := runMacSecrets(m, "init")
	if got != 0 {
		t.Fatalf("status = %d, stderr %q", got, errb)
	}
	if m.Modes[file] != 0o600 || m.Dirs[home+"/.config/ws"] != 0o700 {
		t.Errorf("file mode %o, directory mode %o; want 600 and 700", m.Modes[file], m.Dirs[home+"/.config/ws"])
	}
	s, err := secrets.Load(m, file, id)
	if err != nil || len(s.Env) != 0 {
		t.Errorf("Load = %v, %v; want empty secrets", s, err)
	}
	if strings.Contains(out+errb, "AGE-SECRET-KEY") {
		t.Error("the output holds a key")
	}
}

func TestSecretsInitKeepsWhatIsThere(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[kcFind] = effecttest.Result{}
	m.Files[file] = []byte("kept")
	got, out, errb := runMacSecrets(m, "init")
	if got != 0 {
		t.Fatalf("status = %d, stderr %q", got, errb)
	}
	if string(m.Files[file]) != "kept" || len(m.Inputs) != 0 {
		t.Error("init changed the file or the item")
	}
	if !strings.Contains(out, "stays") {
		t.Errorf("stdout = %q, want it to say what stays", out)
	}
}
