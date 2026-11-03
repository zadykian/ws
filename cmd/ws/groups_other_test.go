//go:build !darwin

package main

import (
	"strings"
	"testing"
)

func TestMacGroupIsUnknownHere(t *testing.T) {
	status, _, stderr := runGroups(fakeProcess(), "mac")
	if status != 2 || !strings.Contains(stderr, `unknown command "mac"`) {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	if _, stdout, _ := runGroups(fakeProcess()); strings.Contains(stdout, "\n  mac ") {
		t.Errorf("help = %q, want no line for mac", stdout)
	}
}
