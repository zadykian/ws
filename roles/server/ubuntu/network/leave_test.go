package network

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/roles/server/ubuntu/network/internal/netplan"
)

func TestWithoutSystemdNothingReloads(t *testing.T) {
	r := newRig("enp5s0", "10.77.0.1/30")
	delete(r.m.Dirs, "/run/systemd/system")
	r.u.States = nil // networkd is not asked about, or the run would fail
	if _, err := r.run(t, false); err != nil {
		t.Fatal(err)
	}
	if _, ok := r.m.Files[netplan.Path]; !ok {
		t.Error("the file is not written")
	}
	for _, call := range r.m.Calls {
		if strings.Contains(call, "networkctl") || strings.Contains(call, "netplan generate") {
			t.Errorf("ran %q without systemd", call)
		}
	}
}

func TestNoInterfaceRemovesTheFile(t *testing.T) {
	r := newRig("", "10.77.0.1/30")
	r.m.Files[netplan.Path] = []byte("old")
	if _, err := r.run(t, false); err != nil {
		t.Fatal(err)
	}
	if _, ok := r.m.Files[netplan.Path]; ok {
		t.Error("the file stays")
	}
	if !slices.Contains(r.m.Calls, "root networkctl reload") {
		t.Errorf("no reload in %q", r.m.Calls)
	}
	again := newRig("", "10.77.0.1/30")
	if changed, err := again.run(t, true); err != nil || changed {
		t.Errorf("no file: changed = %v, err = %v", changed, err)
	}
}

func TestAMachineWithoutThePortIsLeftAlone(t *testing.T) {
	r := newRig("enp9s0", "10.77.0.1/30")
	r.m.Files[netplan.Path] = []byte("kept")
	if changed, err := r.run(t, false); err != nil || changed {
		t.Fatalf("changed = %v, err = %v", changed, err)
	}
	if string(r.m.Files[netplan.Path]) != "kept" {
		t.Error("the file changed")
	}
	if len(r.out.Lines) == 0 || !strings.Contains(r.out.Lines[0], "no interface enp9s0") {
		t.Errorf("out = %q", r.out.Lines)
	}
}
