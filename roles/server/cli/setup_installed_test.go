package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// installedWs is the ws that the units and the frame's checks run.
const installedWs = "/usr/local/bin/ws"

// TestSetupRefusesWhereTheInstalledWsDiffers checks that setup refuses before any write where the
// ws the units run is missing or another version. The tree would be this ws's, and that ws would
// read it. The refusal names the fix.
func TestSetupRefusesWhereTheInstalledWsDiffers(t *testing.T) {
	for name, tc := range map[string]struct {
		version string
		run     effecttest.Result
		args    []string
		fails   []string
	}{
		"another release": {"v1.2.3", effecttest.Result{Out: []byte("v1.2.2\n")}, nil,
			[]string{installedWs + ", which the units run, is v1.2.2", "ws update --version 1.2.3"}},
		"missing": {"v1.2.3", effecttest.Result{Err: errors.New("no such file or directory")}, nil,
			[]string{installedWs + ", which the units run, does not run", "ws update --version 1.2.3"}},
		"a plan": {"v1.2.3", effecttest.Result{Out: []byte("v1.2.2\n")}, []string{"--plan"},
			[]string{"is v1.2.2", "ws update --version 1.2.3"}},
		"a trial's build": {"devel", effecttest.Result{Out: []byte("v1.2.3\n")}, []string{"--in-place"},
			[]string{"this ws is devel", "is v1.2.3", "install this build there"}},
	} {
		t.Run(name, func(t *testing.T) {
			rel := setupRelease(miseBody)
			p, m := setupProcess(rel)
			p.Version = tc.version
			m.Results[installedWs+" version"] = tc.run
			status, stderr := runSetup(p, rel, tc.args...)
			for _, fails := range tc.fails {
				if status != 1 || !strings.Contains(stderr, fails) {
					t.Errorf("status = %d, stderr = %q; want 1 and %q", status, stderr, fails)
				}
			}
			leftNothing(t, m, false)
			if indexOf(m, "run git") >= 0 {
				t.Errorf("setup read the tree first: %q", m.Calls)
			}
		})
	}
}
