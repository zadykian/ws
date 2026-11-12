package base

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/host"
)

// sysctlMachine is a machine with a swappiness in force, and systemd booted or not.
func sysctlMachine(now string, systemd bool) *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Files["/proc/sys/vm/swappiness"] = []byte(now + "\n")
	m.Results["/usr/lib/systemd/systemd-sysctl --strict --prefix=vm.swappiness"] = effecttest.Result{}
	if systemd {
		m.Dirs["/run/systemd/system"] = 0o755
	}
	return m
}

func TestSwappiness(t *testing.T) {
	for _, tt := range []struct {
		name, now   string
		systemd     bool
		wantSysctl  bool
		wantContent string
	}{
		{"another value in force", "60", true, true, "vm.swappiness = 10\n"},
		{"the value in force", "10", true, false, "vm.swappiness = 10\n"},
		{"no systemd", "60", false, false, "vm.swappiness = 10\n"},
	} {
		m := sysctlMachine(tt.now, tt.systemd)
		g := host.Group{Title: "Swappiness", Parts: swappinessParts(m, &effecttest.Out{}, 10)}
		p, err := g.Plan(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if err := g.Apply(t.Context(), p); err != nil {
			t.Fatal(err)
		}
		got := string(m.Files["/etc/sysctl.d/99-swappiness.conf"])
		if !strings.HasSuffix(got, tt.wantContent) || !strings.Contains(got, "Written by github.com/zadykian/ws") {
			t.Errorf("%s: file = %q", tt.name, got)
		}
		ran := false
		for _, c := range m.Calls {
			ran = ran || strings.HasPrefix(c, "root /usr/lib/systemd/systemd-sysctl")
		}
		if ran != tt.wantSysctl {
			t.Errorf("%s: systemd-sysctl ran = %v, want %v", tt.name, ran, tt.wantSysctl)
		}
	}
}
