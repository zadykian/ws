package values

import (
	"strings"
	"testing"
)

func goodIkeward() Ikeward {
	return Ikeward{Profile: "/root/vpn/vpnclient", DNS: "10.20.0.53",
		Domains: []string{"example.internal"}, ForwardFrom: []string{"10.99.0.2"}}
}

func TestIkewardRefusesWhatTheRoleCannotUse(t *testing.T) {
	if err := goodIkeward().check(); err != nil {
		t.Fatalf("good values: %v", err)
	}
	for name, tc := range map[string]struct {
		change func(*Ikeward)
		want   string
	}{
		"relative profile": {func(v *Ikeward) { v.Profile = "vpn/vpnclient" }, "not an absolute path"},
		"a quote":          {func(v *Ikeward) { v.Profile = `/a", interface = "eth9` }, "WS_IKEWARD_PROFILE"},
		"a backslash":      {func(v *Ikeward) { v.Profile = `/a\b` }, "a backslash"},
		"a newline":        {func(v *Ikeward) { v.Profile = "/a\nb" }, "a control character"},
		"no resolver":      {func(v *Ikeward) { v.DNS = "" }, "WS_IKEWARD_DNS"},
		"no domain":        {func(v *Ikeward) { v.Domains = nil }, "WS_IKEWARD_DOMAINS"},
		"a peer's prefix":  {func(v *Ikeward) { v.ForwardFrom = []string{"10.99.0.2", "10.99.0.3/32"} }, "10.99.0.3/32"},
		"a peer's name":    {func(v *Ikeward) { v.ForwardFrom = []string{"mac"} }, "mac"},
	} {
		v := goodIkeward()
		tc.change(&v)
		if err := v.check(); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error %v, want one naming %q", name, err, tc.want)
		}
	}
}

func TestIkewardTakesAMacAtSlotOne(t *testing.T) {
	v := goodIkeward()
	v.ForwardFrom = []string{"10.99.1.2"}
	if err := v.check(); err != nil {
		t.Errorf("a Mac at slot 1: %v", err)
	}
}

func TestIkewardUnsetChecksNothing(t *testing.T) {
	if err := (Ikeward{DNS: "", ForwardFrom: []string{"mac"}}).check(); err != nil {
		t.Errorf("check = %v, want none without a profile", err)
	}
}

func TestParseRefusesABadIkeward(t *testing.T) {
	data := "WS_IKEWARD_PROFILE: /root/vpn/vpnclient\nWS_IKEWARD_DOMAINS: [example.internal]\n"
	if _, err := Parse([]byte(data)); err == nil || !strings.Contains(err.Error(), "WS_IKEWARD_DNS") {
		t.Errorf("Parse = %v, want the resolver missing", err)
	}
}
