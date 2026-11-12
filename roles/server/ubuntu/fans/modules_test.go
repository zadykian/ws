package fans

import (
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestModulesUnloadTheBoardsOwnAndLoadOurs(t *testing.T) {
	for _, tc := range []struct {
		name   string
		loaded []string
		want   []string
	}{
		{"neither", nil, []string{"load nct6687"}},
		{"linux's own", []string{"nct6683"}, []string{"unload Linux's own driver of the chip, nct6683", "load nct6687"}},
		{"ours", []string{"nct6687"}, nil},
	} {
		m := effecttest.NewMachine()
		m.Dirs["/run/systemd/system"] = 0o755
		for _, n := range tc.loaded {
			m.Dirs["/sys/module/"+n] = 0o755
		}
		changes, err := modules{m, &effecttest.Out{}}.Plan(t.Context())
		var got []string
		for _, c := range changes {
			got = append(got, c.String())
		}
		if err != nil || !slices.Equal(got, tc.want) {
			t.Errorf("%s: Plan = %q, %v; want %q", tc.name, got, err, tc.want)
		}
	}
}

func TestModulesPlanNoLoadAfterARefusal(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/run/systemd/system"] = 0o755
	m.Files[refusedMark] = nil
	if changes, err := (modules{m, &effecttest.Out{}}).Plan(t.Context()); err != nil || len(changes) != 0 {
		t.Errorf("Plan = %v, %v; want none after a refusal", changes, err)
	}
}

func TestModulesSkipWithoutSystemd(t *testing.T) {
	out := &effecttest.Out{}
	if changes, err := (modules{effecttest.NewMachine(), out}).Plan(t.Context()); err != nil || len(changes) != 0 || len(out.Lines) != 1 {
		t.Errorf("Plan = %v, %v; says %q; want none and a note", changes, err, out.Lines)
	}
}

func TestLoadNotesSecureBootsRefusalAndFailsOnOthers(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Results["modprobe nct6687"] = effecttest.Result{Err: errors.New("modprobe: ERROR: could not insert 'nct6687': Key was rejected by service")}
	m.Results["touch -- "+refusedMark] = effecttest.Result{}
	if err := (load{modules{m, out}}).Apply(t.Context()); err != nil || len(out.Lines) != 1 || !slices.Contains(m.Calls, "root touch -- "+refusedMark) {
		t.Errorf("Apply = %v, says %q, ran %q; want a note, the mark and no error", err, out.Lines, m.Calls)
	}
	m.Results["modprobe nct6687"] = effecttest.Result{Err: errors.New("modprobe: FATAL: Module nct6687 not found")}
	if err := (load{modules{m, out}}).Apply(t.Context()); err == nil {
		t.Error("Apply with another error = nil, want it")
	}
}
