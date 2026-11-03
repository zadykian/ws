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

const (
	kcFind = "security find-generic-password -s ws-secrets"
	kcGone = "The specified item could not be found in the keychain."
	home   = "/Users/me"
	file   = home + "/.config/ws/secrets.age"
)

func env(k string) string {
	return map[string]string{"HOME": home, "USER": "me"}[k]
}

func TestSecretsUsage(t *testing.T) {
	for _, args := range [][]string{nil, {"init", "x"}, {"nope"}} {
		var out, errb bytes.Buffer
		if got := secretsCommand(context.Background(), args, effecttest.NewMachine(), env, strings.NewReader(""), &out, &errb); got != 2 {
			t.Errorf("%q: status = %d, want 2", args, got)
		}
		if !strings.Contains(errb.String(), "usage: ws mac secrets COMMAND") {
			t.Errorf("%q: stderr = %q, want the usage", args, errb.String())
		}
	}
}

func TestSecretsInitMakesTheItemAndTheFile(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	m := effecttest.NewMachine()
	m.Results[kcFind] = effecttest.Result{Err: errors.New(kcGone)}
	m.Results["security -i"] = effecttest.Result{}
	m.Results[kcFind+" -w"] = effecttest.Result{Out: []byte(id.String() + "\n")}
	var out, errb bytes.Buffer
	if got := secretsCommand(context.Background(), []string{"init"}, m, env, strings.NewReader(""), &out, &errb); got != 0 {
		t.Fatalf("status = %d, stderr %q", got, errb.String())
	}
	if m.Modes[file] != 0o600 || m.Dirs[home+"/.config/ws"] != 0o700 {
		t.Errorf("file mode %o, directory mode %o; want 600 and 700", m.Modes[file], m.Dirs[home+"/.config/ws"])
	}
	s, err := secrets.Load(m, file, id)
	if err != nil || len(s.Env) != 0 {
		t.Errorf("Load = %v, %v; want empty secrets", s, err)
	}
	if strings.Contains(out.String()+errb.String(), "AGE-SECRET-KEY") {
		t.Error("the output holds a key")
	}
}

func TestSecretsInitKeepsWhatIsThere(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[kcFind] = effecttest.Result{}
	m.Files[file] = []byte("kept")
	var out, errb bytes.Buffer
	if got := secretsCommand(context.Background(), []string{"init"}, m, env, strings.NewReader(""), &out, &errb); got != 0 {
		t.Fatalf("status = %d, stderr %q", got, errb.String())
	}
	if string(m.Files[file]) != "kept" || len(m.Inputs) != 0 {
		t.Error("init changed the file or the item")
	}
	if !strings.Contains(out.String(), "stays") {
		t.Errorf("stdout = %q, want it to say what stays", out.String())
	}
}
