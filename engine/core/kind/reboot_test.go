package kind

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// rebootUnit is a reboot state of the module m, whose version m-driver holds.
const rebootUnit = "reboot = { module = \"m\", package = \"m-driver\", hint = \"Until then, x waits.\" }\n"

func TestRebootSaysWhileTheLoadedModuleDiffers(t *testing.T) {
	for _, tc := range []struct {
		name, version, status, loaded, want string
	}{
		{"none loaded", "595.91.07-0ubuntu0.26.04.2", "ii", "",
			"warn: the kernel runs the module m at none, and m-driver holds 595.91.07. Reboot to load it. Until then, x waits."},
		{"an older one", "595.91.07-0ubuntu0.26.04.2", "ii", "595.58.03\n", "module m at 595.58.03, and m-driver holds 595.91.07"},
		{"the installed one", "595.91.07-0ubuntu1", "ii", "595.91.07\n", ""},
		{"not installed", "595.91.07-0ubuntu1", "rc", "", ""},
		{"an epoch and a revision", "1:2.0-1", "ii", "2.0\n", ""},
	} {
		e, m, _, out := testEnv(true)
		m.Packages = effecttest.Packages{"m-driver": {Version: tc.version, Status: tc.status}}
		if tc.loaded != "" {
			m.Files["/sys/module/m/version"] = []byte(tc.loaded)
		}
		got := headlines(t, loadState(t, e, rebootUnit)[0].Step)
		said := strings.Join(out.Lines, "\n")
		if got != nil || (tc.want == "") != (said == "") || len(out.Lines) > 1 || !strings.Contains(said, tc.want) {
			t.Errorf("%s: plan = %q, says %q; want none, and %q", tc.name, got, said, tc.want)
		}
	}
}

func TestRebootSkipsWithoutSystemd(t *testing.T) {
	e, m, _, out := testEnv(false)
	got := headlines(t, loadState(t, e, rebootUnit)[0].Step)
	if got != nil || !slices.Equal(out.Lines, []string{"say: reboot of module m: skipped, systemd is not running"}) {
		t.Errorf("plan = %q, says %q; want none, and the skip", got, out.Lines)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return c != "stat /run/systemd/system" }) {
		t.Errorf("calls = %q, want no read of dpkg or /sys", m.Calls)
	}
}

func TestRebootRefusesBadFields(t *testing.T) {
	e, _, _, _ := testEnv(false)
	stateRefused(t, e, "reboot = { module = \"../x\", package = \"p\" }\n", `reboot: module "../x" is no kernel module's name`)
	stateRefused(t, e, "reboot = { module = \"m\", package = \"-o\" }\n", `reboot: package "-o" is no package's name`)
	stateRefused(t, e, "reboot = { module = \"m\", package = \"p=1\" }\n", `reboot: package "p=1" is no package's name`)
}
