//go:build linux

package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestServerGroupOnLinux(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if got := run([]string{"server"}, &stdout, &stderr); got != 0 || !strings.Contains(stdout.String(), "usage: ws server COMMAND") {
		t.Errorf("status = %d, stdout = %q, stderr = %q", got, stdout.String(), stderr.String())
	}
	stdout.Reset()
	run(nil, &stdout, &stderr)
	if !strings.Contains(stdout.String(), "\n  server  ") {
		t.Errorf("usage = %q, want a line for server", stdout.String())
	}
}
