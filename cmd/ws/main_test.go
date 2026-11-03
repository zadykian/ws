package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantStatus int
		wantOut    string
		wantErr    string
	}{
		{"no command", nil, 0, "usage: ws", ""},
		{"help", []string{"help"}, 0, "usage: ws", ""},
		{"version", []string{"version"}, 0, "", ""},
		{"flag", []string{"--help"}, 0, "usage: ws", ""},
		{"unknown", []string{"nope"}, 2, "", `unknown command "nope"`},
		{"old setup", []string{"setup", "--check", "ssh"}, 2, "", `unknown command "setup"`},
		{"old tunnel", []string{"tunnel", "status"}, 2, "", `unknown command "tunnel"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			got := run(tt.args, &stdout, &stderr)
			if got != tt.wantStatus {
				t.Errorf("status = %d, want %d", got, tt.wantStatus)
			}
			if !strings.Contains(stdout.String(), tt.wantOut) {
				t.Errorf("stdout = %q, want it to contain %q", stdout.String(), tt.wantOut)
			}
			if !strings.Contains(stderr.String(), tt.wantErr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), tt.wantErr)
			}
		})
	}
}

func TestTunnelStatusNeedsRoot(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if got := tunnelStatus(501, &stdout, &stderr); got != 1 {
		t.Errorf("status = %d, want 1", got)
	}
	if !strings.Contains(stderr.String(), "run it as root, with sudo") || stdout.Len() != 0 {
		t.Errorf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}

func TestTunnelKeyNeedsRoot(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if got := tunnelKey(501, &stdout, &stderr); got != 1 {
		t.Errorf("status = %d, want 1", got)
	}
	if !strings.Contains(stderr.String(), "run it as root, with sudo") || stdout.Len() != 0 {
		t.Errorf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}
