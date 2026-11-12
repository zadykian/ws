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

func TestSealRoundTrips(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	//nolint:gosec // G101: made-up values, for a round trip
	in := Secrets{Env: map[string]string{"MY_TOKEN": "${token.github}"}, Token: Tokens{GitHub: "dummy"}}
	armored, err := Seal(in, id)
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
	_, err := Seal(Secrets{Env: map[string]string{"A-B": "dummy"}}, id)
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
		_, err := Seal(Secrets{Env: map[string]string{"A": env}}, id)
		if err == nil || strings.Contains(err.Error(), "wordpart") || strings.Contains(err.Error(), "MISSING") {
			t.Errorf("%s: err = %v, want a refusal that quotes no part of the value", name, err)
		}
	}
}
