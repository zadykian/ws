package zed

import (
	"embed"
	"fmt"
	"strings"

	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/editor"
)

//go:embed global_settings.json.tmpl settings.json.tmpl
var templates embed.FS

// languages gives each neutral language of the file types Zed's name.
var languages = map[string]string{"dockerfile": "Dockerfile", "docker-compose": "Docker Compose"}

// width is the widest line that a list stays on.
const width = 100

// Settings renders the Mac's global_settings.json, under the header that ws mac apply adds.
func Settings() ([]byte, error) {
	settings, err := editor.LoadSettings()
	if err != nil {
		return nil, err
	}
	types, err := fileTypes()
	if err != nil {
		return nil, err
	}
	return host.Render(templates, "global_settings.json.tmpl", struct {
		Mode, Light, Dark, Font, FileTypes string
	}{
		Mode:      quote(settings.Theme.Mode),
		Light:     quote(settings.Theme.Light),
		Dark:      quote(settings.Theme.Dark),
		Font:      quote(settings.Font),
		FileTypes: types,
	})
}

// fileTypes writes the key file_types, as both of Zed's files hold it: a language's patterns on
// one line where it fits, and one to a line where it does not.
func fileTypes() (string, error) {
	types, err := editor.LoadFileTypes()
	if err != nil {
		return "", err
	}
	lines := make([]string, len(types))
	for i, t := range types {
		name, found := languages[t.Language]
		if !found {
			return "", fmt.Errorf("zed has no name for the language %s", t.Language)
		}
		line := "    " + quote(name) + ": " + list(t.Globs)
		if len(line) >= width {
			line = "    " + quote(name) + ": [\n      " +
				strings.Join(quoted(t.Globs), ",\n      ") + "\n    ]"
		}
		lines[i] = line
	}
	return "  \"file_types\": {\n" + strings.Join(lines, ",\n") + "\n  }", nil
}

func list(items []string) string { return "[" + strings.Join(quoted(items), ", ") + "]" }

func quoted(items []string) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = quote(item)
	}
	return out
}
