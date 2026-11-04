package fresh

import (
	"fmt"

	"github.com/zadykian/ws/internal/editor"
)

// server is an entry of lsp. Fresh takes a list of them for a language, as dockerfile has two.
type server struct {
	Command   string         `json:"command"`
	Args      []string       `json:"args"`
	AutoStart bool           `json:"auto_start"`
	Init      map[string]any `json:"initialization_options,omitempty"`
}

// languages gives each language of the catalogue its name in Fresh. ansible has none: no pattern
// tells a playbook from YAML, and Fresh takes no first-line or path rule for it.
var languages = map[string]string{
	"go": "go", "csharp": "csharp", "sql": "sql", "html": "html", "dockerfile": "dockerfile",
	"docker-compose": "docker-compose", "ansible": "",
}

func serverEntries(catalogue []editor.Server, paths map[string]string) (map[string][]server, error) {
	found := map[string][]server{}
	for _, s := range catalogue {
		path, known := paths[s.Name]
		if !known {
			return nil, fmt.Errorf("%s has no path", s.Name)
		}
		for _, name := range s.Languages {
			fresh, known := languages[name]
			if !known {
				return nil, fmt.Errorf("%s: language %s has no name in Fresh", s.Name, name)
			}
			if fresh == "" {
				continue
			}
			found[fresh] = append(found[fresh], server{Command: path, Args: s.Args, AutoStart: true, Init: s.Init})
		}
	}
	return found, nil
}
