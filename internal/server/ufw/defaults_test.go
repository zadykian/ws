package ufw_test

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/internal/server/ufw"
)

func TestDefaultsSetThePoliciesTheFileLacks(t *testing.T) {
	m := booted()
	m.Files[ufw.DefaultFile] = []byte("# policies\nDEFAULT_INPUT_POLICY=\"ACCEPT\"\nDEFAULT_OUTPUT_POLICY=\"ACCEPT\"\n")
	d := ufw.Defaults{M: m, Out: &effecttest.Out{}, Incoming: "deny", Outgoing: "allow"}
	if got, want := planned(t, d), []string{"ufw default deny incoming"}; !slices.Equal(got, want) {
		t.Errorf("plan = %q, want %q", got, want)
	}
	m.Files[ufw.DefaultFile] = []byte("DEFAULT_INPUT_POLICY=\"DROP\"\nDEFAULT_OUTPUT_POLICY=\"ACCEPT\"\n")
	if got := planned(t, d); got != nil {
		t.Errorf("plan = %q, want none once set", got)
	}
}

func TestDefaultsRefuseAnUnknownPolicy(t *testing.T) {
	d := ufw.Defaults{M: booted(), Out: &effecttest.Out{}, Incoming: "drop"}
	if _, err := d.Plan(t.Context()); err == nil {
		t.Error("the policy drop passed")
	}
}
