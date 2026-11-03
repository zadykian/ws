//go:build !darwin

package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestMacGroupIsUnknownHere(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if got := run([]string{"mac"}, &stdout, &stderr); got != 2 || !strings.Contains(stderr.String(), `unknown command "mac"`) {
		t.Errorf("status = %d, stderr = %q", got, stderr.String())
	}
	stdout.Reset()
	run(nil, &stdout, &stderr)
	if strings.Contains(stdout.String(), "\n  mac ") {
		t.Errorf("usage = %q, want no line for mac", stdout.String())
	}
}
