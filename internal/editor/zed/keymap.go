package zed

import (
	_ "embed" // sections.yaml
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/zadykian/ws/internal/editor"
)

//go:embed sections.yaml
var sectionsFile []byte

// layout is what Zed's file has that the keymap lacks: its comments, its contexts and its unbinds.
type layout struct {
	Header   string    `yaml:"header"`
	Sections []section `yaml:"sections"`
}

type section struct {
	Scope   editor.Scope `yaml:"scope"`
	Context string       `yaml:"context"`
	Comment string       `yaml:"comment"`
	Unbind  []unbind     `yaml:"unbind"`
}

// unbind takes a key from Zed's defaults, or from a wider scope: the action it had, in Zed's terms.
type unbind struct {
	Key  string `yaml:"key"`
	To   string `yaml:"to"`
	Args string `yaml:"args"`
}

// Keymap renders Zed's keymap.json.
func Keymap() ([]byte, error) {
	keymap, err := editor.LoadKeymap()
	if err != nil {
		return nil, err
	}
	targets, err := editor.LoadTargets(actionFiles, "actions/*.yaml")
	if err != nil {
		return nil, err
	}
	if err := keymap.Check(targets); err != nil {
		return nil, err
	}
	var l layout
	if err := yaml.Unmarshal(sectionsFile, &l); err != nil {
		return nil, err
	}
	return render(keymap, targets, l), nil
}

func render(keymap editor.Keymap, targets map[string]editor.Target, l layout) []byte {
	var b strings.Builder
	comment(&b, "", l.Header)
	b.WriteString("[\n")
	for i, s := range l.Sections {
		comment(&b, "  ", s.Comment)
		b.WriteString("  {\n")
		if s.Context != "" {
			fmt.Fprintf(&b, "    \"context\": %s,\n", quote(s.Context))
		}
		if len(s.Unbind) > 0 {
			lines := make([]string, len(s.Unbind))
			for j, u := range s.Unbind {
				lines[j] = quote(u.Key) + ": " + action(editor.Target{To: u.To, Args: u.Args}, u.Key)
			}
			object(&b, "unbind", lines, ",")
		}
		object(&b, "bindings", bindings(keymap[s.Scope], targets), "")
		b.WriteString("  }")
		if i < len(l.Sections)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("]\n")
	return []byte(b.String())
}

// bindings lists a scope's keys in order, leaving out each action that Zed has no target for.
func bindings(list []editor.Binding, targets map[string]editor.Target) []string {
	var lines []string
	for _, binding := range list {
		target := targets[binding.Action]
		if target.To == "" {
			continue
		}
		for _, chord := range binding.Keys {
			key := keystroke(chord)
			lines = append(lines, quote(key)+": "+action(target, key))
		}
	}
	return lines
}

func action(t editor.Target, key string) string {
	switch {
	case t.KeyArg:
		return "[" + quote(t.To) + ", " + quote(key) + "]"
	case t.Args != "":
		return "[" + quote(t.To) + ", " + t.Args + "]"
	default:
		return quote(t.To)
	}
}

func comment(b *strings.Builder, indent, text string) {
	for line := range strings.Lines(text) {
		b.WriteString(indent + "// " + line)
	}
}

func object(b *strings.Builder, name string, lines []string, after string) {
	fmt.Fprintf(b, "    %s: {\n", quote(name))
	for i, line := range lines {
		b.WriteString("      " + line)
		if i < len(lines)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("    }" + after + "\n")
}

// quote writes a JSON string. encoding/json would escape <, > and &, which a key holds.
func quote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
