package devtools

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

const goFile = "go1.27.3.linux-amd64.tar.gz"

type goFixture struct {
	m        *effecttest.Machine
	g        Go
	unpacked [][]any
}

func newGoFixture(_ *testing.T, listed string) *goFixture {
	f := &goFixture{m: effecttest.NewMachine()}
	body := []byte("a tarball")
	sum := fmt.Sprintf("%x", sha256.Sum256(body))
	list := `[{"version":"go1.27.3","files":[{"filename":"` + listed + `","sha256":"` + sum + `"}]}]`
	f.m.Results[curl+goReleases] = effecttest.Result{Out: []byte(list)}
	f.m.Pages["https://go.dev/dl/"+goFile] = body
	f.m.Results["mv "+goStaging+" "+goDir] = effecttest.Result{}
	f.g = Go{M: f.m, Version: "1.27.3", Arch: "amd64", unpack: func(archive, dir string, strip int) error {
		f.unpacked = append(f.unpacked, []any{archive, dir, strip})
		return nil
	}}
	return f
}

func TestGoInstallsWhereThereIsNone(t *testing.T) {
	f := newGoFixture(t, goFile)
	changes, err := f.g.Plan(context.Background())
	if err != nil || len(changes) != 2 {
		t.Fatalf("got %d changes, %v", len(changes), err)
	}
	apply(t, changes)
	if want := [][]any{{goTarball, goStaging, 1}}; fmt.Sprint(f.unpacked) != fmt.Sprint(want) {
		t.Errorf("unpacked %v, want %v", f.unpacked, want)
	}
	var roots []string
	for _, c := range f.m.Calls {
		if strings.HasPrefix(c, "root ") {
			roots = append(roots, c)
		}
	}
	want := []string{
		"root rm -rf -- " + goStaging, "root rm -rf -- " + goDir,
		"root mv " + goStaging + " " + goDir, "root rm -f -- " + goTarball,
	}
	if !slices.Equal(roots, want) {
		t.Errorf("root commands %q, want %q", roots, want)
	}
	f.m.Files[goDir+"/VERSION"] = []byte("go1.27.3\ntime 2026-10-01T00:00:00Z\n")
	if changes, err := f.g.Plan(context.Background()); err != nil || len(changes) != 0 {
		t.Errorf("a second plan has %v, %v", changes, err)
	}
}

func TestGoReplacesAnotherVersion(t *testing.T) {
	f := newGoFixture(t, goFile)
	f.m.Files[goDir+"/VERSION"] = []byte("go1.27.2\n")
	if changes, err := f.g.Plan(context.Background()); err != nil || len(changes) != 2 {
		t.Errorf("got %d changes, %v", len(changes), err)
	}
}

func TestGoKeepsTheTarballThatIsRight(t *testing.T) {
	f := newGoFixture(t, goFile)
	f.m.Files[goTarball] = []byte("a tarball")
	changes, err := f.g.Plan(context.Background())
	if err != nil || len(changes) != 1 || !strings.HasPrefix(changes[0].String(), "install Go") {
		t.Errorf("got %v, %v", changes, err)
	}
}

func TestGoRemovesWhatARunCutShortLeft(t *testing.T) {
	f := newGoFixture(t, goFile)
	f.m.Files[goDir+"/VERSION"] = []byte("go1.27.3\n")
	f.m.Files[goTarball] = []byte("x")
	f.m.Files[goStaging] = []byte("x") // the fake removes files alone, not directories
	changes, err := f.g.Plan(context.Background())
	if err != nil || len(changes) != 2 {
		t.Fatalf("got %d changes, %v", len(changes), err)
	}
	apply(t, changes)
	if changes, err := f.g.Plan(context.Background()); err != nil || len(changes) != 0 {
		t.Errorf("a second plan has %v, %v", changes, err)
	}
}

func TestGoFails(t *testing.T) {
	f := newGoFixture(t, "go1.27.3.linux-arm64.tar.gz")
	if _, err := f.g.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "lists no "+goFile) {
		t.Errorf("a list without the file: %v", err)
	}
	f = newGoFixture(t, goFile)
	f.m.Results[curl+goReleases] = effecttest.Result{Err: errors.New("503")}
	if _, err := f.g.Plan(context.Background()); err == nil {
		t.Error("a failed fetch planned")
	}
	f = newGoFixture(t, goFile)
	f.g.unpack = func(string, string, int) error { return errors.New("cut short") }
	changes, _ := f.g.Plan(context.Background())
	apply(t, changes[:1])
	if err := changes[1].Apply(context.Background()); err == nil {
		t.Error("an unpack that failed went on")
	}
	if slices.Contains(f.m.Calls, "root mv "+goStaging+" "+goDir) {
		t.Error("a tree cut short was moved into place")
	}
}
