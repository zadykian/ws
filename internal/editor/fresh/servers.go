package fresh

import (
	"fmt"
	"slices"

	"github.com/zadykian/ws/internal/editor"
)

// language is an entry of languages. Fresh's own entry for the language merges under it.
type language struct {
	Extensions []string `json:"extensions"`
	Filenames  []string `json:"filenames,omitempty"`
	Grammar    string   `json:"grammar"`
}

// server is an entry of lsp. Fresh takes a list of them for a language, as dockerfile has two.
type server struct {
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	AutoStart   bool              `json:"auto_start"`
	Init        map[string]any    `json:"initialization_options,omitempty"`
	LanguageIDs map[string]string `json:"language_id_overrides,omitempty"`
}

// languages gives each language of the catalogue its name in Fresh. ansible has none: no pattern
// tells a playbook from YAML, and Fresh takes no first-line or path rule for it.
var languages = map[string]string{
	"go": "go", "csharp": "csharp", "sql": "sql", "html": "html", "dockerfile": "dockerfile",
	"docker-compose": "docker-compose", "ansible": "",
}

// composeFiles are the names that Zed's own language knows and Fresh, with no such language, lacks.
var composeFiles = []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"}

// composeIDs tells docker-language-server a compose file, by Fresh's key: the file's extension.
var composeIDs = map[string]string{"yaml": "dockercompose", "yml": "dockercompose"}

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
			entry := server{Command: path, Args: s.Args, AutoStart: true, Init: s.Init}
			if fresh == "docker-compose" {
				entry.LanguageIDs = composeIDs
			}
			found[fresh] = append(found[fresh], entry)
		}
	}
	return found, nil
}

func languageEntries(types []editor.FileType) (map[string]language, error) {
	found := map[string]language{}
	for _, t := range types {
		fresh, known := languages[t.Language]
		if !known || fresh == "" {
			return nil, fmt.Errorf("file type %s has no name in Fresh", t.Language)
		}
		switch fresh {
		case "dockerfile":
			found[fresh] = language{
				Extensions: []string{},
				Filenames:  slices.Concat([]string{"Dockerfile", "Containerfile"}, t.Globs),
				Grammar:    "dockerfile",
			}
		case "docker-compose":
			found[fresh] = language{
				Extensions: []string{},
				Filenames:  slices.Concat(composeFiles, t.Globs),
				Grammar:    "yaml",
			}
		default:
			return nil, fmt.Errorf("file type %s: no rule", t.Language)
		}
	}
	return found, nil
}
