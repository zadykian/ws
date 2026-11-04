package zed

import (
	"strings"

	"github.com/zadykian/ws/internal/editor"
)

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

// quote writes a JSON string. encoding/json would escape <, > and &, which a key holds.
func quote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
