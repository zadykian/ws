//go:build linux

package main

import (
	"strings"
	"testing"
)

func TestServerGroupOnLinux(t *testing.T) {
	p := fakeProcess()
	if status, stdout, _ := runGroups(p, "server"); status != 0 || !strings.Contains(stdout, "usage: ws server COMMAND") {
		t.Errorf("status = %d, stdout = %q", status, stdout)
	}
	if _, stdout, _ := runGroups(p); !strings.Contains(stdout, "\n  server ") {
		t.Errorf("help = %q, want a line for server", stdout)
	}
}
