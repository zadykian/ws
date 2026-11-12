package host_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/host"
)

// planThenLink plans each part on a home with no link, then puts a link at ~/.config, as the user may
// between Plan and Apply. It returns the changes and the number of calls made so far.
func planThenLink(t *testing.T, m *effecttest.Machine, parts ...host.Resource) ([]host.Change, int) {
	var all []host.Change
	for _, p := range parts {
		changes, err := p.Plan(t.Context())
		if err != nil || len(changes) != 1 {
			t.Fatalf("Plan = %v, %v; want one change", changes, err)
		}
		all = append(all, changes...)
	}
	m.Links["/home/ws/.config"] = "/srv/target"
	return all, len(m.Calls)
}

// Each change fails at the link before it acts: Stat is the only call that follows.
func TestHomePartsRefuseALinkMadeAfterPlan(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/home/ws/.config/old"] = []byte("x")
	edit := account.File(m, ".config/old", []byte("y"), 0o644)
	edit.Backup = true
	changes, planned := planThenLink(t, m,
		account.File(m, ".config/f", []byte("a"), 0o644),
		edit,
		host.Validated{
			File:  account.File(m, ".config/v", []byte("a"), 0o644),
			Check: func(p string) (string, []string) { return "check", []string{p} },
		},
		account.Dir(m, ".config/d", 0o700),
	)
	for _, c := range changes {
		if err := c.Apply(t.Context()); err == nil || !strings.Contains(err.Error(), "/home/ws/.config is a link to") {
			t.Errorf("%s: Apply = %v; want the link refused", c, err)
		}
	}
	for _, c := range m.Calls[planned:] {
		if !strings.HasPrefix(c, "stat ") {
			t.Errorf("a call after the link: %s", c)
		}
	}
}

// A removal under the home is root's rm, which resolves each directory on the way.
func TestHomeAbsentRefusesALink(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/home/ws/.config/old"] = []byte("x")
	m.Results["rm -rf -- /home/ws/.config/old"] = effecttest.Result{}
	if got := applyAll(t, account.Absent(m, ".config/old")); len(got) != 1 || got[0] != "remove /home/ws/.config/old" {
		t.Fatalf("plan = %q, want the removal", got)
	}
	m.Files["/home/ws/.config/old"] = []byte("x")
	gone := account.Absent(m, ".config/old")
	changes, planned := planThenLink(t, m, gone)
	if err := changes[0].Apply(t.Context()); err == nil || !strings.Contains(err.Error(), "is a link to") {
		t.Errorf("Apply = %v; want the link refused", err)
	}
	if calls := m.Calls[planned:]; slices.ContainsFunc(calls, func(c string) bool { return !strings.HasPrefix(c, "stat ") }) {
		t.Errorf("calls after the link: %q", calls)
	}
	if _, err := gone.Plan(t.Context()); err == nil {
		t.Error("Plan found no error")
	}
	if _, err := gone.Report(t.Context()); err == nil {
		t.Error("Report found no error")
	}
}
