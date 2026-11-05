package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/step"
)

func TestLoginPlan(t *testing.T) {
	r, m := newLogin(effecttest.Result{Out: []byte("ok\n")})
	m.Results[wayCall(r.Host.SSH)] = effecttest.Result{Out: []byte("user root\nhostname 10.1.1.2\nport 22\n" +
		"userknownhostsfile ~/.ssh/known_hosts\n")}
	if p, err := r.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Errorf("Plan where root logs in = %v, %v; want none", p, err)
	}
	m.Results[loginCall] = effecttest.Result{Err: sshFailed("denied\r\nPermission denied\r\n")}
	if p, err := r.Plan(context.Background()); err != nil || len(p.Changes) != 1 {
		t.Errorf("Plan where root is refused = %v, %v; want one change", p, err)
	}
}

func TestLoginStops(t *testing.T) {
	for name, tc := range map[string]struct {
		user, sshErr, want string
	}{
		"refused":  {"root", "denied\r\nPermission denied\r\n", "ssh's error: denied; Permission denied"},
		"changed":  {"root", "Host key verification failed.", "offers another host key than known_hosts"},
		"not root": {"ws", "", "ssh logs in to ws as ws, mac.conf's WS_USER"},
	} {
		r, m := newLogin(effecttest.Result{Err: sshFailed(tc.sshErr)})
		m.Results[wayCall(r.Host.SSH)] = effecttest.Result{Out: []byte("user " + tc.user + "\nhostname 10.1.1.2\nport 22\n" +
			"userknownhostsfile ~/.ssh/known_hosts\n")}
		m.Files["/h/.ssh/id_ed25519.pub"] = []byte("ssh-ed25519 AAAA mac\n")
		err := r.Apply(context.Background(), step.Plan{})
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: Apply = %v; want %q", name, err, tc.want)
		}
	}
}
