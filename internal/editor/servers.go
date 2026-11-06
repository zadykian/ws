package editor

import "fmt"

// Server is a language server of the catalogue. Init is its initialization options.
type Server struct {
	Name      string         `yaml:"server"`
	Languages []string       `yaml:"languages"`
	Args      []string       `yaml:"args"`
	Init      map[string]any `yaml:"init"`
}

// LoadServers reads the embedded catalogue, in its order.
func LoadServers() ([]Server, error) {
	var servers []Server
	if err := load("servers.yaml", &servers); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, s := range servers {
		if s.Name == "" || seen[s.Name] || len(s.Languages) == 0 {
			return nil, fmt.Errorf("servers.yaml: bad entry %q", s.Name)
		}
		seen[s.Name] = true
		for _, language := range s.Languages {
			if !languageName.MatchString(language) {
				return nil, fmt.Errorf("servers.yaml: %s: bad language %q", s.Name, language)
			}
		}
	}
	return servers, nil
}
