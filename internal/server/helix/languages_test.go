package helix

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/server/values"
)

func TestLanguages(t *testing.T) {
	v := values.Default()
	v.Home = "/home/ws"
	got, err := languages(v)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
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
