package main

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestConfCommand(t *testing.T) {
	env := map[string]string{"HOME": "/h", "WS_CONF": ""}
	tests := []struct {
		name       string
		args       []string
		wsConf     string
		wantStatus int
		wantOut    string
		wantErr    string
	}{
		{"home", nil, "", 0, "WS_USER='ada'\n", ""},
		{"env", nil, "/e/mac.conf", 0, "WS_USER='eve'\n", ""},
		{"argument", []string{"/a/mac.conf"}, "/e/mac.conf", 0, "WS_USER='amy'\n", ""},
		{"missing", []string{"/none"}, "", 1, "", "ws mac conf: /none is missing: docs/mac.md"},
		{"refused", []string{"/bad"}, "", 1, "", "ws mac conf: /bad, line 2: unknown key WS_NOPE"},
		{"lines", []string{"--lines", "/pair"}, "", 0, "WS_USER='zadykian'\nWS_TARGET='lan'\nWS_PUBLIC_KEY=''\nWS_LAN='10.88.0.0/24'\n", ""},
		{"pair", []string{"/pair"}, "", 1, "", "WS_LAN and WS_LAN_ADDRESS go together"},
		{"two files", []string{"a", "b"}, "", 2, "", "ws mac conf: accepts at most 1 arg(s), received 2"},
	}
	m := effecttest.NewMachine()
	m.Files["/h/.config/ws/mac.conf"] = []byte("WS_USER=ada\n")
	m.Files["/e/mac.conf"] = []byte("WS_USER=eve\n")
	m.Files["/a/mac.conf"] = []byte("WS_USER=amy\n")
	m.Files["/pair"] = []byte("WS_LAN=10.88.0.0/24\n")
	m.Files["/bad"] = []byte("WS_USER=ada\nWS_NOPE=1\n")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env["WS_CONF"] = tt.wsConf
			p := withEnv(fakeProcess(), env)
			p.machine = m
			got, stdout, stderr := runWs(p, "", append([]string{"mac", "conf"}, tt.args...)...)
			if got != tt.wantStatus {
				t.Errorf("status = %d, want %d", got, tt.wantStatus)
			}
			if !strings.HasPrefix(stdout, tt.wantOut) {
				t.Errorf("stdout = %q, want it to start with %q", stdout, tt.wantOut)
			}
			if !strings.Contains(stderr, tt.wantErr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr, tt.wantErr)
			}
		})
	}
}

func TestConfReportsEachLineOfARefusedFile(t *testing.T) {
	p := withEnv(fakeProcess(), map[string]string{})
	m := effecttest.NewMachine()
	m.Files["/bad"] = []byte("WS_NOPE=1\nWS_NOPE2=1\n")
	p.machine = m
	status, _, stderr := runWs(p, "", "mac", "conf", "/bad")
	want := "ws mac conf: /bad, line 1: unknown key WS_NOPE\nws mac conf: /bad, line 2: unknown key WS_NOPE2\n"
	if status != 1 || stderr != want {
		t.Errorf("status = %d, stderr = %q; want 1, %q", status, stderr, want)
	}
}
