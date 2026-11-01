package main

import (
	"slices"
	"testing"
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
