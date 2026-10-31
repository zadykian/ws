package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
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
		{"missing", []string{"/none"}, "", 1, "", "ws conf: /none is missing: docs/mac.md"},
		{"refused", []string{"/bad"}, "", 1, "", "ws conf: /bad, line 2: unknown key WS_NOPE"},
		{"two files", []string{"a", "b"}, "", 2, "", "usage: ws conf [FILE]"},
	}
	m := effecttest.NewMachine()
	m.Files["/h/.config/ws/mac.conf"] = []byte("WS_USER=ada\n")
	m.Files["/e/mac.conf"] = []byte("WS_USER=eve\n")
	m.Files["/a/mac.conf"] = []byte("WS_USER=amy\n")
	m.Files["/bad"] = []byte("WS_USER=ada\nWS_NOPE=1\n")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env["WS_CONF"] = tt.wsConf
			var stdout, stderr bytes.Buffer
			got := confCommand(tt.args, m, func(k string) string { return env[k] }, &stdout, &stderr)
			if got != tt.wantStatus {
				t.Errorf("status = %d, want %d", got, tt.wantStatus)
			}
			if !strings.HasPrefix(stdout.String(), tt.wantOut) {
				t.Errorf("stdout = %q, want it to start with %q", stdout.String(), tt.wantOut)
			}
			if !strings.Contains(stderr.String(), tt.wantErr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), tt.wantErr)
			}
		})
	}
}
