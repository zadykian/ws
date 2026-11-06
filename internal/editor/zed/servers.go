package zed

import (
	"encoding/json"
	"fmt"

	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/editor"
)

// lspKeys gives the servers whose key in Zed's lsp differs from their name.
var lspKeys = map[string]string{"ansible-language-server": "ansible", "roslyn-language-server": "roslyn"}

type server struct{ Key, Path, Args, Init string }

// ServerSettings renders the settings.json of Zed's server on ws. paths gives each server of the
// catalogue its binary, as devtools installs it, dlv gives Delve's, and terminal the program that
// Zed's terminals run.
func ServerSettings(paths map[string]string, dlv, terminal string) ([]byte, error) {
	catalogue, err := editor.LoadServers()
	if err != nil {
		return nil, err
	}
	servers := make([]server, len(catalogue))
	for i, s := range catalogue {
		path, found := paths[s.Name]
		if !found {
			return nil, fmt.Errorf("%s has no path", s.Name)
		}
		key, renamed := lspKeys[s.Name]
		if !renamed {
			key = s.Name
		}
		servers[i] = server{Key: quote(key), Path: quote(path), Args: list(s.Args)}
		if s.Init != nil {
			init, err := json.Marshal(s.Init)
			if err != nil {
				return nil, err
			}
			servers[i].Init = string(init)
		}
	}
	types, err := fileTypes()
	if err != nil {
		return nil, err
	}
	return host.Render(templates, "settings.json.tmpl", struct {
		Servers                  []server
		FileTypes, Dlv, Terminal string
	}{servers, types, quote(dlv), quote(terminal)})
}
