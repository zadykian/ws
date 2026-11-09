package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

const releases = "https://example.org/releases"

func newInstall(version, url string, status wsStatus) (*WsInstall, *effecttest.Machine) {
	r, m := newRun("lan", status, false, false)
	return &WsInstall{Remote: r.Remote, Version: version, URL: url, Out: &effecttest.Out{}}, m
}

func TestWsInstallPlan(t *testing.T) {
	for name, tc := range map[string]struct{ version, url, held, want, err string }{
		"done":          {"v1.2.3", releases, "v1.2.3", "", ""},
		"older":         {"v1.2.3", releases, "v1.2.2", "would install ws v1.2.3 on ws", ""},
		"newer":         {"v1.2.3", releases, "v1.3.0", "", "ws mac wizard --update"},
		"newer 10":      {"v1.2.3", releases, "v1.10.0", "", "ws mac wizard --update"},
		"a build on ws": {"v1.2.3", releases, "v1.2.4-0.2026-abc+dirty", "would install ws v1.2.3 on ws", ""},
		"none":          {"v1.2.3", releases, "", "would install ws v1.2.3 on ws", ""},
		"no status":     {"v1.2.3", releases, "v1.2.3", "would install ws v1.2.3 on ws", ""},
		"a build":       {"v1.2.4-0.20261030-abc+dirty", releases, "v1.2.3", "", "not a release"},
		"devel":         {"devel", releases, "", "", "not a release"},
		"an address":    {"v1.2.3", "http://example.org", "", "", "must start with https://"},
	} {
		i, m := newInstall(tc.version, tc.url, wsStatus{Version: tc.held})
		if name == "no status" {
			delete(m.Results, remoteLine(i.Remote, "status"))
		}
		p, err := i.Plan(context.Background())
		got := ""
		if len(p.Changes) > 0 {
			got = p.Changes[0].String()
		}
		if (err == nil) != (tc.err == "") || (err != nil && !strings.Contains(err.Error(), tc.err)) ||
			!strings.Contains(got, tc.want) || (got == "") != (tc.want == "") {
			t.Errorf("%s: Plan = %q, %v; want %q, error %q", name, got, err, tc.want, tc.err)
		}
	}
}
