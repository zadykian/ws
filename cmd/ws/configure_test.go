package main

import (
	"strings"
	"testing"
)

func TestMacConfigureAtATerminalDrawsAForm(t *testing.T) {
	p, _ := macTerminal()
	status, stdout, stderr := runWs(p, "\x03", "mac", "configure")
	if status != 130 || stdout != "" || !strings.Contains(stderr, enterScreen) || !strings.Contains(stderr, leaveScreen) {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}
