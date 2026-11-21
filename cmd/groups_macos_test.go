//go:build darwin

package cmd

import (
	"strings"
	"testing"
)

func TestMacGroupOnDarwin(t *testing.T) {
	p := fakeProcess()
	if status, stdout, stderr := runGroups(p, "client"); status != 0 || !strings.Contains(stdout, "Available Commands:") || stderr != "" {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	if _, stdout, _ := runGroups(p); !strings.Contains(stdout, "\n  client ") {
		t.Errorf("help = %q, want a line for client", stdout)
	}
}

func TestUnitGroupOnDarwin(t *testing.T) {
	status, stdout, stderr := runGroups(fakeProcess(), "unit", "list", "--help")
	if status != 0 || !strings.Contains(stdout, "/opt/ws/roles/client/macos") || stderr != "" {
		t.Errorf("status = %d, stdout = %q, stderr = %q; want the Mac's specs", status, stdout, stderr)
	}
}
