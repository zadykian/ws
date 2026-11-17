package kind

import (
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// dkmsModulesOn is the load of y, after the unload of x, on a machine booted or not.
func dkmsModulesOn(booted bool) (dkmsModules, *effecttest.Machine, *effecttest.Out) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	if booted {
		m.Dirs["/run/systemd/system"] = 0o755
	}
	return dkmsModules{m: m, out: out, load: "y", unload: []string{"x"}, hint: "Until then, z waits."}, m, out
}

func TestDkmsUnloadsAndLoads(t *testing.T) {
	for _, tc := range []struct {
		name   string
		loaded []string
		want   []string
	}{
		{"neither", nil, []string{"load the module y"}},
		{"the other's", []string{"x"}, []string{"unload the module x", "load the module y"}},
		{"ours", []string{"y"}, nil},
	} {
		d, m, _ := dkmsModulesOn(true)
		for _, n := range tc.loaded {
			m.Dirs["/sys/module/"+n] = 0o755
		}
		changes, err := d.Plan(t.Context())
		var got []string
		for _, c := range changes {
			got = append(got, c.String())
		}
		if err != nil || !slices.Equal(got, tc.want) {
			t.Errorf("%s: plan = %q, %v; want %q", tc.name, got, err, tc.want)
		}
	}
}

func TestDkmsReadsADashAsSysfsWritesIt(t *testing.T) {
	d, m, _ := dkmsModulesOn(true)
	d.load = "y-z"
	m.Dirs["/sys/module/y_z"] = 0o755
	if changes, err := d.Plan(t.Context()); err != nil || len(changes) != 0 {
		t.Errorf("plan = %v, %v; want none, as /sys/module/y_z is y-z", changes, err)
	}
}

func TestDkmsPlansNoLoadAfterARefusal(t *testing.T) {
	d, m, _ := dkmsModulesOn(true)
	m.Files[refusedMark("y")] = nil
	if changes, err := d.Plan(t.Context()); err != nil || len(changes) != 0 {
		t.Errorf("plan = %v, %v; want none after a refusal", changes, err)
	}
}

func TestDkmsLoadSkipsWithoutSystemd(t *testing.T) {
	d, _, out := dkmsModulesOn(false)
	changes, err := d.Plan(t.Context())
	if err != nil || len(changes) != 0 || !slices.Equal(out.Lines, []string{"say: load of y: skipped, systemd is not running"}) {
		t.Errorf("plan = %v, %v, says %q; want none and the skip", changes, err, out.Lines)
	}
}

func TestDkmsNotesSecureBootsRefusal(t *testing.T) {
	d, m, out := dkmsModulesOn(true)
	m.Results["modprobe y"] = effecttest.Result{Err: errors.New("modprobe: ERROR: could not insert 'y': Key was rejected by service")}
	m.Results["touch -- /run/ws-refused-y"] = effecttest.Result{}
	want := []string{"say: the kernel refused y, as Secure Boot is on and DKMS's key is not enrolled. Until then, z waits."}
	if err := (loadModule{d}).Apply(t.Context()); err != nil || !slices.Equal(out.Lines, want) ||
		!slices.Contains(m.Calls, "root touch -- /run/ws-refused-y") {
		t.Errorf("apply = %v, says %q, ran %q; want the note, the mark and no error", err, out.Lines, m.Calls)
	}
	m.Results["modprobe y"] = effecttest.Result{Err: errors.New("modprobe: FATAL: Module y not found")}
	if err := (loadModule{d}).Apply(t.Context()); err == nil {
		t.Error("apply with another error = nil, want it")
	}
}

func TestDkmsUnitLoadsAfterTheBuild(t *testing.T) {
	e, m, _, body := dkmsMachine(t)
	m.Dirs["/run/systemd/system"] = 0o755
	units := loadOne(t, e, dkmsText(body, map[string]string{"load": `"y"`, "unload": `["x"]`}),
		map[string]string{"dkms.conf": dkmsConf})
	got := headlines(t, units[0].Step)
	if i := slices.Index(got, "build and install m/1 for "+dkmsKernel); i < 0 || !slices.Equal(got[i+1:], []string{"load the module y"}) {
		t.Errorf("plan = %q, want the build, then the load", got)
	}
}
