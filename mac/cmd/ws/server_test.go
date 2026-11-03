package main

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
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

func TestServerApplyArgs(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want []string
		ok   bool
	}{
		{nil, nil, true},
		{[]string{"--"}, []string{}, true},
		{[]string{"--", "--check", "-e", "@f"}, []string{"--check", "-e", "@f"}, true},
		{[]string{"--check"}, nil, false},
		{[]string{"x", "--"}, nil, false},
	} {
		got, ok := serverApplyArgs(tc.args)
		if ok != tc.ok || !slices.Equal(got, tc.want) {
			t.Errorf("serverApplyArgs(%q) = %q, %v, want %q, %v", tc.args, got, ok, tc.want, tc.ok)
		}
	}
}

func TestServerCommandRefusesOtherArgs(t *testing.T) {
	for _, args := range [][]string{{"apply", "--check"}, {"other"}} {
		var stdout, stderr bytes.Buffer
		m := effecttest.NewMachine()
		if got := serverCommand(context.Background(), args, 0, fstest.MapFS{}, "devel", m, &stdout, &stderr); got != 2 ||
			!strings.Contains(stderr.String(), "usage: ws server COMMAND") {
			t.Errorf("%q: status = %d, stderr = %q", args, got, stderr.String())
		}
	}
}

func TestServerCommandRefusesNonRoot(t *testing.T) {
	var stdout, stderr bytes.Buffer
	m := effecttest.NewMachine()
	m.Files["/etc/os-release"] = []byte("ID=ubuntu\nVERSION_ID=\"26.04\"\n")
	if got := serverCommand(context.Background(), []string{"apply"}, 1000, fstest.MapFS{}, "devel", m, &stdout, &stderr); got != 1 ||
		!strings.Contains(stderr.String(), "run it as root") {
		t.Errorf("status = %d, stderr = %q", got, stderr.String())
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "exec ") }) {
		t.Errorf("calls = %q, want no exec", m.Calls)
	}
}

func TestExecPlaybook(t *testing.T) {
	m := effecttest.NewMachine()
	if err := execPlaybook(m, "v1.2.3", []string{"--check", "-e", "@x.yml"}); err != nil {
		t.Fatal(err)
	}
	d := "/var/lib/ws/v1.2.3"
	want := "exec ansible-playbook -i " + d + "/inventory.ini " + d + "/site.yml --check -e @x.yml"
	if !slices.Equal(m.Calls, []string{want}) {
		t.Errorf("calls = %q, want %q", m.Calls, want)
	}
	wantEnv := []string{"LC_ALL=C.UTF-8", "ANSIBLE_CONFIG=" + d + "/ansible.cfg", "ANSIBLE_COLLECTIONS_PATH=" + d + "/collections"}
	if len(m.ExecEnv) != 1 || !slices.Equal(m.ExecEnv[0], wantEnv) {
		t.Errorf("env = %q, want %q", m.ExecEnv, wantEnv)
	}
}
