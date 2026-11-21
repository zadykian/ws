package cli

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/values"
)

// The board and the kernel of the fans' tests, and what the role's spec pins.
const (
	fansBoard   = "Micro-Star International Co., Ltd."
	fansKernel  = "7.0.0-38-generic"
	fansPin     = "nct6687d/7e3d7b36c55f"
	fansTree    = "/usr/src/nct6687d-7e3d7b36c55f"
	fansURL     = "https://github.com/Fred78290/nct6687d/archive/7e3d7b36c55f06ab5bb2077bc85329f592996b2a.tar.gz"
	fansInForce = "sh -c readlink /proc/$(systemctl show -P MainPID ws-fans.service)/exe"
	fansLoads   = "readlink -f /usr/local/bin/ws"
)

// fansUnits are the role's units, in the order they run.
var fansUnits = []string{"packages/fans", "dkms/fans", "service/fans"}

// withBoard gives the machine a board whose maker DMI names maker.
func withBoard(m *effecttest.Machine, maker string) {
	m.Files["/sys/class/dmi/id/board_vendor"] = []byte(maker + "\n")
}

// fansMachine is a machine on the board, running the kernel, with the directories of /etc
// that Ubuntu ships.
func fansMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	for _, d := range []string{"/etc", "/etc/modprobe.d", "/etc/systemd", "/etc/systemd/system"} {
		m.Dirs[d] = 0o755
	}
	withBoard(m, fansBoard)
	m.Results["uname -r"] = effecttest.Result{Out: []byte(fansKernel + "\n")}
	return m
}

