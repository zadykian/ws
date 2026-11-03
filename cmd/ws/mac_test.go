package main

import (
	"strings"
	"testing"
)

func TestSecretsPassesItsArgumentsAndStatus(t *testing.T) {
	status, _, stderr := runWs(fakeProcess(), "", "mac", "secrets", "nope")
	if status != 2 || !strings.Contains(stderr, "usage: ws mac secrets COMMAND") {
		t.Errorf("status = %d, stderr = %q; want the usage of secrets, status 2", status, stderr)
	}
}
