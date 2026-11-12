package ufw_test

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/ufw"
)

func TestDefaultsRunUfw(t *testing.T) {
	m := booted()
	m.Results["ufw default allow outgoing"] = effecttest.Result{}
	changes, err := ufw.Defaults{M: m, Out: &effecttest.Out{}, Outgoing: "allow"}.Plan(t.Context())
	if err != nil || len(changes) != 1 || changes[0].Apply(t.Context()) != nil {
		t.Fatalf("changes = %v, %v", changes, err)
	}
	if !slices.Contains(m.Calls, "root ufw default allow outgoing") {
		t.Errorf("calls = %q, want ufw default allow outgoing", m.Calls)
	}
}

func TestDefaultsAndEnableSkipWithoutSystemd(t *testing.T) {
	out := &effecttest.Out{}
	m := effecttest.NewMachine()
	for _, r := range []host.Resource{ufw.Defaults{M: m, Out: out, Incoming: "deny"}, ufw.Enable{M: m, Out: out}} {
		if got := planned(t, r); got != nil {
			t.Errorf("%T planned %q without systemd", r, got)
		}
	}
	if len(out.Lines) != 2 {
		t.Errorf("said %q, want two skips", out.Lines)
	}
}

func TestEnableRunsUfw(t *testing.T) {
	m := booted()
	m.Results["ufw --force enable"] = effecttest.Result{}
	changes, err := ufw.Enable{M: m, Out: &effecttest.Out{}}.Plan(t.Context())
	if err != nil || len(changes) != 1 || changes[0].Apply(t.Context()) != nil {
		t.Fatalf("changes = %v, %v", changes, err)
	}
	if !slices.Contains(m.Calls, "root ufw --force enable") {
		t.Errorf("calls = %q, want ufw --force enable", m.Calls)
	}
}
