package host_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
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
