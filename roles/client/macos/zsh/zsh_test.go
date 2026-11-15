package zsh_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/roles/client/macos/zsh"
)

const (
	hookPath  = "/etc/ws/zshrc"
	zshSource = "source /etc/ws/zshrc\n"
)

func zshStep(m *effecttest.Machine, out *effecttest.Out) resource.Lazy {
	return (&zsh.Zsh{Home: "/h", Hook: zsh.Hook, M: m, Out: out,
		Now: func() time.Time { return time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC) }}).Step()
}

func rootCalls(m *effecttest.Machine) []string {
	var got []string
	for _, c := range m.Calls {
		if strings.Contains(c, "root ") {
			got = append(got, c)
		}
	}
	return got
}

func TestZshMakesTheHookAndZshrcThenNothing(t *testing.T) {
	m := effecttest.NewMachine()
	out := &effecttest.Out{}
	s := zshStep(m, out)
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 3 {
		t.Fatalf("Plan = %v, %v; want /etc/ws, the hook and the line", p.Changes, err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files[hookPath]); got != string(zsh.Hook) {
		t.Errorf("the hook = %q, want the embedded file", got)
	}
	if got := string(m.Files["/h/.zshrc"]); got != zshSource {
		t.Errorf("~/.zshrc = %q, want %q", got, zshSource)
	}
	m.Calls = nil
	if p, err := s.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Errorf("second Plan = %v, %v; want none", p.Changes, err)
	}
	if calls := rootCalls(m); len(calls) != 0 {
		t.Errorf("the second plan asked for root: %q", calls)
	}
	items, err := s.Report(context.Background())
	if err != nil || len(items) != 3 || items[1].Conflict != "" || items[1].Want != "owner 0:0, mode 644" || items[2].Found != items[2].Want {
		t.Errorf("Report = %+v, %v; want /etc/ws, the hook and the line, no conflict", items, err)
	}
}

// The hook is root's by its place: a run replaces what lies there, and an old line stays.
func TestZshReplacesTheHookAndKeepsAnOldLine(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Files[hookPath] = []byte("echo mine\n")
	m.Files["/h/.zshrc"] = []byte("source ~/.config/ws/zshrc\n")
	m.Results["cp -p /h/.zshrc /h/.zshrc.20261109100000~"] = effecttest.Result{}
	s := zshStep(m, out)
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 4 || len(out.Lines) != 0 {
		t.Fatalf("Plan = %v, %v, said %q; want /etc/ws, a write, the copy and the new line, no warning", p.Changes, err, out.Lines)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/h/.zshrc"]); got != "source ~/.config/ws/zshrc\n"+zshSource {
		t.Errorf("~/.zshrc = %q, want the old line kept and the new one added", got)
	}
	if string(m.Files[hookPath]) != string(zsh.Hook) {
		t.Errorf("the hook = %q, want the embedded file", m.Files[hookPath])
	}
}

func TestZshWarnsOfALinkedZshrcThatLacksTheLine(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Dirs["/etc/ws"] = 0o755
	m.Files[hookPath] = zsh.Hook
	m.Files["/h/.zshrc"] = []byte("export EDITOR=vi\n")
	m.Links["/h/.zshrc"] = "/dots/zshrc"
	s := zshStep(m, out)
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || !strings.Contains(strings.Join(out.Lines, "\n"), "a link to /dots/zshrc") {
		t.Errorf("Plan = %v, %v, out %v; want nothing and a warning", p.Changes, err, out.Lines)
	}
	items, _ := s.Report(context.Background())
	if len(items) != 3 || !strings.Contains(items[2].Conflict, "a link to /dots/zshrc") {
		t.Errorf("Report = %+v; want the line's conflict", items)
	}
}

// A report warns of nothing: a plan says the warnings, and a report only reads.
func TestZshReportWarnsNothing(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Files["/h/.zshrc"] = []byte("export EDITOR=vi\n")
	m.Links["/h/.zshrc"] = "/dots/zshrc"
	if _, err := zshStep(m, out).Report(context.Background()); err != nil || len(out.Lines) != 0 {
		t.Errorf("Report = %v, said %q; want no warning", err, out.Lines)
	}
}
