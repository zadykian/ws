package claude

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

const (
	claudeBin  = "/root/.local/bin/claude"
	listCmd    = "env HOME=/root env -u CLAUDE_CONFIG_DIR " + claudeBin + " plugin marketplace list --json"
	addCmd     = "env HOME=/root env -u CLAUDE_CONFIG_DIR " + claudeBin + " plugin marketplace add --json "
	acmeSource = "https://git.example.internal/team/plugins.git"
)

var acme = values.Marketplace{Name: "acme", Source: acmeSource}

func TestBare(t *testing.T) {
	for in, want := range map[string]string{
		"https://x/y.git/": "https://x/y", "https://x/y.git#v1": "https://x/y#v1", "a/b": "a/b", "a//": "a",
	} {
		if got := bare(in); got != want {
			t.Errorf("bare(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMarketplacesPlan(t *testing.T) {
	cases := []struct {
		name, list string
		claude     bool
		want       int
		fail       string
	}{
		{"no claude", "", false, 1, ""},
		{"none listed", "", true, 1, ""},
		{"known", `[{"name":"acme","url":"https://git.example.internal/team/plugins"}]`, true, 0, ""},
		{"other source", `[{"name":"acme","repo":"o/r","ref":"v1"}]`, true, 0, "from o/r#v1, not " + acmeSource},
	}
	for _, c := range cases {
		m := effecttest.NewMachine()
		if c.claude {
			m.Files[claudeBin] = nil
			m.Results[listCmd] = effecttest.Result{Out: []byte(c.list)}
		}
		got, err := Marketplaces{M: m, User: root, List: []values.Marketplace{acme}}.Plan(context.Background())
		if (err != nil) != (c.fail != "") || (err != nil && !strings.Contains(err.Error(), c.fail)) {
			t.Errorf("%s: error %v, want %q", c.name, err, c.fail)
		}
		if len(got) != c.want {
			t.Errorf("%s: %d changes, want %d", c.name, len(got), c.want)
		}
	}
}

func TestMarketplaceAddChecksItsName(t *testing.T) {
	for name, tc := range map[string]struct{ out, fail string }{
		"named": {"cloning\n{\"marketplace\":\"acme\"}\n", ""},
		"other": {"{\"marketplace\":\"other\"}\n", "names itself other, not acme"},
	} {
		m := effecttest.NewMachine()
		m.Files[claudeBin] = nil
		m.Results[listCmd] = effecttest.Result{Out: []byte("[]")}
		m.Results[addCmd+acmeSource] = effecttest.Result{Out: []byte(tc.out)}
		p := Marketplaces{M: m, User: root, List: []values.Marketplace{acme}}
		changes, err := p.Plan(context.Background())
		if err != nil || len(changes) != 1 {
			t.Fatalf("%s: %v %v", name, changes, err)
		}
		m.Results[listCmd] = effecttest.Result{Out: []byte(`[{"name":"acme","url":"` + acmeSource + `"}]`)}
		err = changes[0].Apply(context.Background())
		if (err != nil) != (tc.fail != "") || (err != nil && !strings.Contains(err.Error(), tc.fail)) {
			t.Errorf("%s: error %v, want %q", name, err, tc.fail)
		}
	}
}
