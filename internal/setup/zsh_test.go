package setup_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/mac"
)

func zshStep(m *effecttest.Machine, out *effecttest.Out) *setup.Zsh {
	return &setup.Zsh{Home: "/h", Hook: mac.Zshrc, M: m, Out: out}
}

func TestZshLeavesAHookNotItsOwn(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Dirs["/h/.config"], m.Dirs["/h/.config/ws"] = 0o755, 0o700
	m.Files["/h/.config/ws/zshrc"] = []byte("echo mine\n")
	p, err := zshStep(m, out).Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || !strings.Contains(strings.Join(out.Lines, "\n"), "is not ws's") {
		t.Errorf("Plan = %v, %v, out %v; want nothing and a warning", p.Changes, err, out.Lines)
	}
}
