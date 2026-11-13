package settings

import (
	"testing"

	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

func TestSettingsCheckIsAPlanError(t *testing.T) {
	if _, err := (Check{values.Wireguard{Address: "x", Port: 1}}).Plan(t.Context()); err == nil {
		t.Error("Plan = nil, want the refusal")
	}
	w := values.Wireguard{Address: "10.99.0.1/24", Port: 51820}
	if cs, err := (Check{w}).Plan(t.Context()); err != nil || len(cs) != 0 {
		t.Errorf("Plan = %v, %v; want nothing", cs, err)
	}
}
