package secrets

import (
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestLoadDecryptsAndParses(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	m := effecttest.NewMachine()
	m.Files["/s/secrets.age"] = encrypt(t, "[env]\nA = \"b\"\n[file]\nvpn_p12 = \"AP9wMTI=\"\n", id.Recipient())
	s, err := Load(m, "/s/secrets.age", id)
	if err != nil {
		t.Fatal(err)
	}
	if s.Env["A"] != "b" || string(s.File.VPNP12) != "\x00\xffp12" {
		t.Errorf("got %+v", s)
	}
}

func TestLoadErrorsNameAFileNeverAValue(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	other, _ := age.GenerateX25519Identity()
	m := effecttest.NewMachine()
	m.Files["/s/secrets.age"] = encrypt(t, "[env]\nA = \"hunter2\"\n", id.Recipient())
	if _, err := Load(m, "/s/secrets.age", other); err == nil || !strings.Contains(err.Error(), "/s/secrets.age") {
		t.Errorf("wrong identity: %v", err)
	}
	m.Files["/s/secrets.age"] = encrypt(t, "[env]\nA = \"hunter2\"\nbogus = 1\n[bogus]\n", id.Recipient())
	if _, err := Load(m, "/s/secrets.age", id); err == nil || strings.Contains(err.Error(), "hunter2") ||
		!strings.Contains(err.Error(), "/s/secrets.age") {
		t.Errorf("bad file: %v", err)
	}
}

func TestLoadRawKeepsReferences(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	armored, err := Encrypt([]byte("[token]\ngithub = \"dummy\"\n[env]\nMY_TOKEN = \"${token.github}\"\n"), id.Recipient())
	if err != nil {
		t.Fatal(err)
	}
	m := effecttest.NewMachine()
	m.Files["/f"] = armored
	raw, err := LoadRaw(m, "/f", id)
	if err != nil || raw.Env["MY_TOKEN"] != "${token.github}" {
		t.Errorf("LoadRaw env = %q, %v; want the reference", raw.Env["MY_TOKEN"], err)
	}
	full, err := Load(m, "/f", id)
	if err != nil || full.Env["MY_TOKEN"] != "dummy" {
		t.Errorf("Load env = %q, %v; want the value", full.Env["MY_TOKEN"], err)
	}
}
