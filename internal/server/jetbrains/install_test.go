package jetbrains

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/effect/effecttest"
)

const (
	testDist = "/h/.cache/JetBrains/RemoteDev/dist"
	testDir  = testDist + "/0123456789abc_goland-2026.2.3"
	testLink = "https://dl/goland-2026.2.3.tar.gz"
)

var testBody = []byte("archive")

// fixture is a machine where Gateway's directory and the home are made, and the commands an
// install runs have results.
func fixture(avail uint64) (*effecttest.Machine, Site, install) {
	m := effecttest.NewMachine()
	m.Dirs["/h"] = 0o755
	m.Dirs[testDist] = 0o755
	for _, line := range []string{
		"rm -rf -- " + testDir + " " + testDir + ".part", "rm -rf -- " + testDir + ".part",
		"rm -rf -- /h/.cache/ws-jetbrains/playbook",
		"mv -T --update=none-fail -- " + testDir + ".part " + testDir,
	} {
		m.Results[line] = effecttest.Result{}
	}
	m.Pages[testLink] = testBody
	s := Site{
		M: m, Out: &effecttest.Out{}, Dist: testDist, Downloads: "/h/.cache/ws-jetbrains/playbook", Home: "/h",
		UID: 1000, GID: 1000, Wait: time.Nanosecond,
		Avail:  func(string) (uint64, error) { return avail, nil },
		Unpack: func(string, string, int) error { return nil },
		Lock:   func(_ string, fn func() error) error { return fn() },
	}
	b := Build{Code: "GO", Link: testLink, Size: 100, Name: "goland-2026.2.3", Dir: testDir}
	return m, s, install{s: s, b: b, sum: fmt.Sprintf("%x", sha256.Sum256(testBody))}
}

func TestInstallUnpacksThenMovesIntoPlace(t *testing.T) {
	m, s, c := fixture(1 << 30)
	m.Results["chown -R -h 1000:1000 -- "+testDir+".part"] = effecttest.Result{}
	var unpacked []string
	s.Unpack = func(archive, dir string, strip int) error {
		unpacked = append(unpacked, fmt.Sprint(archive, " ", dir, " ", strip))
		return nil
	}
	c.s = s
	if err := c.unpack(context.Background()); err != nil {
		t.Fatal(err)
	}
	if want := "/h/.cache/ws-jetbrains/playbook/goland-2026.2.3.tar.gz " + testDir + ".part 1"; !slices.Equal(unpacked, []string{want}) {
		t.Errorf("unpacked %v", unpacked)
	}
	if _, ok := m.Files[testDir+".part/"+marker]; !ok {
		t.Error("no marker in the unpacked tree")
	}
	if !slices.Contains(m.Calls, "run mv -T --update=none-fail -- "+testDir+".part "+testDir) {
		t.Errorf("not moved into place: %v", m.Calls)
	}
}

func TestInstallFailsWhereTheUnpackFails(t *testing.T) {
	m, s, c := fixture(1 << 30)
	s.Unpack = func(string, string, int) error { return errors.New("cut") }
	c.s = s
	if err := c.unpack(context.Background()); err == nil {
		t.Fatal("no error")
	}
	if slices.ContainsFunc(m.Calls, func(l string) bool { return strings.HasPrefix(l, "run mv") }) {
		t.Error("moved a tree that did not unpack")
	}
}
