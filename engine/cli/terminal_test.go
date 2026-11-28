package cli

import (
	"os"
	"syscall"
	"testing"
)

func TestAPipeIsNoTerminal(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close(); _ = w.Close() }()
	if isTerminal(int(r.Fd())) {
		t.Error("a pipe's read end is a terminal")
	}
}

func TestDevNullIsNoTerminal(t *testing.T) {
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = null.Close() }()
	if isTerminal(int(null.Fd())) {
		t.Error("/dev/null is a terminal")
	}
}

func TestAPseudoTerminalIsATerminal(t *testing.T) {
	pty, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("no pseudo-terminal here: %v", err)
	}
	defer func() { _ = pty.Close() }()
	if !isTerminal(int(pty.Fd())) {
		t.Error("a pseudo-terminal is no terminal")
	}
}
