package ikeward

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func forwardMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Results["rm -rf -- "+sysctlPath] = effecttest.Result{}
	return m
}

// apply plans the group and applies its plan once. The fake's ufw changes no file, so a second
// plan would find the same changes.
func apply(t *testing.T, parts []host.Resource) {
	g := host.Group{Title: "forward", Parts: parts}
	p, err := g.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Apply(t.Context(), p); err != nil {
		t.Fatal(err)
	}
}

func TestSysctlWritesTheFile(t *testing.T) {
	m := forwardMachine()
	apply(t, sysctl(m, true))
	if string(m.Files[sysctlPath]) != string(sysctlFile) {
		t.Errorf("%s = %q, want the embedded file", sysctlPath, m.Files[sysctlPath])
	}
}

func TestSysctlRemovesTheFileWhereNoneIsWanted(t *testing.T) {
	m := forwardMachine()
	m.Files[sysctlPath] = sysctlFile
	apply(t, sysctl(m, false))
	if !slices.Contains(m.Calls, "root rm -rf -- "+sysctlPath) {
		t.Errorf("calls = %q, lack the removal", m.Calls)
	}
}
