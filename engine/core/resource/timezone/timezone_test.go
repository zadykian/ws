package timezone

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

func TestTimezone(t *testing.T) {
	for _, tt := range []struct {
		name       string
		legacy     bool
		wantLegacy string
	}{{"no /etc/timezone", false, ""}, {"with /etc/timezone", true, "Europe/Paris\n"}} {
		m := effecttest.NewMachine()
		m.Links["/etc/localtime"] = "/usr/share/zoneinfo/Etc/UTC"
		m.Files["/usr/share/zoneinfo/Europe/Paris"] = []byte("tz")
		if tt.legacy {
			m.Files["/etc/timezone"] = []byte("Etc/UTC\n")
		}
		g := resource.Group{Title: "Timezone", Parts: []resource.Resource{Zone{M: m, Name: "Europe/Paris"}}}
		changed, err := step.Run(t.Context(), []step.Step{g}, &effecttest.Out{}, false)
		if err != nil || !changed {
			t.Fatalf("%s: changed = %v, err = %v", tt.name, changed, err)
		}
		if got := m.Links["/etc/localtime"]; got != "/usr/share/zoneinfo/Europe/Paris" {
			t.Errorf("%s: link = %q", tt.name, got)
		}
		if got := string(m.Files["/etc/timezone"]); got != tt.wantLegacy {
			t.Errorf("%s: /etc/timezone = %q, want %q", tt.name, got, tt.wantLegacy)
		}
	}
}

func TestTimezoneRefusals(t *testing.T) {
	for _, zone := range []string{"", "/Etc/UTC", "../x", "Etc/../../x", "Etc//UTC", "..", "Europe/Pariss", "Europe"} {
		m := effecttest.NewMachine()
		m.Files["/usr/share/zoneinfo/Europe/Paris"] = []byte("tz")
		g := resource.Group{Title: "Timezone", Parts: []resource.Resource{Zone{M: m, Name: zone, Hint: "Name another."}}}
		if _, err := g.Plan(t.Context()); err == nil || !strings.HasSuffix(err.Error(), ". Name another.") {
			t.Errorf("zone %q: error = %v, want a refusal that ends with the hint", zone, err)
		}
	}
}
