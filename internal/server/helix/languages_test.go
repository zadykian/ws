package helix

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/server/values"
)

func TestLanguages(t *testing.T) {
	v := values.Default()
	v.Helix.Gopls, v.Helix.Roslyn = "/home/ws/go/bin/gopls", "/home/ws/.dotnet/tools/roslyn-language-server"
	v.Helix.Grammars = []string{"go", "c-sharp"}
	got, err := languages(v)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`use-grammars = { only = ["go", "c-sharp"] }`,
		`command = "/home/ws/go/bin/gopls"`,
		`command = "/home/ws/.dotnet/tools/roslyn-language-server"`,
		`args = ["--stdio", "--autoLoadProjects"]`,
	} {
		if !strings.Contains(string(got), want) {
			t.Errorf("languages.toml lacks %q", want)
		}
	}
}

func TestQuoteEscapes(t *testing.T) {
	if got := quote(`a"b\c`); got != `"a\"b\\c"` {
		t.Errorf("got %s", got)
	}
}
