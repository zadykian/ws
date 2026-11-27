package file_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/file"
)

func rootCalls(m *effecttest.Machine) []string {
	var got []string
	for _, c := range m.Calls {
		if strings.HasPrefix(c, "read root ") || strings.HasPrefix(c, "stat root ") {
			got = append(got, c)
		}
	}
	return got
}

func TestPublicPlansACreateAsFileDoes(t *testing.T) {
	m := effecttest.NewMachine()
	want, err := file.File{M: effecttest.NewMachine(), Path: "/etc/x", Content: []byte("a\n"), Mode: 0o644}.Plan(context.Background())
	if err != nil || len(want) != 1 {
		t.Fatalf("File plan = %v, %v", want, err)
	}
	got, err := file.Public{M: m, Path: "/etc/x", Content: []byte("a\n"), Mode: 0o644}.Plan(context.Background())
	if err != nil || len(got) != 1 || got[0].String() != want[0].String() {
		t.Errorf("Public plan = %v, %v; want File's create", got, err)
	}
}

func TestPublicAsksForNoRootWhereNothingDiffers(t *testing.T) {
	m := effecttest.NewMachine()
	p := file.Public{M: m, Path: "/etc/x", Content: []byte("a\n"), Mode: 0o644}
	if got := applyAll(t, p); len(got) != 1 {
		t.Fatalf("first plan = %q, want a create", got)
	}
	m.Calls = nil
	if got := applyAll(t, p); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
	if got := rootCalls(m); len(got) != 0 {
		t.Errorf("second plan ran as root: %q", got)
	}
}

func TestPublicWritesOtherContentAndFixesTheOwner(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/etc/x"] = []byte("old\n")
	p := file.Public{M: m, Path: "/etc/x", Content: []byte("a\n"), Mode: 0o644}
	if got := applyAll(t, p); len(got) != 1 || !strings.HasPrefix(got[0], "write /etc/x") {
		t.Errorf("content plan = %q, want a write", got)
	}
	m.Owners["/etc/x"] = [2]int{1000, 1000}
	if got := applyAll(t, p); len(got) != 1 || !strings.HasPrefix(got[0], "set owner 0:0 and mode 644 of /etc/x") {
		t.Errorf("owner plan = %q, want an owner change", got)
	}
	m.Modes["/etc/x"] = 0o600
	if got := applyAll(t, p); len(got) != 1 || !strings.HasPrefix(got[0], "set owner 0:0 and mode 644 of /etc/x") {
		t.Errorf("mode plan = %q, want a mode change", got)
	}
}

func TestPublicReadsAsRootWhereTheUserCannot(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/etc/x"] = []byte("a\n")
	m.Root["/etc/x"] = effect.Attrs{Mode: 0o644}
	p := file.Public{M: m, Path: "/etc/x", Content: []byte("a\n"), Mode: 0o644}
	if got := applyAll(t, p); got != nil || !slices.Contains(m.Calls, "read root /etc/x") {
		t.Errorf("plan = %q, calls %q; want none, read as root", got, m.Calls)
	}
}

func TestPublicDir(t *testing.T) {
	m := effecttest.NewMachine()
	d := file.PublicDir{M: m, Path: "/etc/ws", Mode: 0o755}
	if got := applyAll(t, d); len(got) != 1 || !strings.HasPrefix(got[0], "create directory /etc/ws") {
		t.Errorf("first plan = %q, want a create", got)
	}
	m.Calls = nil
	if got := applyAll(t, d); got != nil || len(rootCalls(m)) != 0 {
		t.Errorf("second plan = %q, root calls %q; want none", got, rootCalls(m))
	}
	m.Dirs["/etc/ws"] = 0o700
	if got := applyAll(t, d); len(got) != 1 || !strings.HasPrefix(got[0], "set owner 0:0 and mode 755 of /etc/ws") {
		t.Errorf("mode plan = %q", got)
	}
}
