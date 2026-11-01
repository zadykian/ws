package secrets

import (
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestLoadDecryptsTheFilesItNames(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	m := effecttest.NewMachine()
	m.Files["/s/secrets.age"] = encrypt(t, "shell_secret_env: {A: b}\nikeward_p12_file: p.age\n"+
		"ikeward_sswan_file: /o/s.age\nikeward_p12_password: pw\n", id.Recipient())
	m.Files["/s/p.age"] = encrypt(t, "\x00\xffp12", id.Recipient())
	m.Files["/o/s.age"] = encrypt(t, "sswan", id.Recipient())
	s, err := Load(m, "/s/secrets.age", id)
	if err != nil {
		t.Fatal(err)
	}
	if string(s.Files.P12) != "\x00\xffp12" || string(s.Files.Sswan) != "sswan" {
		t.Errorf("files %q, %q", s.Files.P12, s.Files.Sswan)
	}
}

func TestLoadErrorsNameAFileNeverAValue(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	other, _ := age.GenerateX25519Identity()
	m := effecttest.NewMachine()
	m.Files["/s/secrets.age"] = encrypt(t, "shell_secret_env: {A: hunter2}\nikeward_p12_file: gone.age\n", id.Recipient())
	for name, who := range map[string]age.Identity{"missing profile file": id, "wrong identity": other} {
		_, err := Load(m, "/s/secrets.age", who)
		if err == nil || !strings.Contains(err.Error(), "/s/") || strings.Contains(err.Error(), "hunter2") {
			t.Errorf("%s: %v", name, err)
		}
	}
	m.Files["/s/secrets.age"] = encrypt(t, "shell_secret_env: {A: hunter2}\nbogus: 1\n", id.Recipient())
	if _, err := Load(m, "/s/secrets.age", id); err == nil || strings.Contains(err.Error(), "hunter2") {
		t.Errorf("unknown key: %v", err)
	}
}
