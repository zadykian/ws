package helix

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

const roslynStore = "/root/.dotnet/tools/.store/roslyn-language-server"

func newRoslyn(m *effecttest.Machine) Roslyn {
	return Roslyn{M: m, User: host.User{Name: "root", Home: "/root"},
		Path: "/root/.dotnet/tools/roslyn-language-server", Version: "5.12.0-1.26475.2"}
}

func TestRoslynPlan(t *testing.T) {
	cases := []struct {
		name string
		dirs []string
		want string // the change's text, or empty for none
	}{
		{"current", []string{"5.12.0-1.26475.2"}, ""},
		{"missing", nil, "install roslyn-language-server 5.12.0-1.26475.2"},
		{"older", []string{"5.11.0"}, "replace roslyn-language-server 5.11.0 with 5.12.0-1.26475.2"},
		{"two", []string{"5.11.0", "5.12.0-1.26475.2"},
			"replace roslyn-language-server 5.11.0, 5.12.0-1.26475.2 with 5.12.0-1.26475.2"},
	}
	for _, c := range cases {
		m := effecttest.NewMachine()
		for _, d := range c.dirs {
			m.Dirs[roslynStore+"/"+d] = 0o755
		}
		if len(c.dirs) > 0 {
			m.Dirs[roslynStore] = 0o755
		}
		got, err := newRoslyn(m).Plan(context.Background())
		switch {
		case err != nil:
			t.Errorf("%s: %v", c.name, err)
		case c.want == "" && len(got) != 0, c.want != "" && (len(got) != 1 || got[0].String() != c.want):
			t.Errorf("%s: changes %v, want %q", c.name, got, c.want)
		}
	}
}

func TestRoslynInstallsFromNuGetAlone(t *testing.T) {
	m := effecttest.NewMachine()
	r := newRoslyn(m)
	changes, err := r.Plan(context.Background())
	if err != nil || len(changes) != 1 {
		t.Fatalf("%v, %v", changes, err)
	}
	if err := changes[0].Apply(context.Background()); err == nil {
		t.Fatal("the tool is installed with no .NET")
	}
	m.Files[dotnetBin] = nil
	line := "env HOME=/root env DOTNET_CLI_TELEMETRY_OPTOUT=1 " + dotnetBin + " tool update --global " +
		"roslyn-language-server --version 5.12.0-1.26475.2 --allow-downgrade --source https://api.nuget.org/v3/index.json"
	m.Results[line] = effecttest.Result{}
	if err := changes[0].Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(m.Calls, "run "+line) {
		t.Errorf("calls %q", strings.Join(m.Calls, "\n"))
	}
}
