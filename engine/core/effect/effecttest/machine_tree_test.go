package effecttest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func treeMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Dirs["/a"] = 0o755
	m.Dirs["/a/d"] = 0o700
	m.Files["/a/d/f"] = []byte("f")
	m.Files["/a/g"] = []byte("g")
	m.Modes["/a/g"] = 0o755
	m.Links["/a/l"] = "g"
	m.Files["/ab"] = []byte("beside")
	return m
}

func TestMachineRemovesATree(t *testing.T) {
	m := treeMachine()
	if _, err := m.RunRoot(context.Background(), "rm", "-rf", "--", "/a"); err != nil {
		t.Fatal(err)
	}
	if len(m.Dirs)+len(m.Links)+len(m.Modes)+len(m.Files) != 1 || m.Files["/ab"] == nil {
		t.Errorf("left Files %v, Dirs %v, Links %v, Modes %v", m.Files, m.Dirs, m.Links, m.Modes)
	}
}

func TestMachineMovesATree(t *testing.T) {
	m := treeMachine()
	if _, err := m.RunRoot(context.Background(), "mv", "-T", "--", "/a", "/b"); err != nil {
		t.Fatal(err)
	}
	if string(m.Files["/b/d/f"]) != "f" || m.Dirs["/b/d"] != 0o700 || m.Links["/b/l"] != "g" || m.Modes["/b/g"] != 0o755 ||
		len(m.Files) != 3 || len(m.Dirs) != 2 || m.Files["/ab"] == nil {
		t.Errorf("moved Files %v, Dirs %v, Links %v, Modes %v", m.Files, m.Dirs, m.Links, m.Modes)
	}
}

func TestMachineTreeCommandsYieldToTheTable(t *testing.T) {
	boom := errors.New("boom")
	m := treeMachine()
	m.Results["rm -rf -- /a"] = effecttest.Result{Err: boom}
	if _, err := m.RunRoot(context.Background(), "rm", "-rf", "--", "/a"); !errors.Is(err, boom) || len(m.Files) != 3 {
		t.Errorf("rm = %v, Files %v", err, m.Files)
	}
}
