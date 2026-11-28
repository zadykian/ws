package effecttest_test

import (
	"context"
	"errors"
	"io/fs"
	"slices"
	"syscall"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

var (
	_ effect.Lines = (*effecttest.Machine)(nil)
)

func TestMachinePassLines(t *testing.T) {
	boom := errors.New("boom")
	m := effecttest.NewMachine()
	m.Results["ssh ws"] = effecttest.Result{Out: []byte("event\nlog\n\nlast"), Err: boom}
	take := func(line string) bool { return line == "event" }
	if err := m.PassLines(context.Background(), []byte("in"), take, "ssh", "ws"); !errors.Is(err, boom) {
		t.Errorf("err = %v, want the result's", err)
	}
	if want := []string{"log", "", "last"}; !slices.Equal(m.Passed, want) {
		t.Errorf("Passed = %q, want %q", m.Passed, want)
	}
	if !slices.Equal(m.Calls, []string{"run ssh ws"}) || !slices.Equal(m.Inputs, []string{"in"}) {
		t.Errorf("Calls = %q, Inputs = %q", m.Calls, m.Inputs)
	}
}

func TestMachineAfterChangesTheNextResult(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["launchctl print x"], m.Results["launchctl bootout x"] = effecttest.Result{Out: []byte("x")}, effecttest.Result{}
	m.After = map[string]map[string]effecttest.Result{"launchctl bootout x": {"launchctl print x": {Out: []byte("gone")}}}
	ctx := context.Background()
	before, _ := m.Run(ctx, "launchctl", "print", "x")
	if _, err := m.RunRoot(ctx, "launchctl", "bootout", "x"); err != nil {
		t.Fatal(err)
	}
	if after, _ := m.Run(ctx, "launchctl", "print", "x"); string(before) != "x" || string(after) != "gone" {
		t.Errorf("print = %q, then %q; want x, then gone", before, after)
	}
}

func TestMachineRootMkdirRecordsTheDirectory(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/etc/old"] = 0o700
	for _, args := range [][]string{{"-m", "750", "/etc/new"}, {"-p", "--", "/var/x", "/etc/old"}} {
		if _, err := m.RunRoot(context.Background(), "mkdir", args...); err != nil {
			t.Fatal(err)
		}
	}
	for path, want := range map[string]effect.Attrs{"/etc/new": {Mode: 0o750}, "/var/x": {Mode: 0o755}, "/etc/old": {Mode: 0o700}} {
		if got, err := m.RootAttrs(path); err != nil || got != want {
			t.Errorf("RootAttrs(%s) = %+v, %v; want %+v", path, got, err, want)
		}
	}
}

func TestMachineReadHomeFileRefusesLinksDirectoriesAndOversize(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/f"] = []byte("one")
	m.Owners["/h/f"] = [2]int{1000, 1001}
	m.Files["/h/d/g"] = []byte("two")
	m.Files["/h/big"] = make([]byte, effect.HomeFileMax+1)
	m.Files["/out"] = []byte("secret")
	m.Dirs["/h/dir"] = 0o755
	m.Links["/h/l"] = "/out"
	m.Files["/h/l"] = []byte("secret")
	m.Links["/h/d"] = "/elsewhere"
	got, a, err := m.ReadHomeFile("/h", "/h/f")
	if err != nil || string(got) != "one" || a.UID != 1000 || a.GID != 1001 {
		t.Errorf("a file: %q, %+v, %v; want one with its owner", got, a, err)
	}
	for path, want := range map[string]error{
		"/h/l":    syscall.ELOOP,
		"/h/d/g":  syscall.ELOOP,
		"/h/dir":  effect.ErrNotRegular,
		"/h/big":  effect.ErrTooLarge,
		"/h/none": fs.ErrNotExist,
	} {
		if got, _, err := m.ReadHomeFile("/h", path); !errors.Is(err, want) {
			t.Errorf("ReadHomeFile(%s) = %q, %v; want %v", path, got, err, want)
		}
	}
	for _, path := range []string{"/out", "/h", "/h/../out"} {
		if _, _, err := m.ReadHomeFile("/h", path); err == nil {
			t.Errorf("ReadHomeFile(%s): no error, want one for a path not below the home", path)
		}
	}
}

func TestMachineRunTerminalRecordsItsOwnKind(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["fnox get A"] = effecttest.Result{Out: []byte("v\n")}
	if out, err := m.RunTerminal(context.Background(), "fnox", "get", "A"); err != nil || string(out) != "v\n" {
		t.Errorf("RunTerminal = %q, %v; want the result's", out, err)
	}
	if !slices.Equal(m.Calls, []string{"terminal fnox get A"}) {
		t.Errorf("Calls = %q, want the terminal's line", m.Calls)
	}
}
