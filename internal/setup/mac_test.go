package setup_test

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/internal/setup"
)

func TestNamesComeInSetupShsOrder(t *testing.T) {
	want := []string{"ssh", "cable", "git", "ghostty", "zed", "cld", "zsh", "tunnel", "networks"}
	if got := setup.Names(); !slices.Equal(got, want) {
		t.Errorf("Names = %q; want %q", got, want)
	}
	var got []string
	for _, s := range (setup.Mac{Conf: conf.Conf{}, Home: "/h"}).Steps(false) {
		got = append(got, s.Name())
	}
	if !slices.Equal(got, want) {
		t.Errorf("Steps = %q; want %q", got, want)
	}
}

func TestMacStepsTakeTheCheck(t *testing.T) {
	for _, check := range []bool{false, true} {
		for _, s := range (setup.Mac{}).Steps(check) {
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
