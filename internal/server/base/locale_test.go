package base

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func TestLocaleRefusals(t *testing.T) {
	for _, name := range []string{"", "en_US.UTF-8 UTF-8", "english", "en_us.utf8", "en_US.UTF-8\n#", "de_DE@euro"} {
		m := effecttest.NewMachine()
		g := host.Group{Title: "Locale", Parts: localeParts(m, name)}
		if _, err := g.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "base_locale") {
			t.Errorf("locale %q: error = %v, want a refusal naming base_locale", name, err)
		}
	}
}
