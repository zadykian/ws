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
