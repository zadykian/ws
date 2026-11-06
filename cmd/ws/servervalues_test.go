package main

import (
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestWriteVarsStartsTheFile(t *testing.T) {
	m := effecttest.NewMachine()
	if err := writeVars(m, []byte("---\n")); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files[serverVars]); got != "---\n" {
		t.Errorf("file = %q", got)
	}
	if m.Dirs["/etc/ws"] != 0o755 || m.Modes[serverVars] != 0o600 {
		t.Errorf("directory %o, file %o", m.Dirs["/etc/ws"], m.Modes[serverVars])
	}
}

func TestWriteVarsKeepsTheModeOfAFile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/etc/ws"] = 0o700
	m.Files[serverVars] = []byte("---\n")
	m.Modes[serverVars] = 0o640
	if err := writeVars(m, []byte("---\nws_user: ada\n")); err != nil {
		t.Fatal(err)
	}
	if m.Dirs["/etc/ws"] != 0o700 || m.Modes[serverVars] != 0o640 {
		t.Errorf("directory %o, file %o", m.Dirs["/etc/ws"], m.Modes[serverVars])
	}
}
