package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestMacCommand(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantStatus int
		wantOut    string
		wantErr    string
	}{
		{"no command", nil, 0, "usage: ws mac COMMAND", ""},
		{"help", []string{"help"}, 0, "usage: ws mac COMMAND", ""},
		{"tunnel alone", []string{"tunnel"}, 2, "", "usage: ws mac tunnel run | status | key"},
		{"tunnel status extra", []string{"tunnel", "status", "x"}, 2, "", "usage: ws mac tunnel run"},
		{"tunnel loaded extra", []string{"tunnel", "loaded", "x"}, 2, "", "usage: ws mac tunnel loaded"},
		{"tunnel routes alone", []string{"tunnel", "routes"}, 2, "", "usage: ws mac tunnel routes"},
		{"resolver alone", []string{"resolver"}, 2, "", "usage: ws mac resolver"},
		{"unknown", []string{"nope"}, 2, "", `ws mac: unknown command "nope"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if got := macCommand(tt.args, &stdout, &stderr); got != tt.wantStatus {
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

func TestUsageListsGroups(t *testing.T) {
	g := []group{{name: "mac", summary: "the Mac's commands"}}
	if got := usageOf(g); !strings.Contains(got, "\n  mac  the Mac's commands\n") {
		t.Errorf("usage = %q, want a line for mac", got)
	}
	if got := usageOf(nil); strings.Contains(got, "\n  mac ") {
		t.Errorf("usage = %q, want no line for mac", got)
	}
}
