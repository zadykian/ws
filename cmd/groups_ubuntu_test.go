//go:build linux

package cmd

import (
	"strings"
	"testing"
)

func TestServerGroupOnLinux(t *testing.T) {
	p := fakeProcess()
	if status, stdout, _ := runGroups(p, "server"); status != 0 || !strings.Contains(stdout, "Available Commands:") || !strings.Contains(stdout, "\n  fans ") {
		t.Errorf("status = %d, stdout = %q, want the commands of server", status, stdout)
	}
	if status, stdout, _ := runGroups(p, "unit"); status != 0 || !strings.Contains(stdout, "\n  list ") {
		t.Errorf("status = %d, stdout = %q, want the commands of unit", status, stdout)
	}
	if _, stdout, _ := runGroups(p); !strings.Contains(stdout, "\n  server ") || !strings.Contains(stdout, "\n  unit ") {
		t.Errorf("help = %q, want a line for server and for unit", stdout)
	}
}

func TestUpdateEndsInTheServersSetupOnLinux(t *testing.T) {
	if status, stdout, _ := runGroups(fakeProcess(), "update", "--help"); status != 0 || !strings.Contains(stdout, "ws server setup, and none where it has no side yet") {
		t.Errorf("status = %d, stdout = %q, want an update that ends in ws server setup", status, stdout)
	}
}
