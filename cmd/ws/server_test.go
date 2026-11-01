package main

import (
	"bytes"
	"context"
	"io"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

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
		{"older ubuntu", 0, "ID=ubuntu\nVERSION_ID=\"24.04\"\n", "the playbook is for Ubuntu 26.04, not ubuntu 24.04"},
		{"debian", 0, "ID=debian\nVERSION_ID=\"13\"\n", "the playbook is for Ubuntu 26.04, not debian 13"},
		{"no file", 0, "", "the playbook is for Ubuntu 26.04, not  "},
		{"quoted id", 0, "ID=\"ubuntu\"\nVERSION_ID=26.04\n", ""},
	} {
		if got := serverRefusal(tc.euid, []byte(tc.text)); got != tc.want {
			t.Errorf("%s: serverRefusal = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestServerCommandRefusesOtherArgs(t *testing.T) {
	for _, args := range [][]string{{"apply", "--tags", "base"}, {"apply", "--only", "nothing"}, {"apply", "--", "-C"}, {"other"}} {
		var stdout, stderr bytes.Buffer
		m := effecttest.NewMachine()
		if got := serverCommand(context.Background(), args, 0, fstest.MapFS{}, "devel", m, strings.NewReader(""), &stdout, &stderr); got != 2 ||
			!strings.Contains(stderr.String(), "usage: ws server COMMAND") {
			t.Errorf("%q: status = %d, stderr = %q", args, got, stderr.String())
		}
	}
}

func TestServerCommandRefusesNonRoot(t *testing.T) {
	var stdout, stderr bytes.Buffer
	m := effecttest.NewMachine()
	m.Files["/etc/os-release"] = []byte("ID=ubuntu\nVERSION_ID=\"26.04\"\n")
	if got := serverCommand(context.Background(), []string{"apply"}, 1000, fstest.MapFS{}, "devel", m, strings.NewReader(""), &stdout, &stderr); got != 1 ||
		!strings.Contains(stderr.String(), "run it as root") {
		t.Errorf("status = %d, stderr = %q", got, stderr.String())
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "exec ") }) {
		t.Errorf("calls = %q, want no exec", m.Calls)
	}
}

func TestLoadSecrets(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	m := effecttest.NewMachine()
	var buf bytes.Buffer
	a := armor.NewWriter(&buf)
	w, _ := age.Encrypt(a, id.Recipient())
	_, _ = w.Write([]byte("shell_secret_env: {A: hunter2}\n"))
	_ = w.Close()
	_ = a.Close()
	m.Files["/s.age"] = buf.Bytes()
	s, vars, err := loadSecrets(m, "/s.age", strings.NewReader(id.String()+"\n"), io.Discard)
	if err != nil || s.Shell.SecretEnv["A"] != "hunter2" || !strings.Contains(string(vars), "hunter2") {
		t.Errorf("got %+v, %q, %v", s, vars, err)
	}
	if s, vars, err = loadSecrets(m, "", nil, io.Discard); err != nil || vars != nil || s.Shell.SecretEnv != nil {
		t.Errorf("no file: %+v, %q, %v", s, vars, err)
	}
	other, _ := age.GenerateX25519Identity()
	if _, _, err = loadSecrets(m, "/s.age", strings.NewReader(other.String()), io.Discard); err == nil ||
		strings.Contains(err.Error(), "hunter2") {
		t.Errorf("wrong identity: %v", err)
	}
	if _, _, err = loadSecrets(m, "/s.age", strings.NewReader(""), io.Discard); err == nil {
		t.Error("no identity: no error")
	}
}
