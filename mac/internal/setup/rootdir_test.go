package setup

import (
	"fmt"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

func TestRootDirPlan(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/etc/ok"], m.Files["/etc/bad"] = nil, nil
	m.Root["/etc/ok"] = effect.Attrs{Mode: 0o644}
	m.Root["/etc/bad"] = effect.Attrs{UID: 501, GID: 20, Mode: 0o700}
	for path, want := range map[string]string{
		"/etc/new": "would make /etc/new, root's, with mode 644",
		"/etc/bad": "would make /etc/bad root's, with mode 644, rather than 501:20 700",
		"/etc/ok":  "",
	} {
		c, err := rootDir(m, path, 0o644)
		if err != nil || (want == "") != (c == nil) || (c != nil && fmt.Sprint(c) != want) {
			t.Errorf("%s: %v, %v; want %q", path, c, err, want)
		}
	}
}
