package main

import (
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
