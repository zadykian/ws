//go:build !darwin

package cmd

import (
	"strings"
	"testing"
)

func TestMacGroupIsUnknownHere(t *testing.T) {
	status, _, stderr := runGroups(fakeProcess(), "client")
	if status != 2 || !strings.Contains(stderr, `unknown command "client"`) {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	if _, stdout, _ := runGroups(fakeProcess()); strings.Contains(stdout, "\n  client ") {
		t.Errorf("help = %q, want no line for client", stdout)
	}
}
