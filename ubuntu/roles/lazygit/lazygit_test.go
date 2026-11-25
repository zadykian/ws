package lazygit_test

import (
	"bytes"
	_ "embed"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// config and wrapper are the role's files, as the mise config reads them.
//
//go:embed files/config.yml
var config []byte

//go:embed files/lazygit
var wrapper []byte

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
	lines, err := os.ReadFile("mise.guest.toml")
	if err != nil || !strings.Contains(string(lines), `"apt:git-delta" = "latest"`) {
		t.Errorf("mise.guest.toml names no package git-delta: %v", err)
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
