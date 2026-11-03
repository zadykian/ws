package main

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"filippo.io/age"
	"filippo.io/age/armor"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestServerRefusal(t *testing.T) {
	ok := []byte("NAME=\"Ubuntu\"\nID=ubuntu\nVERSION_ID=\"26.04\"\n")
	for _, tc := range []struct {
		name string
		euid int
		text string
		want string
	}{
		{"ubuntu 26.04 as root", 0, string(ok), ""},
		{"not root", 1000, string(ok), "run it as root"},
		{"older ubuntu", 0, "ID=ubuntu\nVERSION_ID=\"24.04\"\n", "Ubuntu 26.04 is the one system for this, not ubuntu 24.04"},
		{"debian", 0, "ID=debian\nVERSION_ID=\"13\"\n", "Ubuntu 26.04 is the one system for this, not debian 13"},
		{"no file", 0, "", "Ubuntu 26.04 is the one system for this, not  "},
		{"quoted id", 0, "ID=\"ubuntu\"\nVERSION_ID=26.04\n", ""},
	} {
		if got := serverRefusal(tc.euid, []byte(tc.text)); got != tc.want {
			t.Errorf("%s: serverRefusal = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestServerApplyRefusesOtherArgs(t *testing.T) {
	for _, args := range [][]string{
		{"apply", "--tags", "base"}, {"apply", "--only", "nothing"}, {"apply", "--only", "base,"}, {"apply", "--only", ""},
		{"apply", "--only"}, {"apply", "-C"}, {"apply", "extra"}, {"other"},
	} {
		status, _, stderr := runWs(fakeProcess(), "", append([]string{"server"}, args...)...)
		if status != 2 || !strings.Contains(stderr, "--help' for usage.") {
			t.Errorf("%q: status = %d, stderr = %q", args, status, stderr)
		}
	}
}

func TestServerApplyRefusesNonRoot(t *testing.T) {
	p := fakeProcess()
	m := p.machine.(*effecttest.Machine)
	p.euid = 1000
	m.Files["/etc/os-release"] = []byte("ID=ubuntu\nVERSION_ID=\"26.04\"\n")
	status, _, stderr := runWs(p, "", "server", "apply")
	if status != 1 || stderr != "ws server apply: run it as root\n" {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "exec ") }) {
		t.Errorf("calls = %q, want no exec", m.Calls)
	}
}

func TestServerApplyHelp(t *testing.T) {
	status, stdout, _ := runWs(fakeProcess(), "", "server", "apply", "-h")
	for _, want := range []string{"--check", "--only", "--secrets", "ws-secrets"} {
		if status != 0 || !strings.Contains(stdout, want) {
			t.Errorf("status = %d, help = %q, want it to hold %q", status, stdout, want)
		}
	}
}

func TestLoadSecrets(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	m := effecttest.NewMachine()
	var buf bytes.Buffer
	a := armor.NewWriter(&buf)
	w, _ := age.Encrypt(a, id.Recipient())
	_, _ = w.Write([]byte("[env]\nA = \"hunter2\"\n"))
	_ = w.Close()
	_ = a.Close()
	m.Files["/s.age"] = buf.Bytes()
	env := func(k string) string {
		if k == "CREDENTIALS_DIRECTORY" {
			return "/creds"
		}
		return ""
	}
	m.Files["/creds/ws-secrets"] = []byte("# created: now\n" + id.String() + "\n")
	s, err := loadSecrets(m, "/s.age", env)
	if err != nil || s.Env["A"] != "hunter2" {
		t.Errorf("got %+v, %v", s, err)
	}
	if s, err = loadSecrets(m, "", noEnv); err != nil || s.Env != nil {
		t.Errorf("no file: %+v, %v", s, err)
	}
	other, _ := age.GenerateX25519Identity()
	m.Files["/creds/ws-secrets"] = []byte(other.String())
	if _, err = loadSecrets(m, "/s.age", env); err == nil || strings.Contains(err.Error(), "hunter2") {
		t.Errorf("wrong identity: %v", err)
	}
	if _, err = loadSecrets(m, "/s.age", noEnv); err == nil || !strings.Contains(err.Error(), "LoadCredential") {
		t.Errorf("no credential: %v", err)
	}
	delete(m.Files, "/creds/ws-secrets")
	if _, err = loadSecrets(m, "/s.age", env); err == nil {
		t.Error("no credential file: no error")
	}
}

func TestServerFansRefusesAConfigThatFailsItsCheck(t *testing.T) {
	p := fakeProcess()
	p.machine.(*effecttest.Machine).Files["/ws-fans.json"] = []byte(`{"interval": 0}`)
	status, _, stderr := runWs(p, "", "server", "fans", "/ws-fans.json")
	if status != 1 || !strings.HasPrefix(stderr, "ws server fans: /ws-fans.json: ") || !strings.Contains(stderr, "interval") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	if status, _, _ = runWs(p, "", "server", "fans", "a", "b"); status != 2 {
		t.Errorf("two configs: status = %d, want 2", status)
	}
	if status, _, stderr = runWs(p, "", "server", "fans"); status != 1 || !strings.Contains(stderr, "/etc/ws-fans.json") {
		t.Errorf("no config: status = %d, stderr = %q", status, stderr)
	}
}

func noEnv(string) string { return "" }
