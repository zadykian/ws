//go:build linux

package cmd

import (
	"strings"
	"testing"
)

func TestUbuntuGroupOnLinux(t *testing.T) {
	p := fakeProcess()
	if status, stdout, _ := runGroups(p, "ubuntu", "guest"); status != 0 || !strings.Contains(stdout, "Available Commands:") || !strings.Contains(stdout, "\n  fans ") {
		t.Errorf("status = %d, stdout = %q, want the commands of ubuntu guest", status, stdout)
	}
	if status, stdout, _ := runGroups(p, "unit"); status != 0 || !strings.Contains(stdout, "\n  list ") {
		t.Errorf("status = %d, stdout = %q, want the commands of unit", status, stdout)
	}
	if _, stdout, _ := runGroups(p); !strings.Contains(stdout, "\n  ubuntu ") || !strings.Contains(stdout, "\n  unit ") {
		t.Errorf("help = %q, want a line for ubuntu and for unit", stdout)
	}
}

func TestUpdateEndsInTheGuestsSetupOnLinux(t *testing.T) {
	if status, stdout, _ := runGroups(fakeProcess(), "update", "--help"); status != 0 || !strings.Contains(stdout, "ws ubuntu guest setup, and none where it has no side yet") {
		t.Errorf("status = %d, stdout = %q, want an update that ends in ws ubuntu guest setup", status, stdout)
	}
}
