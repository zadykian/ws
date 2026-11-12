package setup_test

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/step"
)

func TestNamesComeInSetupShsOrder(t *testing.T) {
	want := []string{"ssh", "cable", "git", "ghostty", "zed", "cld", "zsh", "tunnel", "networks"}
	if got := setup.Names(); !slices.Equal(got, want) {
		t.Errorf("Names = %q; want %q", got, want)
	}
	var got []string
	for _, s := range (setup.Mac{Conf: conf.Conf{}, Home: "/h"}).Rows(setup.Options{Options: step.Options{Check: true}}) {
		got = append(got, s.Name())
	}
	if !slices.Equal(got, want) {
		t.Errorf("Rows = %q; want %q", got, want)
	}
}

func TestMacStepsTakeTheCheck(t *testing.T) {
	for _, check := range []bool{false, true} {
		for _, s := range (setup.Mac{}).Rows(setup.Options{Options: step.Options{Check: check}}) {
			var got bool
			switch s := s.(type) {
			case *setup.Cld:
				got = s.Check
			case *setup.Tunnel:
				got = s.Check
			case *setup.Networks:
				got = s.Check
			default:
				continue
			}
			if got != check {
				t.Errorf("%s: Check = %v; want %v", s.Name(), got, check)
			}
		}
	}
}

func TestMacStateReadsUnderACheck(t *testing.T) {
	m := effecttest.NewMachine()
	var text bytes.Buffer
	mac := setup.Mac{Conf: conf.Conf{User: "ada"}, Home: "/h", M: m, Out: &effecttest.Out{}, Text: &text}
	reports, err := mac.State(t.Context())
	if err != nil {
		t.Fatalf("State: %v", err)
	}
	var names []string
	for _, r := range reports {
		names = append(names, r.Step)
	}
	if want := setup.Names(); !slices.Equal(names, want) {
		t.Errorf("reports of %q; want %q", names, want)
	}
	for _, call := range m.Calls {
		if strings.HasPrefix(call, "write ") || strings.HasPrefix(call, "root ") {
			t.Errorf("State made the call %q; a read makes none that writes", call)
		}
	}
	if text.Len() != 0 {
		t.Errorf("State wrote %q on stdout; want nothing", text.String())
	}
}
