package netplan

import "testing"

func TestErrorLine(t *testing.T) {
	const root = "/tmp/abc.netplan"
	tests := []struct {
		name string
		r    Result
		want string
	}{
		{"netplan's own prefix", Result{RC: 1, Stderr: "** (generate:1): WARNING: x\nCommand failed: bad indent.\n  secret: hunter2\n"}, "bad indent"},
		{"an ERROR line", Result{RC: 1, Stderr: "ERROR: Invalid YAML: oops:\n"}, "ERROR: Invalid YAML: oops"},
		{"the root in a path", Result{RC: 1, Stderr: root + "/etc/netplan/x.yaml:3:5: Error in network definition: bad\n  key: pass"}, "/etc/netplan/x.yaml:3:5: Error in network definition: bad"},
		{"the message, where the command did not run", Result{RC: -1, Msg: "no such file"}, "no such file"},
		{"the exit code", Result{RC: 3, Stderr: "just a warning\n"}, "netplan exited with 3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errorLine(tt.r, root); got != tt.want {
				t.Errorf("errorLine = %q, want %q", got, tt.want)
			}
		})
	}
}
