package nvidia

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

const versionQuery = "dpkg-query -W -f ${Version} ${db:Status-Abbrev}\n " + smiPackage

func TestRebootSaysWhileTheLoadedDriverDiffers(t *testing.T) {
	for _, tc := range []struct {
		name, dpkg, loaded, want string
	}{
		{"none loaded", "595.91.07-0ubuntu0.26.04.2 ii \n", "", "driver none, and 595.91.07 is installed"},
		{"an older one", "595.91.07-0ubuntu0.26.04.2 ii \n", "595.58.03\n", "driver 595.58.03, and 595.91.07"},
		{"the installed one", "595.91.07-0ubuntu1 ii \n", "595.91.07\n", ""},
		{"not installed", "595.91.07-0ubuntu1 rc \n", "", ""},
	} {
		m, out := effecttest.NewMachine(), &effecttest.Out{}
		m.Dirs["/run/systemd/system"] = 0o755
		m.Results[versionQuery] = effecttest.Result{Out: []byte(tc.dpkg)}
		if tc.loaded != "" {
			m.Files[loadedVersion] = []byte(tc.loaded)
		}
		changes, err := reboot{m, out}.Plan(t.Context())
		said := strings.Join(out.Lines, "\n")
		if err != nil || len(changes) != 0 || (tc.want == "") != (said == "") || !strings.Contains(said, tc.want) {
			t.Errorf("%s: Plan = %v, %v, says %q; want %q", tc.name, changes, err, said, tc.want)
		}
	}
}

func TestRebootSkipsWithoutSystemd(t *testing.T) {
	out := &effecttest.Out{}
	changes, err := reboot{effecttest.NewMachine(), out}.Plan(t.Context())
	if err != nil || len(changes) != 0 || len(out.Lines) != 1 || !strings.Contains(out.Lines[0], "skipped") {
		t.Errorf("Plan = %v, %v, says %q; want a skip", changes, err, out.Lines)
	}
}
