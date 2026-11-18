package file

import (
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// checkedHomeLines is homeLinesIn's lines at the top, checked by "checker PATH".
func checkedHomeLines(m *effecttest.Machine) HomeLines {
	l := homeLinesIn(m, true)
	l.Check = func(p string) (string, []string) { return "checker", []string{p} }
	return l
}

// applyUntilError applies the plan's changes in order, as a step does, and returns the first error.
func applyUntilError(t *testing.T, l HomeLines) error {
	cs, err := l.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cs {
		if err := c.Apply(t.Context()); err != nil {
			return err
		}
	}
	return nil
}

func TestHomeLinesCheckRefusedLeavesTheFile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/home/ws/f"], m.Modes["/home/ws/f"] = []byte("a\n"), 0o640
	m.Owners["/home/ws/f"] = [2]int{1002, 1003}
	m.Results["checker /home/ws/f.ws-check"] = effecttest.Result{Out: []byte("bad\n"), Err: errors.New("exit status 1")}
	err := applyUntilError(t, checkedHomeLines(m))
	if err == nil || !strings.Contains(err.Error(), "checker refuses the new /home/ws/f") {
		t.Fatalf("apply = %v, want the check's refusal", err)
	}
	if string(m.Files["/home/ws/f"]) != "a\n" || m.Owners["/home/ws/f"] != [2]int{1002, 1003} || m.Modes["/home/ws/f"] != 0o640 {
		t.Errorf("file %q, %v, %o; want it as it was", m.Files["/home/ws/f"], m.Owners["/home/ws/f"], m.Modes["/home/ws/f"])
	}
	for p := range m.Files {
		if strings.HasSuffix(p, "~") || strings.HasSuffix(p, ".ws-check") {
			t.Errorf("%s stayed", p)
		}
	}
}

func TestHomeLinesCheckPassedWrites(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/home/ws/f"] = []byte("a\n")
	m.Results["checker /home/ws/f.ws-check"] = effecttest.Result{}
	if err := applyUntilError(t, checkedHomeLines(m)); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/home/ws/f"]); got != "ws\n\na\n" {
		t.Errorf("file = %q, want the line at the top", got)
	}
	if got := string(m.Files["/home/ws/f.20260102030405~"]); got != "a\n" {
		t.Errorf("copy = %q, want the old file", got)
	}
	if _, ok := m.Files["/home/ws/f.ws-check"]; ok {
		t.Error("the checked copy stayed")
	}
}

func TestHomeLinesCheckOnANewFile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["checker /home/ws/f.ws-check"] = effecttest.Result{Err: errors.New("exit status 1")}
	if err := applyUntilError(t, checkedHomeLines(m)); err == nil {
		t.Fatal("a refused new file was written")
	}
	if _, ok := m.Files["/home/ws/f"]; ok {
		t.Error("a refused new file was made")
	}
	m.Results["checker /home/ws/f.ws-check"] = effecttest.Result{}
	if err := applyUntilError(t, checkedHomeLines(m)); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/home/ws/f"]); got != "ws\n" || m.Owners["/home/ws/f"] != [2]int{1000, 1001} {
		t.Errorf("file = %q, owner %v; want the line, the account's", got, m.Owners["/home/ws/f"])
	}
}
