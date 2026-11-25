package networks

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/resource/file"
)

func TestRoutesAndResolverTexts(t *testing.T) {
	if got, want := routesText("ws", "values", []string{"10.1.0.0/24", "10.2.0.0/24"}),
		"# "+file.Mark+": ws macos tunnel routes --server ws values.\n10.1.0.0/24\n10.2.0.0/24\n"; got != want {
		t.Errorf("routesText = %q; want %q", got, want)
	}
	if got, want := resolverText([]string{"10.0.0.1", "10.0.0.2"}),
		resolverHeader+"\nnameserver 10.0.0.1\nnameserver 10.0.0.2\n"; got != want {
		t.Errorf("resolverText = %q; want %q", got, want)
	}
}

func TestResolverHeaderNamesTheKey(t *testing.T) {
	if want := "from WS_RESOLVERS."; !strings.HasSuffix(resolverHeader, want) {
		t.Errorf("resolverHeader = %q; want it to end with %q", resolverHeader, want)
	}
}

func TestResolvers(t *testing.T) {
	got := resolvers("a.test=10.0.0.1,10.0.0.2 b.test=10.0.0.3")
	want := []resolver{{"a.test", []string{"10.0.0.1", "10.0.0.2"}}, {"b.test", []string{"10.0.0.3"}}}
	if !slices.EqualFunc(got, want, func(a, b resolver) bool {
		return a.Domain == b.Domain && slices.Equal(a.Addresses, b.Addresses)
	}) {
		t.Errorf("resolvers = %v; want %v", got, want)
	}
}
