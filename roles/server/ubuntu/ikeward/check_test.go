package ikeward

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

func good() values.Ikeward {
	return values.Ikeward{Profile: "/root/vpn/vpnclient", DNS: "10.20.0.53",
		Domains: []string{"example.internal"}, ForwardFrom: []string{"10.99.0.2"}}
}

func TestCheckRefusesWhatTheRoleCannotUse(t *testing.T) {
	if err := check(good()); err != nil {
		t.Fatalf("good values: %v", err)
	}
	for name, tc := range map[string]struct {
		change func(*values.Ikeward)
		want   string
	}{
		"relative profile": {func(v *values.Ikeward) { v.Profile = "vpn/vpnclient" }, "not an absolute path"},
		"no resolver":      {func(v *values.Ikeward) { v.DNS = "" }, "ikeward_dns"},
		"no domain":        {func(v *values.Ikeward) { v.Domains = nil }, "ikeward_domains"},
		"a peer's prefix":  {func(v *values.Ikeward) { v.ForwardFrom = []string{"10.99.0.2", "10.99.0.3/32"} }, "10.99.0.3/32"},
		"a peer's name":    {func(v *values.Ikeward) { v.ForwardFrom = []string{"mac"} }, "mac"},
	} {
		v := good()
		tc.change(&v)
		if err := check(v); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error %v, want one naming %q", name, err, tc.want)
		}
	}
}

func TestCheckTakesAMacAtSlotOne(t *testing.T) {
	v := good()
	v.ForwardFrom = []string{"10.99.1.2"}
	if err := check(v); err != nil {
		t.Errorf("a Mac at slot 1: %v", err)
	}
}
