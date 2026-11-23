package setup_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/internal/setup"
)

func TestCheckInstalled(t *testing.T) {
	for name, tc := range map[string]struct {
		version string
		run     effecttest.Result
		fails   string
	}{
		"the same release":   {"v1.2.3", effecttest.Result{Out: []byte("v1.2.3\n")}, ""},
		"the same build":     {"devel", effecttest.Result{Out: []byte("devel\n")}, ""},
		"another release":    {"v1.2.3", effecttest.Result{Out: []byte("v1.2.2\n")}, "is v1.2.2: run ws update --version 1.2.3,"},
		"an rc":              {"v1.2.3-rc.1", effecttest.Result{Out: []byte("v1.2.2\n")}, "ws update --version 1.2.3-rc.1,"},
		"a file that fails":  {"v1.2.3", effecttest.Result{Err: errors.New("exec format error")}, "does not run: exec format error"},
		"a trial's build":    {"devel", effecttest.Result{Out: []byte("v1.2.3\n")}, "install this build there"},
		"a pseudo-version's": {"v1.2.4-0.20261101000000-abcdef123456", effecttest.Result{Out: []byte("devel\n")}, "is devel: install"},
	} {
		t.Run(name, func(t *testing.T) {
			m := effecttest.NewMachine()
			m.Results["/usr/local/bin/ws version"] = tc.run
			err := setup.CheckInstalled(t.Context(), m, tc.version)
			if tc.fails == "" && err != nil || tc.fails != "" && (err == nil || !strings.Contains(err.Error(), tc.fails)) {
				t.Errorf("CheckInstalled = %v, want %q", err, tc.fails)
			}
			if got := strings.Join(m.Calls, "\n"); got != "run /usr/local/bin/ws version" {
				t.Errorf("calls = %q, want the run of its version alone", got)
			}
		})
	}
}
