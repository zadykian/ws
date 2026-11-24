package devtools

import "testing"

func TestServerPathsNameEachServer(t *testing.T) {
	paths := ServerPaths("/srv/x")
	if len(paths) != 7 {
		t.Fatalf("%d paths: %v", len(paths), paths)
	}
	for name, want := range map[string]string{
		"gopls":                      "/usr/local/bin/gopls",
		"postgres-language-server":   "/usr/local/bin/postgres-language-server",
		"dockerfile-language-server": "/srv/x/.local/lib/language-servers/node_modules/.bin/docker-langserver",
	} {
		if paths[name] != want {
			t.Errorf("%s: %q, want %q", name, paths[name], want)
		}
	}
}

func TestPathsFollowTheHome(t *testing.T) {
	if got, want := Roslyn("/srv/x"), "/srv/x/.dotnet/tools/roslyn-language-server"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
