package file

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

// rootReads is a machine whose reads by name answer the target of a link that the user swapped in
// after the check: root:x, owner 0 and mode 0777. A read of a home file fails at the link.
type rootReads struct{ *effecttest.Machine }

func (rootReads) ReadRootFile(string) ([]byte, error) { return []byte("root:x\n"), nil }

func (rootReads) RootAttrs(string) (effect.Attrs, error) { return effect.Attrs{Mode: 0o777}, nil }

func TestHomeLinesPlansACreateWithNoSecondRead(t *testing.T) {
	m := effecttest.NewMachine()
	l := homeLinesIn(m, true)
	l.M = rootReads{m}
	cs, err := l.Plan(context.Background())
	if err != nil || len(cs) != 1 {
		t.Fatalf("Plan = %v, %v; want one create", cs, err)
	}
	if text := cs[0].String() + step.PendingOf(cs[0]).Text; strings.Contains(text, "root:x") {
		t.Errorf("the plan shows the target of a link: %q", text)
	}
	calls := joinCalls(m)
	if !strings.Contains(calls, "read home /home/ws/f") || strings.Contains(calls, "read root") {
		t.Errorf("calls %v; want one read of the home file and no read by name", m.Calls)
	}
}

func TestHomeLinesKeepTakesTheAttributesOfItsRead(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/home/ws/f"], m.Modes["/home/ws/f"] = []byte("a\n"), 0o640
	m.Owners["/home/ws/f"] = [2]int{1002, 1003}
	cs, err := homeLinesIn(m, true).Plan(context.Background())
	if err != nil || len(cs) != 2 {
		t.Fatalf("Plan = %v, %v; want a keep and a write", cs, err)
	}
	m.Owners["/home/ws/f"], m.Modes["/home/ws/f"] = [2]int{1004, 1005}, 0o600
	if err := cs[0].Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	kept := "/home/ws/f.20260102030405~"
	if m.Owners[kept] != [2]int{1004, 1005} || m.Modes[kept] != 0o600 {
		t.Errorf("the copy has owner %v and mode %o; want those of its read", m.Owners[kept], m.Modes[kept])
	}
}

func TestHomeLinesKeepFailsWhereTheFileWentAfterThePlan(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/home/ws/f"] = []byte("a\n")
	cs, err := homeLinesIn(m, true).Plan(context.Background())
	if err != nil || len(cs) != 2 {
		t.Fatalf("Plan = %v, %v; want a keep and a write", cs, err)
	}
	delete(m.Files, "/home/ws/f")
	if err := cs[0].Apply(context.Background()); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Apply = %v, want fs.ErrNotExist", err)
	}
	if _, kept := m.Files["/home/ws/f.20260102030405~"]; kept {
		t.Error("the keep wrote a copy of a file that was gone")
	}
}
