package base

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/step"
)

const supported = "# comment\nC.UTF-8 UTF-8\nde_DE@euro ISO-8859-15\nen_US.UTF-8 UTF-8\n"

func TestLocaleRefusals(t *testing.T) {
	for _, name := range []string{"", "en_US.UTF-8 UTF-8", "english", "en_us.utf8", "en_US.UTF-8\n#", "xx_YY.UTF-8", "en_US"} {
		m := effecttest.NewMachine()
		m.Files[supportedFile] = []byte(supported)
		g := host.Group{Title: "Locale", Parts: localeParts(m, name)}
		if _, err := g.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "base_locale") {
			t.Errorf("locale %q: error = %v, want a refusal naming base_locale", name, err)
		}
	}
}

func TestLocale(t *testing.T) {
	for _, tt := range []struct {
		name, gen, avail string
		generates        bool
		wantGen          string
	}{
		{"commented", "# en_US.UTF-8 UTF-8\n# fr_FR.UTF-8 UTF-8\n", "C\n", true, "en_US.UTF-8 UTF-8\n# fr_FR.UTF-8 UTF-8\n"},
		{"absent file", "", "C\n", true, "en_US.UTF-8 UTF-8\n"},
		{"enabled, not made", "en_US.UTF-8 UTF-8\n", "C\n", true, "en_US.UTF-8 UTF-8\n"},
		{"made", "en_US.UTF-8 UTF-8\n", "C\nen_US.utf8\n", false, "en_US.UTF-8 UTF-8\n"},
		{"made, not enabled", "# en_US.UTF-8 UTF-8\n", "C\nen_US.utf8\n", false, "# en_US.UTF-8 UTF-8\n"},
	} {
		m := effecttest.NewMachine()
		m.Files[supportedFile] = []byte(supported)
		if tt.gen != "" {
			m.Files["/etc/locale.gen"] = []byte(tt.gen)
		}
		m.Results["locale -a"] = effecttest.Result{Out: []byte(tt.avail)}
		m.Results["locale-gen"] = effecttest.Result{}
		g := host.Group{Title: "Locale", Parts: localeParts(m, "en_US.UTF-8")}
		// The fake's `locale -a` stays unchanged after locale-gen, so a run that generates fails its
		// check that the second plan is empty.
		_, err := step.Run(t.Context(), []step.Step{g}, &effecttest.Out{}, false)
		if (err != nil) != tt.generates {
			t.Errorf("%s: err = %v, want an error: %v", tt.name, err, tt.generates)
		}
		if got := string(m.Files["/etc/locale.gen"]); got != tt.wantGen {
			t.Errorf("%s: locale.gen = %q, want %q", tt.name, got, tt.wantGen)
		}
		if got := string(m.Files["/etc/locale.conf"]); got != "LANG=en_US.UTF-8\n" {
			t.Errorf("%s: locale.conf = %q", tt.name, got)
		}
		ran := false
		for _, c := range m.Calls {
			ran = ran || c == "root locale-gen"
		}
		if ran != tt.generates {
			t.Errorf("%s: locale-gen ran = %v, want %v", tt.name, ran, tt.generates)
		}
	}
}

func TestLocaleOfOtherForms(t *testing.T) {
	for name, line := range map[string]string{"C.UTF-8": "C.UTF-8 UTF-8", "de_DE@euro": "de_DE@euro ISO-8859-15"} {
		m := effecttest.NewMachine()
		m.Files[supportedFile] = []byte(supported)
		m.Results["locale -a"] = effecttest.Result{Out: []byte("C\n")}
		g := host.Group{Title: "Locale", Parts: localeParts(m, name)}
		if _, err := g.Plan(t.Context()); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got, ok := entry(supported, name); !ok || got != line {
			t.Errorf("entry(%q) = %q, %v, want %q", name, got, ok, line)
		}
	}
}