// fansValues is the values of an empty file.
func fansValues(t *testing.T) values.Values {
	v, err := values.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// fansSpecs loads the specs with the values v on m, and returns each unit's step by its name.
func fansSpecs(t *testing.T, m *effecttest.Machine, u *effecttest.Units, out *effecttest.Out, v values.Values) map[string]step.Step {
	steps := map[string]step.Step{}
	for _, unit := range loadSpecs(t, kind.Env{M: m, U: u, Out: out, Data: v, Self: selfPath}) {
		steps[unit.Name] = unit.Step
	}
	return steps
}

// apply plans and applies the unit name of steps.
func apply(t *testing.T, steps map[string]step.Step, name string) {
	p, err := steps[name].Plan(t.Context())
	if err != nil {
		t.Fatalf("plan %s: %v", name, err)
	}
	if err := steps[name].Apply(t.Context(), p); err != nil {
		t.Fatalf("apply %s: %v", name, err)
	}
}

func TestFansUnitsRunInOrder(t *testing.T) {
	units := loadSpecs(t, kind.Env{M: fansMachine(), U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: fansValues(t), Self: selfPath})
	names := kind.Names(units)
	for _, pair := range [][2]string{{"packages/fans", "dkms/fans"}, {"state/ws-binary", "service/fans"}} {
		first, then := slices.Index(names, pair[0]), slices.Index(names, pair[1])
		if first < 0 || then < first {
			t.Errorf("%s runs at %d, %s at %d; want the first before", pair[0], first, pair[1], then)
		}
		if got := units[then].Step.WaitsOn(); got != pair[0] {
			t.Errorf("%s waits on %q, want %s", pair[1], got, pair[0])
		}
	}
	// ws-fans drives nct6687's files and reads the GPU by nvidia-smi, so it starts after both,
	// which its one after cannot name: the names' order keeps them first.
	service := slices.Index(names, "service/fans")
	for _, first := range []string{"dkms/fans", "packages/nvidia"} {
		if i := slices.Index(names, first); i < 0 || i > service {
			t.Errorf("%s runs at %d, service/fans at %d; want it first", first, i, service)
		}
	}
}

func TestFansOffOnAnotherBoard(t *testing.T) {
	another := `as its board_vendor is not "` + fansBoard + `"`
	for _, tc := range []struct {
		name, maker, vendor, why string
	}{
		{"QEMU's board", "QEMU", fansBoard, another},
		{"no DMI file", "", fansBoard, another},
		{"no vendor named", fansBoard, "", "as its when gives no board_vendor"},
	} {
		m, out := fansMachine(), &effecttest.Out{}
		delete(m.Files, "/sys/class/dmi/id/board_vendor")
		if tc.maker != "" {
			withBoard(m, tc.maker)
		}
		v := fansValues(t)
		v.Fans.BoardVendor = tc.vendor
		steps := fansSpecs(t, m, &effecttest.Units{}, out, v)
		var want []string
		for _, unit := range fansUnits {
			if got, err := planned(t, steps, unit); err != nil || got != nil {
				t.Errorf("%s: %s plans %q, %v; want none", tc.name, unit, got, err)
			}
			want = append(want, "say: "+unit+": off on this machine, "+tc.why)
		}
		if !slices.Equal(out.Lines, want) {
			t.Errorf("%s: said %q, want %q", tc.name, out.Lines, want)
		}
		if i := slices.IndexFunc(m.Calls, func(c string) bool {
			return strings.Contains(c, "uname") || strings.Contains(c, "dkms") || strings.Contains(c, "modprobe")
		}); i >= 0 {
			t.Errorf("%s: ran %q off the board", tc.name, m.Calls[i])
		}
	}
}

func TestFansPackagesNameTheRunningKernelsHeaders(t *testing.T) {
	m := fansMachine()
	m.Packages = effecttest.Packages{}
	got, err := planned(t, specUnits(t, m, &effecttest.Out{}), "packages/fans")
	if want := []string{"install dkms linux-headers-generic linux-headers-" + fansKernel}; err != nil || !slices.Equal(got, want) {
		t.Errorf("plan = %q, %v; want %q", got, err, want)
	}
}

func TestFansDriverIsThePinnedSource(t *testing.T) {
	got, err := planned(t, specUnits(t, fansMachine(), &effecttest.Out{}), "dkms/fans")
	if err != nil || len(got) < 6 {
		t.Fatalf("plan = %q, %v; want the source, its conf, the build and the load", got, err)
	}
	if want := "install " + fansURL + " in " + fansTree; got[2] != want {
		t.Errorf("plan = %q, want its third %q", got, want)
	}
	if want := []string{"add " + fansPin + " to DKMS", "build and install " + fansPin + " for " + fansKernel,
		"load the module nct6687"}; !slices.Equal(got[len(got)-3:], want) {
		t.Errorf("plan = %q, want it to end %q", got, want)
	}
	m := fansMachine()
	m.Files[fansTree+"/.ws-source"] = []byte(fansURL + "\n")
	m.Dirs["/sys/module/nct6683"] = 0o755
	for _, c := range []string{"dkms add " + fansPin, "dkms install " + fansPin + " -k " + fansKernel,
		"modprobe -r nct6683", "modprobe nct6687"} {
		m.Results[c] = effecttest.Result{}
	}
	apply(t, specUnits(t, m, &effecttest.Out{}), "dkms/fans")
	if conf := string(m.Files[fansTree+"/dkms.conf"]); !strings.Contains(conf, "PACKAGE_VERSION=\"7e3d7b36c55f\"\n") {
		t.Errorf("dkms.conf = %q, want the pin's version", conf)
	}
	var ran []string
	for _, c := range m.Calls {
		if strings.HasPrefix(c, "root dkms ") || strings.HasPrefix(c, "root modprobe ") {
			ran = append(ran, c)
		}
	}
	if want := []string{"root dkms add " + fansPin, "root dkms install " + fansPin + " -k " + fansKernel,
		"root modprobe -r nct6683", "root modprobe nct6687"}; !slices.Equal(ran, want) {
		t.Errorf("ran %q, want %q", ran, want)
	}
}

// fansService applies service/fans with the defaults on m, and returns systemd's fake.
func fansService(t *testing.T, m *effecttest.Machine) *effecttest.Units {
	u := &effecttest.Units{}
	apply(t, fansSpecs(t, m, u, &effecttest.Out{}, fansValues(t)), "service/fans")
	return u
}

func TestFansServiceInstallsEnablesAndStartsThenChangesNothing(t *testing.T) {
	m := fansMachine()
	u := fansService(t, m)
	config, err := values.Default().Fans.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/etc/ws/fans.json"]); got != config+"\n" {
		t.Errorf("fans.json = %q, want the values' config and a newline", got)
	}
	if !strings.Contains(string(m.Files["/etc/systemd/system/ws-fans.service"]),
		"\nExecStart=/usr/local/bin/ws server fans /etc/ws/fans.json\n") {
		t.Errorf("ws-fans.service runs no /usr/local/bin/ws: %q", m.Files["/etc/systemd/system/ws-fans.service"])
	}
	for _, p := range []string{"/etc/modprobe.d/ws-fans.conf", "/etc/modules-load.d/ws-fans.conf"} {
		if m.Files[p] == nil {
			t.Errorf("%s is not written", p)
		}
	}
	for _, d := range []string{"/etc/ws", "/etc/modules-load.d"} {
		if m.Dirs[d] != 0o755 {
			t.Errorf("%s has the mode %o, want 755", d, m.Dirs[d])
		}
	}
	if got := u.States["ws-fans.service"]; !got.Enabled || !got.Active || !slices.Contains(u.Calls, "daemon-reload") {
		t.Errorf("ws-fans = %+v, calls %q; want systemd reloaded, and the unit enabled and running", got, u.Calls)
	}
	m.Results[fansInForce] = effecttest.Result{Out: []byte("/usr/local/bin/ws\n")}
	m.Results[fansLoads] = effecttest.Result{Out: []byte("/usr/local/bin/ws\n")}
	steps := fansSpecs(t, m, u, &effecttest.Out{}, fansValues(t))
	if got, err := planned(t, steps, "service/fans"); err != nil || got != nil {
		t.Errorf("second plan = %q, %v; want none", got, err)
	}
}

func TestFansServiceRestartsOnANewWs(t *testing.T) {
	m := fansMachine()
	u := fansService(t, m)
	u.Calls = nil
	m.Results[fansInForce] = effecttest.Result{Out: []byte("/usr/local/bin/ws (deleted)\n")}
	m.Results[fansLoads] = effecttest.Result{Out: []byte("/usr/local/bin/ws\n")}
	apply(t, fansSpecs(t, m, u, &effecttest.Out{}, fansValues(t)), "service/fans")
	if !slices.Contains(u.Calls, "restart ws-fans.service") {
		t.Errorf("calls = %q, want a restart on the new ws", u.Calls)
	}
}

func TestFansConfigRefusedFailsThePlan(t *testing.T) {
	m := fansMachine()
	m.Files["/etc/ws/fans.json"] = []byte("old\n")
	v := fansValues(t)
	v.Fans.Curves[0].Sensor = "none"
	if _, err := planned(t, fansSpecs(t, m, &effecttest.Units{}, &effecttest.Out{}, v), "service/fans"); err == nil {
		t.Error("plan with a curve on a missing sensor = nil, want the loader's refusal")
	}
	if got := string(m.Files["/etc/ws/fans.json"]); got != "old\n" {
		t.Errorf("fans.json = %q, want the old one kept", got)
	}
}
