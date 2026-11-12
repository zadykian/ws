package main

import "testing"

func TestProcessLockDirFallsBackToTmp(t *testing.T) {
	for _, tc := range []struct{ tmpdir, want string }{{"", "/tmp"}, {"/var/x", "/var/x"}} {
		p := process{getenv: func(k string) string {
			if k == "TMPDIR" {
				return tc.tmpdir
			}
			return ""
		}}
		if got := p.lockDir(); got != tc.want {
			t.Errorf("TMPDIR %q: lockDir() = %q, want %q", tc.tmpdir, got, tc.want)
		}
	}
}
