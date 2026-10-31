package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

const cableLink = "eth0 - -\neth1 1 -\n"

func newCable(t *testing.T, status, link string, env map[string]string) (*Cable, *effecttest.Machine, *effecttest.Prompt) {
	root, m := newRoot(effecttest.Result{Out: []byte("ok\n")})
	r := &WsRun{Host: root.Host, Remote: &Remote{SSH: root.Host.SSH, Script: []byte("x")}}
	m.Results[remoteLine(r.Remote, "status")] = effecttest.Result{Out: []byte("run abc 0 on\nactive -\n" + status)}
	m.Results["mkdir -p /h"] = effecttest.Result{}
	files := &Remote{SSH: r.Host.SSH, Script: []byte("f")}
	m.Results[remoteLine(files, "link")] = effecttest.Result{Out: []byte(link)}
	p := &effecttest.Prompt{T: t, Answers: map[string]string{}}
	c := &Cable{Host: r.Host, Run: r, Root: root, Files: files, Setup: &Setup{Run: func(context.Context, bool) (bool, error) { return false, nil }},
		Env: func(k string) (string, bool) { v, ok := env[k]; return v, ok }, P: p, Tty: true, M: m, Out: r.Host.Out}
	return c, m, p
}

func TestCablePlan(t *testing.T) {
	for name, tc := range map[string]struct {
		conf, status, link, want string
		env                      map[string]string
	}{
		"done":       {"WS_CABLE_SERVICE=\n", "", cableLink, "", nil},
		"named":      {"WS_CABLE_PORT=eth1\nWS_CABLE_SERVICE=\"USB\"\n", "cable eth1\n", cableLink, "", nil},
		"another":    {"WS_CABLE_PORT=eth1\nWS_CABLE_SERVICE=\"USB\"\n", "cable eth0\n", cableLink, "again", nil},
		"undocked":   {"", "", "eth0 - -\n", "", nil},
		"forced":     {"", "", "eth0 - -\n", "ask ws's port", map[string]string{"WS_CABLE_PORT": "eth0"}},
		"docked":     {"", "", cableLink, "ask ws's port", nil},
		"has end":    {"", "", "eth1 1 10.77.0.1/24\n", "ask the Mac's service", nil},
		"none given": {"", "", cableLink, "empty WS_CABLE_SERVICE", map[string]string{"WS_CABLE_SERVICE": ""}},
	} {
		c, m, _ := newCable(t, tc.status, tc.link, tc.env)
		m.Files["/h/mac.conf"] = []byte("WS_TARGET=lan\n" + tc.conf)
		p, err := c.Plan(context.Background())
		got := errText(err)
		for _, ch := range p.Changes {
			got += ch.String()
		}
		if !strings.Contains(got, tc.want) || (tc.want == "" && got != "") {
			t.Errorf("%s: Plan = %q; want %q", name, got, tc.want)
		}
	}
}
