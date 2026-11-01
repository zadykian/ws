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
		"mkdir -m 0755 -- " + testDir + ".part",
		"tar -xzf /h/.cache/ws-jetbrains/playbook/goland-2026.2.3.tar.gz -C " + testDir + ".part --strip-components=1",
		"touch -- " + testDir + ".part/" + marker,
	} {
		m.Results[line] = effecttest.Result{}
	}
	m.Pages[testLink] = testBody
	s := Site{
		M: m, Out: &effecttest.Out{}, Dist: testDist, Downloads: "/h/.cache/ws-jetbrains/playbook", Home: "/h",
		UID: 1000, GID: 1000, Wait: time.Nanosecond,
		Avail: func(string) (uint64, error) { return avail, nil },
		AsUser: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return m.Run(ctx, name, args...)
		},
		Lock: func(_ string, fn func() error) error { return fn() },
	}
	b := Build{Code: "GO", Link: testLink, Size: 100, Name: "goland-2026.2.3", Dir: testDir}
	return m, s, install{s: s, b: b, sum: fmt.Sprintf("%x", sha256.Sum256(testBody))}
}

func TestInstallUnpacksThenMovesIntoPlace(t *testing.T) {
	m, _, c := fixture(1 << 30)
	if err := c.unpack(context.Background()); err != nil {
		t.Fatal(err)
	}
	part := testDir + ".part"
	want := []string{
		"run rm -rf -- " + part,
		"run mkdir -m 0755 -- " + part,
		"run tar -xzf /h/.cache/ws-jetbrains/playbook/goland-2026.2.3.tar.gz -C " + part + " --strip-components=1",
		"run touch -- " + part + "/" + marker,
		"run mv -T --update=none-fail -- " + part + " " + testDir,
	}
	var got []string
	for _, l := range m.Calls {
		if strings.HasPrefix(l, "run ") {
			got = append(got, l)
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("commands:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestInstallFailsWhereTheUnpackFails(t *testing.T) {
	m, _, c := fixture(1 << 30)
	m.Results["tar -xzf /h/.cache/ws-jetbrains/playbook/goland-2026.2.3.tar.gz -C "+testDir+".part --strip-components=1"] =
		effecttest.Result{Err: errors.New("cut")}
	if err := c.unpack(context.Background()); err == nil {
		t.Fatal("no error")
	}
	if slices.ContainsFunc(m.Calls, func(l string) bool { return strings.HasPrefix(l, "run mv") }) {
		t.Error("moved a tree that did not unpack")
	}
}
