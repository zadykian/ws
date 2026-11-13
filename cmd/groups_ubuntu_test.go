//go:build linux

package cmd

import (
	"strings"
	"testing"
)

func TestServerGroupOnLinux(t *testing.T) {
	p := fakeProcess()
	if status, stdout, _ := runGroups(p, "server"); status != 0 || !strings.Contains(stdout, "Available Commands:") {
		t.Errorf("status = %d, stdout = %q", status, stdout)
	}
	if status, stdout, _ := runGroups(p, "server", "ubuntu"); status != 0 || !strings.Contains(stdout, "\n  apply ") {
		t.Errorf("status = %d, stdout = %q, want the commands of server ubuntu", status, stdout)
	}
	if _, stdout, _ := runGroups(p); !strings.Contains(stdout, "\n  server ") {
		t.Errorf("help = %q, want a line for server", stdout)
	}
}
