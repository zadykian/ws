package lazygit

import (
	"bytes"
	"context"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

func steps() []step.Step {
	return Steps(values.Default(), secrets.Secrets{}, effecttest.NewMachine(), &effecttest.Units{}, &effecttest.Out{})
}

func TestStepsAreTheRole(t *testing.T) {
	var names []string
	for _, s := range steps() {
		names = append(names, s.Name())
	}
	if want := []string{packageStep, wrapperStep, configStep}; !slices.Equal(names, want) {
		t.Errorf("steps %v, want %v", names, want)
	}
}

func TestTheWrapperWaitsOnThePackage(t *testing.T) {
	if got := steps()[1].WaitsOn(); got != packageStep {
		t.Errorf("the wrapper waits on %q", got)
	}
}

const configPath = "/home/ws/.config/lazygit/config.yml"

// configOn returns the config's step for the user ws on a machine that has the user's home.
func configOn() (*effecttest.Machine, step.Step) {
	m := effecttest.NewMachine()
	m.Dirs["/home/ws"] = 0o755
	m.Results["id -u ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g ws"] = effecttest.Result{Out: []byte("1000\n")}
	v := values.Default()
	v.User, v.Home = "ws", "/home/ws"
	return m, Steps(v, secrets.Secrets{}, m, &effecttest.Units{}, &effecttest.Out{})[2]
}

func TestConfigIsWrittenOnceForTheUser(t *testing.T) {
	m, s := configOn()
	if _, err := step.Run(context.Background(), []step.Step{s}, &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	changed, err := step.Run(context.Background(), []step.Step{s}, &effecttest.Out{}, false)
	if err != nil || changed {
		t.Errorf("the second run: changed %v, %v", changed, err)
	}
	if o := m.Owners[configPath]; o != [2]int{1000, 1000} {
		t.Errorf("owner %v", o)
	}
	if !bytes.Equal(m.Files[configPath], config) {
		t.Error("config.yml differs from the embedded file")
	}
}

func TestConfigRefusesALink(t *testing.T) {
	m, s := configOn()
	m.Links["/home/ws/.config"] = "/srv/target"
	if _, err := step.Run(context.Background(), []step.Step{s}, &effecttest.Out{}, false); err == nil {
		t.Error("no error")
	}
	if len(m.Owners) != 0 || len(m.Files) != 0 {
		t.Errorf("changed %v %v", m.Owners, m.Files)
	}
}

func TestTheConfigStartsWithItsHeader(t *testing.T) {
	if !bytes.HasPrefix(config, []byte("# Written by github.com/zadykian/ws,")) {
		t.Error("config.yml lacks the header")
	}
}

// leaves returns the dotted path of each value in the config that is not a map, such as
// git.skipHookPrefix.
func leaves(t *testing.T) map[string]any {
	var doc map[string]any
	if err := yaml.Unmarshal(config, &doc); err != nil {
		t.Fatal(err)
	}
	found := map[string]any{}
	var walk func(prefix string, node map[string]any)
	walk = func(prefix string, node map[string]any) {
		for k, v := range node {
			if inner, ok := v.(map[string]any); ok {
				walk(prefix+k+".", inner)
				continue
			}
			found[prefix+k] = v
		}
	}
	walk("", doc)
	return found
}

// A commit in lazygit runs the hooks: an empty prefix skips them on no summary, and w is off.
func TestTheConfigKeepsTheHooks(t *testing.T) {
	keys := leaves(t)
	if v, ok := keys["git.skipHookPrefix"]; !ok || v != "" {
		t.Errorf("git.skipHookPrefix is %q, want it empty", v)
	}
	if v := keys["keybinding.files.commitChangesWithoutHook"]; v != "<disabled>" {
		t.Errorf("keybinding.files.commitChangesWithoutHook is %q", v)
	}
}

// A colour attribute holds one colour at most, as #rrggbb, beside lazygit's modifiers.
func TestTheThemeHoldsOneColourAKey(t *testing.T) {
	hex := regexp.MustCompile(`^#[0-9a-f]{6}$`)
	modifiers := []string{"bold", "default", "reverse", "underline", "strikethrough"}
	theme := 0
	for path, v := range leaves(t) {
		if !strings.HasPrefix(path, "gui.theme.") {
			continue
		}
		theme++
		attrs, ok := v.([]any)
		if !ok {
			t.Errorf("%s is %T, not a list", path, v)
			continue
		}
		colours := 0
		for _, a := range attrs {
			s, _ := a.(string)
			switch {
			case hex.MatchString(s):
				colours++
			case !slices.Contains(modifiers, s):
				t.Errorf("%s holds %v", path, a)
			}
		}
		if colours != 1 {
			t.Errorf("%s holds %d colours", path, colours)
		}
	}
	if theme != 12 {
		t.Errorf("the theme sets %d keys, want lazygit's 12", theme)
	}
}

// The pager is delta, which the package git-delta installs beside lazygit.
func TestThePagerIsDeltaFromItsPackage(t *testing.T) {
	pagers, _ := leaves(t)["git.pagers"].([]any)
	if len(pagers) != 1 {
		t.Fatalf("pagers %v", pagers)
	}
	pager, _ := pagers[0].(map[string]any)
	if cmd, _ := pager["pager"].(string); !strings.HasPrefix(cmd, "delta ") {
		t.Errorf("pager %q", cmd)
	}
	group, _ := steps()[0].(host.Group)
	if packages, _ := group.Parts[0].(host.Packages); !slices.Contains(packages.Names, "git-delta") {
		t.Errorf("packages %v", packages.Names)
	}
}

// Each edit opens Fresh, which needs lazygit's terminal.
func TestTheEditorIsFresh(t *testing.T) {
	keys := leaves(t)
	for _, key := range []string{"os.edit", "os.editAtLine", "os.editAtLineAndWait", "os.openDirInEditor"} {
		if v, _ := keys[key].(string); !strings.HasPrefix(v, "fresh ") {
			t.Errorf("%s is %q", key, v)
		}
	}
	if keys["os.editInTerminal"] != true {
		t.Error("os.editInTerminal is not true")
	}
}
