package effect

import (
	"slices"
	"testing"
)

func TestWithEnv(t *testing.T) {
	base := []string{"PATH=/bin", "LANG=C", "A=1"}
	got := withEnv(base, []string{"LANG=C.UTF-8", "NEW=x"})
	want := []string{"PATH=/bin", "A=1", "LANG=C.UTF-8", "NEW=x"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if !slices.Equal(base, []string{"PATH=/bin", "LANG=C", "A=1"}) {
		t.Errorf("base changed: %v", base)
	}
}

func TestOSMachineExecNoCommand(t *testing.T) {
	if err := (OSMachine{}).Exec("ws-no-such-command", nil, nil); err == nil {
		t.Error("Exec of a missing command returned nil")
	}
}
