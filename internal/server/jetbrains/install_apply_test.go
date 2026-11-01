package jetbrains

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestInstallLeavesABuildInPlace(t *testing.T) {
	m, _, c := fixture(1 << 30)
	m.Files[testDir+"/"+marker] = nil
	if err := c.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(m.Calls, "\n"), "download") {
		t.Error("downloaded a build that is in place")
	}
}

func TestInstallRemovesTheDownloadsWhereItFails(t *testing.T) {
	m, _, c := fixture(1 << 30)
	c.sum = strings.Repeat("0", 64)
	if err := c.Apply(context.Background()); err == nil {
		t.Fatal("a wrong sum is no error")
	}
	if !slices.Contains(m.Calls, "run rm -rf -- /h/.cache/ws-jetbrains/playbook") {
		t.Errorf("downloads kept: %v", m.Calls)
	}
}

func TestInstallDownloadsThenUnpacksUnderTheLock(t *testing.T) {
	m, s, c := fixture(1 << 30)
	m.Results["chown -R -h 1000:1000 -- "+testDir+".part"] = effecttest.Result{}
	var locked int
	s.Lock = func(_ string, fn func() error) error { locked++; return fn() }
	c.s = s
	if err := c.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if locked != 2 {
		t.Errorf("locked %d times, want 2: to clear, then to unpack", locked)
	}
	calls := strings.Join(m.Calls, "\n")
	for _, want := range []string{"download " + testLink, "run mv -T --update=none-fail", "run rm -rf -- /h/.cache/ws-jetbrains/playbook"} {
		if !strings.Contains(calls, want) {
			t.Errorf("no call %q in\n%s", want, calls)
		}
	}
	if m.Owners["/h/.cache/ws-jetbrains/playbook"] != [2]int{1000, 1000} {
		t.Error("the downloads' directory is not the user's")
	}
}
