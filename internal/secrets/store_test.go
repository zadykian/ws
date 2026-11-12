package secrets

import (
	"context"
	"errors"
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/zadykian/ws/effect/effecttest"
)

// storeFixture is a machine with the credential, and the file where toml is not empty, sealed to
// the identity that it returns.
func storeFixture(t *testing.T, toml string) (*effecttest.Machine, age.Identity) {
	id, _ := age.GenerateX25519Identity()
	m := effecttest.NewMachine()
	m.Files[CredstorePath] = []byte("sealed")
	m.Results[credDecrypt] = effecttest.Result{Out: []byte(id.String() + "\n")}
	if toml != "" {
		m.Files[FilePath] = encrypt(t, toml, id.Recipient())
	}
	return m, id
}

func TestStoreEditKeepsReferences(t *testing.T) {
	m, id := storeFixture(t, "[token]\ngithub = \"dummy\"\n[env]\nA = \"${token.github}\"\n")
	err := Store{M: m}.Edit(context.Background(), func(s *Secrets) error { return s.Set(KindEnv, "B", "dummy-b") })
	if err != nil {
		t.Fatal(err)
	}
	plain, _ := Decrypt(m.Files[FilePath], id)
	got, err := Decode(plain)
	if err != nil || got.Env["A"] != "${token.github}" || got.Env["B"] != "dummy-b" {
		t.Errorf("Decode = %+v, %v; want A with its reference and B", got, err)
	}
}

func TestStoreEditWithoutTheCredential(t *testing.T) {
	err := Store{M: effecttest.NewMachine()}.Edit(context.Background(), func(*Secrets) error { return nil })
	if !errors.Is(err, ErrNoCredential) {
		t.Errorf("err = %v, want ErrNoCredential", err)
	}
}

func TestStoreRefusedEditWritesNothing(t *testing.T) {
	m, _ := storeFixture(t, "[env]\nA = \"dummy\"\n")
	before := string(m.Files[FilePath])
	stop := errors.New("stop")
	if err := (Store{M: m}).Edit(context.Background(), func(*Secrets) error { return stop }); !errors.Is(err, stop) {
		t.Errorf("err = %v, want fn's", err)
	}
	err := Store{M: m}.Edit(context.Background(), func(s *Secrets) error {
		s.Env["A-B"] = "dummy"
		return nil
	})
	if err == nil || string(m.Files[FilePath]) != before {
		t.Errorf("err = %v; want a refusal that left the file as it was", err)
	}
}

func TestSealRoundTrips(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	//nolint:gosec // G101: made-up values, for a round trip
	in := Secrets{Env: map[string]string{"MY_TOKEN": "${token.github}"}, Token: Tokens{GitHub: "dummy"}}
	armored, err := seal(in, id)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := Decrypt(armored, id)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(plain)
	if err != nil || got.Env["MY_TOKEN"] != "${token.github}" || got.Token.GitHub != "dummy" {
		t.Errorf("Decode = %+v, %v; want the input with its reference", got, err)
	}
}

func TestSealRefusesWhatDecodeRefuses(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	_, err := seal(Secrets{Env: map[string]string{"A-B": "dummy"}}, id)
	if err == nil || strings.Contains(err.Error(), "dummy") {
		t.Errorf("err = %v, want a refusal with no value", err)
	}
}

func TestSealRefusesWhatExpandRefuses(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	for name, env := range map[string]string{
		"missing": "pa${ss.wordpart}rd",
		"stray":   "a$b",
		"absent":  "${env.MISSING}",
		"loop":    "${env.A}",
	} {
		_, err := seal(Secrets{Env: map[string]string{"A": env}}, id)
		if err == nil || strings.Contains(err.Error(), "wordpart") || strings.Contains(err.Error(), "MISSING") {
			t.Errorf("%s: err = %v, want a refusal that quotes no part of the value", name, err)
		}
	}
}

func TestStoreWritesRootsFile(t *testing.T) {
	m, _ := storeFixture(t, "")
	if err := (Store{M: m}).Set(context.Background(), "env A", "dummy"); err != nil {
		t.Fatal(err)
	}
	if m.Modes[FilePath] != 0o600 || m.Owners[FilePath] != [2]int{} || m.Dirs["/etc/ws"] != 0o755 {
		t.Errorf("file %o, owner %v, directory %o; want root's 600 in 755", m.Modes[FilePath],
			m.Owners[FilePath], m.Dirs["/etc/ws"])
	}
	if _, plain := m.Files[FilePath+".plain"]; plain || len(m.Files) != 2 {
		t.Errorf("files %d, want the credential and the sealed file alone", len(m.Files))
	}
}

func TestStoreKeysOfNoFile(t *testing.T) {
	m, _ := storeFixture(t, "")
	keys, err := Store{M: m}.Keys(context.Background())
	if err != nil || len(keys) != 0 {
		t.Errorf("Keys = %v, %v; want none", keys, err)
	}
}

func TestStoreKeysNameNoValue(t *testing.T) {
	m, _ := storeFixture(t, "[token]\ngithub = \"dummy-t\"\n[env]\nB = \"dummy-b\"\n")
	keys, err := Store{M: m}.Keys(context.Background())
	if err != nil || strings.Join(keys, ",") != "env B,token github" {
		t.Errorf("Keys = %v, %v; want env B and token github", keys, err)
	}
}

func TestStoreSetKeepsAGitEntrysRest(t *testing.T) {
	m, id := storeFixture(t, "[git.\"https://h.example\"]\nusername = \"bob\"\ntoken = \"dummy-1\"\n")
	if err := (Store{M: m}).Set(context.Background(), "git https://h.example", "dummy-2"); err != nil {
		t.Fatal(err)
	}
	plain, _ := Decrypt(m.Files[FilePath], id)
	got, _ := Decode(plain)
	if g := got.Git["https://h.example"]; g.Username != "bob" || g.Token != "dummy-2" {
		t.Errorf("git = %+v; want bob and the new token", g)
	}
}

func TestStoreSetRefusesALabelWithNoSpace(t *testing.T) {
	m, _ := storeFixture(t, "")
	if err := (Store{M: m}).Set(context.Background(), "envA", "dummy"); err == nil || len(m.Files) != 1 {
		t.Errorf("err = %v, files %d; want a refusal and no file", err, len(m.Files))
	}
}
