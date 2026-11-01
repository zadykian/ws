package main

import (
	"bytes"
	"runtime/debug"
	"testing"
)

func TestVersionOf(t *testing.T) {
	tests := []struct {
		name string
		info *debug.BuildInfo
		ok   bool
		want string
	}{
		{"no build info", nil, false, "devel"},
		{"devel", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true, "devel"},
		{"empty", &debug.BuildInfo{}, true, "devel"},
		{"tag", &debug.BuildInfo{Main: debug.Module{Version: "v0.1.0"}}, true, "v0.1.0"},
		{"pseudo", &debug.BuildInfo{Main: debug.Module{Version: "v0.1.1-0.20261101120000-8d1e5310abcd+dirty"}}, true,
			"v0.1.1-0.20261101120000-8d1e5310abcd+dirty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := versionOf(tt.info, tt.ok); got != tt.want {
				t.Errorf("versionOf = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVersionCommand(t *testing.T) {
	var out bytes.Buffer
	if got := versionCommand(&out); got != 0 {
		t.Fatalf("status = %d, want 0", got)
	}
	if out.Len() == 0 || out.Bytes()[out.Len()-1] != '\n' {
		t.Errorf("output = %q, want one line", out.String())
	}
}
