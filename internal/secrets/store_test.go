package secrets

import (
	"strings"
	"testing"

	"filippo.io/age"
)

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
