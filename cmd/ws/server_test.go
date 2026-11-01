package main

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

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
