package host

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/remote"
)

func TestLoginNamesTheServer(t *testing.T) {
	r, m := newLogin(effecttest.Result{})
	r.SSH.Host = "lab"
	m.Files["/h/mac.conf"] = append(m.Files["/h/mac.conf"], "WS_IDENTITY_PUB_KEY=~/.ssh/id_ed25519.pub\n"...)
	m.Files["/h/.ssh/id_ed25519.pub"] = []byte("ssh-ed25519 AAAA mac\n")
	m.Results["ssh-keygen -l -f /h/.ssh/id_ed25519.pub"] = effecttest.Result{}
	w := remote.Way{Host: "10.1.1.2", Port: "22", User: "ada"}
	for name, tc := range map[string]struct {
		err, sshErr, want string
		check             bool
	}{
		"changed": {"", "Host key verification failed.", "lab at 10.1.1.2:22 offers another host key than known_hosts holds under lab: where lab was installed again, run ssh-keygen -R lab", false},
		"refused": {"", "denied", "ada on lab doesn't log in by the Mac's key, /h/.ssh/id_ed25519.pub, and the wizard adds no key to lab. lab takes its keys", false},
		"checked": {"", "denied", "ada on lab doesn't log in by the Mac's key; a run would stop, as the wizard adds no key to lab. ssh's error: denied", true},
	} {
		r.Check = tc.check
		err := r.refuse(context.Background(), w, tc.sshErr)
		got := strings.Join(r.Out.(*effecttest.Out).Lines, "\n")
		if err != nil {
			got = err.Error()
		}
		if !strings.Contains(got, tc.want) {
			t.Errorf("%s: refuse = %q; want %q", name, got, tc.want)
		}
	}
	if got := (loginStop{"lab"}).String(); !strings.HasSuffix(got, "would stop, as WS_USER on lab refuses the Mac's key") {
		t.Errorf("loginStop = %q; want lab", got)
	}
}

func TestMacReachClaudeNamesTheServer(t *testing.T) {
	m := effecttest.NewMachine()
	out := &effecttest.Out{}
	r := MacReach{SSH: remote.SSH{Host: "lab", M: m, F: m, Lock: "/l"}, Conf: "/h/mac.conf", Out: out}
	err := r.Claude(context.Background())
	said := strings.Join(out.Lines, "\n")
	if err == nil || !strings.Contains(err.Error(), "WS_USER's claude on lab still has no login") ||
		!strings.Contains(said, "claude auth login runs as WS_USER on lab: take its steps") {
		t.Errorf("Claude = %v, said %q; want lab", err, said)
	}
}
