package plan

import (
	"testing"

	"github.com/zadykian/ws/engine/core/host/account"
)

func TestBelowCleansThePathAndTheHome(t *testing.T) {
	a := account.Account{User: account.User{Name: "ws", Home: "/home/ws/"}}
	for p, want := range map[string]string{
		"/home/ws": "", "/home/ws//.ssh/k": ".ssh/k", "/home/ws/a/../b": "b", "/etc/k": "/etc/k",
	} {
		if got := Below(a, p); got != want {
			t.Errorf("Below(%q) = %q, want %q", p, got, want)
		}
	}
}
