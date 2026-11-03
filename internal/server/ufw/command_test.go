package ufw_test

import (
	"testing"

	"github.com/zadykian/ws/internal/server/ufw"
)

func TestRuleStringIsTheCommandLine(t *testing.T) {
	r := ufw.Rule{Action: "allow", Proto: "tcp", ToPort: "22", To: ufw.Any, FromPort: "any",
		From: "10.0.0.0/8", Dir: "in", Interface: "eth0", Comment: "ssh"}
	if got, want := r.String(), "allow in on eth0 from 10.0.0.0/8 to any port 22 proto tcp"; got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
}

func TestValidRefusesAnIncompleteRule(t *testing.T) {
	ok := ufw.Rule{Action: "allow", Proto: "any", ToPort: "53", To: "1.2.3.4", FromPort: "any",
		From: ufw.Any, Dir: "in", Comment: "c"}
	if err := ok.Valid(); err != nil {
		t.Errorf("complete rule: %v", err)
	}
	for name, bad := range map[string]ufw.Rule{
		"direction": func() ufw.Rule { r := ok; r.Dir = "up"; return r }(),
		"comment":   func() ufw.Rule { r := ok; r.Comment = ""; return r }(),
		"field":     func() ufw.Rule { r := ok; r.To = ""; return r }(),
	} {
		if bad.Valid() == nil {
			t.Errorf("%s: rule %+v passed", name, bad)
		}
	}
}

func TestRouteIsTheRouteCommand(t *testing.T) {
	r := ufw.Route("allow", "wg0", "ikeward0", "10.99.0.2", "ikeward")
	if got, want := r.String(), "route allow in on wg0 out on ikeward0 from 10.99.0.2 to any"; got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
	if err := r.Valid(); err != nil {
		t.Error(err)
	}
}
