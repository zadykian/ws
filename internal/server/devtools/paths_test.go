package devtools

import "testing"

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
