package secrets

import (
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestWriteFileRoundTrips(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	m := effecttest.NewMachine()
	//nolint:gosec // G101: made-up values, for a round trip
	in := Secrets{Env: map[string]string{"MY_TOKEN": "${token.github}"}, Token: Tokens{GitHub: "dummy"}}
	if err := WriteFile(m, "/h/.config/ws/secrets.age", in, id); err != nil {
		t.Fatal(err)
	}
	if m.Modes["/h/.config/ws/secrets.age"] != 0o600 || m.Dirs["/h/.config/ws"] != 0o700 {
		t.Error("want the file 0600 and its directory 0700")
	}
	plain, err := Decrypt(m.Files["/h/.config/ws/secrets.age"], id)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(plain)
	if err != nil || got.Env["MY_TOKEN"] != "${token.github}" || got.Token.GitHub != "dummy" {
		t.Errorf("Decode = %+v, %v; want the input with its reference", got, err)
	}
}

func TestWriteFileRefusesWhatDecodeRefuses(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	m := effecttest.NewMachine()
	err := WriteFile(m, "/h/.config/ws/secrets.age", Secrets{Env: map[string]string{"A-B": "dummy"}}, id)
	if err == nil || strings.Contains(err.Error(), "dummy") {
		t.Errorf("err = %v, want a refusal with no value", err)
	}
	if len(m.Files) != 0 {
		t.Error("a refused file was written")
	}
}

func TestWriteFileRefusesWhatExpandRefuses(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	for name, env := range map[string]string{
		"missing": "pa${ss.wordpart}rd",
		"stray":   "a$b",
		"absent":  "${env.MISSING}",
		"loop":    "${env.A}",
	} {
		m := effecttest.NewMachine()
		err := WriteFile(m, "/h/.config/ws/secrets.age", Secrets{Env: map[string]string{"A": env}}, id)
		if err == nil || strings.Contains(err.Error(), "wordpart") || strings.Contains(err.Error(), "MISSING") {
			t.Errorf("%s: err = %v, want a refusal that quotes no part of the value", name, err)
		}
		if len(m.Files) != 0 {
			t.Errorf("%s: a refused file was written", name)
		}
	}
}

func TestWriteFileTightensItsDirectory(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	m := effecttest.NewMachine()
	if err := m.Mkdir("/h/.config/ws", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(m, "/h/.config/ws/secrets.age", Secrets{}, id); err != nil {
		t.Fatal(err)
	}
	if m.Dirs["/h/.config/ws"] != 0o700 {
		t.Errorf("the directory's mode = %o, want 700", m.Dirs["/h/.config/ws"])
	}
}
