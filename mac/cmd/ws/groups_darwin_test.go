//go:build darwin

package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestMacGroupOnDarwin(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if got := run([]string{"mac"}, &stdout, &stderr); got != 0 || !strings.Contains(stdout.String(), "usage: ws mac COMMAND") {
		t.Errorf("status = %d, stdout = %q, stderr = %q", got, stdout.String(), stderr.String())
	}
	stdout.Reset()
	run(nil, &stdout, &stderr)
	if !strings.Contains(stdout.String(), "\n  mac  ") {
		t.Errorf("usage = %q, want a line for mac", stdout.String())
	}
}
