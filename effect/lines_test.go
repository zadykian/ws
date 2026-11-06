package effect_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/zadykian/ws/effect"
)

func TestPassLinesKeepsWhatItTakesOffStderr(t *testing.T) {
	var shown bytes.Buffer
	take := func(line string) bool { return line == "event" }
	script := `echo one; echo event; echo two >&2; printf last`
	err := effect.OSMachine{Stderr: &shown}.PassLines(context.Background(), nil, take, "sh", "-c", script)
	if want := "one\ntwo\nlast"; err != nil || shown.String() != want {
		t.Errorf("Stderr = %q, want %q, err = %v", shown.String(), want, err)
	}
}
