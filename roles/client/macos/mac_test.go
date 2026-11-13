package macos_test

import (
	"bytes"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/networks"
	"github.com/zadykian/ws/roles/client/macos/tunnel"
)

func TestNamesComeInSetupShsOrder(t *testing.T) {
	want := []string{"ssh", "cable", "git", "ghostty", "zed", "cld", "zsh", "tunnel", "networks"}
	if got := macos.Names(); !slices.Equal(got, want) {
		t.Errorf("Names = %q; want %q", got, want)
	}
	var got []string
	for _, s := range (macos.Mac{Conf: conf.Conf{}, Home: "/h"}).Rows(macos.Options{Options: step.Options{Check: true}}) {
		got = append(got, s.Name())
	}
	if !slices.Equal(got, want) {
		t.Errorf("Rows = %q; want %q", got, want)
	}
}

func TestRolesFollowNames(t *testing.T) {
	var names []string
	for _, r := range (macos.Mac{Text: io.Discard}).Roles(false) {
		names = append(names, r.Name)
		steps := r.Steps(nil, nil, nil)
		if len(steps) != 1 || steps[0].Name() != r.Name {
			t.Errorf("role %q holds steps %v, want one step of its name", r.Name, steps)
		}
	}
	if !slices.Equal(names, macos.Names()) {
		t.Errorf("roles = %q, want %q", names, macos.Names())
	}
}

func TestMacStepsTakeTheCheck(t *testing.T) {
	for _, check := range []bool{false, true} {
		for _, s := range (macos.Mac{}).Rows(macos.Options{Options: step.Options{Check: check}}) {
			var got bool
			switch s := s.(type) {
			case *tunnel.Tunnel:
				got = s.Check
			case *networks.Networks:
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

// TestMacCldTakesTheCheck reads cld's plan, which words the update by the run's check: with it,
// the versions follow.
func TestMacCldTakesTheCheck(t *testing.T) {
	for check, want := range map[bool]string{
		false: "would run cld update",
		true:  "would run cld update: cld 0.13.0, the latest 0.14.0",
	} {
		m := effecttest.NewMachine()
		m.Modes["/h/.local/bin/cld"], m.Files["/h/.local/bin/cld"] = 0o755, nil
		m.Results["/h/.local/bin/cld --version"] = effecttest.Result{Out: []byte("cld 0.13.0\n")}
		m.Results["/h/.local/bin/cld help update"] = effecttest.Result{}
		m.Results["/h/.local/bin/cld help restore tabs"] = effecttest.Result{Err: errors.New("no")}
		m.Results["brew list --cask ghostty"] = effecttest.Result{Err: errors.New("no")}
		m.Results["curl -fsSIL --retry 3 -o /dev/null -w %{url_effective} https://github.com/zadykian/cld/releases/latest"] =
			effecttest.Result{Out: []byte("https://github.com/zadykian/cld/releases/tag/v0.14.0")}
		mac := macos.Mac{Home: "/h", M: m, Out: &effecttest.Out{}, Text: io.Discard}
		for _, r := range mac.Roles(check) {
			if r.Name != "cld" {
				continue
			}
			p, err := r.Steps(m, nil, mac.Out)[0].Plan(t.Context())
			if err != nil || len(p.Changes) == 0 || p.Changes[0].String() != want {
				t.Errorf("check %v: Plan = %v, %v; want %q first", check, p.Changes, err, want)
			}
		}
	}
}

func TestMacStateReadsUnderACheck(t *testing.T) {
	m := effecttest.NewMachine()
	var text bytes.Buffer
	mac := macos.Mac{Conf: conf.Conf{User: "ada"}, Home: "/h", M: m, Out: &effecttest.Out{}, Text: &text}
	reports, err := mac.State(t.Context())
	if err != nil {
		t.Fatalf("State: %v", err)
	}
	var names []string
	for _, r := range reports {
		names = append(names, r.Step)
	}
	if want := macos.Names(); !slices.Equal(names, want) {
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
