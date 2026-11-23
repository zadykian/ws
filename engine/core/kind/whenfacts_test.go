package kind

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// The facts beside the board's and the GPU's: interface, set and container.

func TestInterfaceHoldsWhereTheLinkExists(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs[netDevices+"/cab0"] = 0o755
	if got, _ := planGated(t, m, `{ interface = "cab0" }`); got == nil {
		t.Error("cab0 exists: plan = none, want the file")
	}
	got, said := planGated(t, m, `{ interface = "cab1" }`)
	if got != nil || !slices.Contains(said, `say: file/a: off on this machine, as its interface is not "cab1"`) {
		t.Errorf("no cab1: plan = %q, said %q; want none, and why", got, said)
	}
}

func TestInterfaceRefusesAPath(t *testing.T) {
	e, _, _, _ := testEnv()
	for want, ok := range map[string]bool{
		"eth0": true, "enp1s0f0np0": true, ".x": true, "...": true, strings.Repeat("a", 15): true, "": true,
		"../eth0": false, ".": false, "..": false, "a b": false, "a:1": false, strings.Repeat("a", 16): false,
	} {
		_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": gatedFile(`{ interface = "` + want + `" }`), "r/a": "x"}))
		if ok != (err == nil) || !ok && !strings.Contains(err.Error(), `when: interface "`+want+`" is not of the form`) {
			t.Errorf("load with interface %q = %v, want loaded %v", want, err, ok)
		}
	}
}

func TestSetHoldsForAnyText(t *testing.T) {
	m := effecttest.NewMachine()
	for _, want := range []string{"/root/vpn/vpnclient", "x", "a b"} {
		if got, said := planGated(t, m, `{ set = "`+want+`" }`); got == nil || said != nil {
			t.Errorf("set %q: plan = %q, said %q; want the file, and no line", want, got, said)
		}
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, " /sys/") }) {
		t.Errorf("calls %q; want no fact read", m.Calls)
	}
}

func TestSetEmptyIsOff(t *testing.T) {
	got, said := planGated(t, effecttest.NewMachine(), `{ set = "" }`)
	if want := []string{"say: file/a: off on this machine, as its when gives no set"}; got != nil || !slices.Equal(said, want) {
		t.Errorf("plan = %q, said %q; want none and %q", got, said, want)
	}
	e, _, _, _ := testEnv()
	_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": gatedFile(`{ set = "!x" }`), "r/a": "x"}))
	if err == nil || !strings.Contains(err.Error(), "holds on no machine") {
		t.Errorf("load with set \"!x\" = %v, want it refused", err)
	}
}

func TestWhenContainerHoldsOnEachMark(t *testing.T) {
	for _, mark := range []string{"/.dockerenv", "/run/.containerenv", "/run/systemd/container"} {
		m := effecttest.NewMachine()
		m.Files[mark] = nil
		if got, _ := planGated(t, m, `{ container = "yes" }`); got == nil {
			t.Errorf("%s: plan = none, want the file", mark)
		}
		got, said := planGated(t, m, `{ container = "!yes" }`)
		if got != nil || !slices.Contains(said, `say: file/a: off on this machine, as its container is "yes"`) {
			t.Errorf("%s with !yes: plan = %q, said %q; want none, and why", mark, got, said)
		}
	}
	if got, _ := planGated(t, effecttest.NewMachine(), `{ container = "!yes" }`); got == nil {
		t.Error("no mark with !yes: plan = none, want the file")
	}
	if got, _ := planGated(t, effecttest.NewMachine(), `{ container = "yes" }`); got != nil {
		t.Errorf("no mark: plan = %q, want none", got)
	}
}

func TestWhenContainerRefusesAnotherWant(t *testing.T) {
	e, _, _, _ := testEnv()
	for _, want := range []string{"no", "true", "!no"} {
		_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": gatedFile(`{ container = "` + want + `" }`), "r/a": "x"}))
		if err == nil || !strings.Contains(err.Error(), `when: container "`+want+`" is not of the form ^yes$`) {
			t.Errorf("load with container %q = %v, want it refused", want, err)
		}
	}
}
