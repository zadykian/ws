package ikeward

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
)

const sysctlNowCall = "/usr/lib/systemd/systemd-sysctl --strict --prefix=net.ipv4.ip_forward"

func forwardMachine(forwarding string) *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Dirs["/run/systemd/system"] = 0o755
	m.Files[ipForward] = []byte(forwarding)
	m.Results[sysctlNowCall] = effecttest.Result{}
	m.Results["rm -rf -- "+sysctlPath] = effecttest.Result{}
	return m
}

// apply plans the group and applies its plan once. The fake's ufw changes no file, so a second
// plan would find the same changes.
func apply(t *testing.T, parts []resource.Resource) {
	g := resource.Group{Title: "forward", Parts: parts}
	p, err := g.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Apply(t.Context(), p); err != nil {
		t.Fatal(err)
	}
}

func TestSysctlWritesTheFileAndTurnsForwardingOn(t *testing.T) {
	m := forwardMachine("0\n")
	apply(t, sysctl(m, true))
	if string(m.Files[sysctlPath]) != string(sysctlFile) {
		t.Errorf("%s = %q, want the embedded file", sysctlPath, m.Files[sysctlPath])
	}
	if !slices.Contains(m.Calls, "root "+sysctlNowCall) {
		t.Errorf("calls = %q, lack the systemd-sysctl", m.Calls)
	}
}

func TestSysctlPlansNothingOnceForwardingIsOn(t *testing.T) {
	m := forwardMachine("1\n")
	m.Files[sysctlPath] = sysctlFile
	g := resource.Group{Title: "sysctl", Parts: sysctl(m, true)}
	if p, err := g.Plan(t.Context()); err != nil || len(p.Changes) != 0 {
		t.Errorf("plan = %v, %v; want nothing", p.Changes, err)
	}
}

func TestSysctlRemovesTheFileWhereNoneIsWanted(t *testing.T) {
	m := forwardMachine("1\n")
	m.Files[sysctlPath] = sysctlFile
	apply(t, sysctl(m, false))
	if !slices.Contains(m.Calls, "root rm -rf -- "+sysctlPath) {
		t.Errorf("calls = %q, lack the removal", m.Calls)
	}
}
