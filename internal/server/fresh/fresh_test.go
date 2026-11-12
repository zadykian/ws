package fresh

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
)

// stepNamed returns the role's step of that name.
func stepNamed(t *testing.T, m *effecttest.Machine, v values.Values, name string) step.Step {
	for _, s := range Steps(v, secrets.Secrets{}, m, &effecttest.Units{}, &effecttest.Out{}) {
		if s.Name() == name {
			return s
		}
	}
	t.Fatalf("no step %q", name)
	return nil
}

func userMachine() (*effecttest.Machine, values.Values) {
	m := effecttest.NewMachine()
	m.Dirs["/home/ws"] = 0o755
	m.Results["id -u ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g ws"] = effecttest.Result{Out: []byte("1000\n")}
	v := values.Default()
	v.User, v.Home = "ws", "/home/ws"
	return m, v
}

func TestConfigIsWrittenOnceForTheUser(t *testing.T) {
	m, v := userMachine()
	s := stepNamed(t, m, v, configStep)
	if _, err := step.Run(context.Background(), []step.Step{s}, &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	changed, err := step.Run(context.Background(), []step.Step{s}, &effecttest.Out{}, false)
	if err != nil || changed {
		t.Errorf("the second run: changed %v, %v", changed, err)
	}
	for _, path := range []string{
		"/home/ws/.config/fresh/config.json", "/home/ws/.config/fresh/themes/jetbrains-dark.json",
	} {
		if o := m.Owners[path]; o != [2]int{1000, 1000} {
			t.Errorf("%s owner %v", path, o)
		}
	}
}

func TestConfigRefusesALink(t *testing.T) {
	m, v := userMachine()
	m.Links["/home/ws/.config/fresh/themes"] = "/srv/target"
	s := stepNamed(t, m, v, configStep)
	if _, err := step.Run(context.Background(), []step.Step{s}, &effecttest.Out{}, false); err == nil {
		t.Error("no error")
	}
	if len(m.Owners) != 0 || len(m.Files) != 0 {
		t.Errorf("changed %v %v", m.Owners, m.Files)
	}
}

func TestConfigNamesTheHomesServers(t *testing.T) {
	m, v := userMachine()
	s := stepNamed(t, m, v, configStep)
	if _, err := step.Run(context.Background(), []step.Step{s}, &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/home/ws/.config/fresh/config.json"]); !strings.Contains(got, `"/home/ws/go/bin/gopls"`) {
		t.Error("config.json lacks the user's gopls")
	}
}

func TestTheActionListNamesThePinnedVersion(t *testing.T) {
	data, err := os.ReadFile("files/actions.txt")
	if err != nil {
		t.Fatal(err)
	}
	line, _, _ := strings.Cut(string(data), "\n")
	if want := "Fresh " + strings.TrimPrefix(tag, "v") + ":"; !strings.Contains(line, want) {
		t.Errorf("the list's first line %q lacks %q: list the actions of the new version", line, want)
	}
}

func TestTheThemeIsStrictJSONUnderItsName(t *testing.T) {
	var theme map[string]any
	if err := json.Unmarshal(themeFile, &theme); err != nil {
		t.Fatal(err)
	}
	if theme["name"] != "JetBrains Dark" {
		t.Fatalf("name %v, want JetBrains Dark", theme["name"])
	}
	for section, value := range theme {
		colours, isMap := value.(map[string]any)
		if !isMap {
			continue
		}
		for name, colour := range colours {
			triple, isList := colour.([]any)
			if !isList || len(triple) != 3 {
				t.Errorf("%s.%s: %v isn't [r, g, b]", section, name, colour)
			}
		}
	}
}
