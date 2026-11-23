package effect_test

import (
	"bytes"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
)

func TestTerminalOut(t *testing.T) {
	var w bytes.Buffer
	term := effect.NewTerminal(&w)
	term.Say("hello %s", "mac")
	term.Raw("ws %s", "key")
	term.Warn("careful")
	term.ByHand("open %s", "System Settings")
	term.Diff("f", []byte("a\n"), []byte("b\n"))
	term.Flush()
	term.Flush()
	want := "hello mac\nws key\nwarning: careful\n--- f\n+++ f\n@@ -1 +1 @@\n-a\n+b\n" +
		"\nWhat stays by hand:\n- open System Settings\n"
	if w.String() != want {
		t.Errorf("output = %q, want %q", w.String(), want)
	}
}
