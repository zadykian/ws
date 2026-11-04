package devtools

import "testing"

func TestServerPathsNameEachServerOfTheCatalogue(t *testing.T) {
	paths := ServerPaths("/srv/x")
	if len(paths) != 7 {
		t.Fatalf("%d paths: %v", len(paths), paths)
	}
	for name, want := range map[string]string{
		"gopls":                      "/srv/x/go/bin/gopls",
		"postgres-language-server":   "/usr/local/bin/postgres-language-server",
		"dockerfile-language-server": "/srv/x/.local/lib/zed-servers/node_modules/.bin/docker-langserver",
	} {
		if paths[name] != want {
			t.Errorf("%s: %q, want %q", name, paths[name], want)
		}
	}
}

func TestPathsFollowTheHome(t *testing.T) {
	tests := map[string]string{
		Gopls("/srv/x"):  "/srv/x/go/bin/gopls",
		Roslyn("/srv/x"): "/srv/x/.dotnet/tools/roslyn-language-server",
		Dlv("/srv/x"):    "/srv/x/go/bin/dlv",
	}
	for got, want := range tests {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}
