package devtools

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

const goFile = "go1.27.3.linux-amd64.tar.gz"

func goTreeMachine(listed string) *effecttest.Machine {
	m := effecttest.NewMachine()
	sum := fmt.Sprintf("%x", sha256.Sum256([]byte("a tarball")))
	list := `[{"version":"go1.27.3","files":[{"filename":"` + listed + `","sha256":"` + sum + `"}]}]`
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Results[curl+goReleases] = effecttest.Result{Out: []byte(list)}
	return m
}

func TestGoTreePlansItsSumFromTheList(t *testing.T) {
	m := goTreeMachine(goFile)
	changes, err := goTree(m, "1.27.3").Plan(t.Context())
	if err != nil || len(changes) != 3 {
		t.Fatalf("got %d changes, %v; want the directory, the download and the install", len(changes), err)
	}
	if want := "download https://go.dev/dl/" + goFile + " to /var/cache/ws/" + goFile; changes[1].String() != want {
		t.Errorf("change = %q, want %q", changes[1], want)
	}
	if want := "install https://go.dev/dl/" + goFile + " in " + goDir; changes[2].String() != want {
		t.Errorf("change = %q, want %q", changes[2], want)
	}
}

func TestGoTreeCurrentAsksForNoList(t *testing.T) {
	m := goTreeMachine(goFile)
	m.Files[goDir+"/.ws-source"] = []byte("https://go.dev/dl/" + goFile + "\n")
	if changes, err := goTree(m, "1.27.3").Plan(t.Context()); err != nil || len(changes) != 0 {
		t.Errorf("got %v, %v; want none", changes, err)
	}
	for _, c := range m.Calls {
		if strings.HasPrefix(c, "run curl") {
			t.Errorf("a current tree fetched: %q", m.Calls)
		}
	}
}

func TestGoTreeFails(t *testing.T) {
	m := goTreeMachine("go1.27.3.linux-arm64.tar.gz")
	if _, err := goTree(m, "1.27.3").Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "lists no "+goFile) {
		t.Errorf("a list without the file: %v", err)
	}
	m = goTreeMachine(goFile)
	m.Results[curl+goReleases] = effecttest.Result{Err: errors.New("503")}
	if _, err := goTree(m, "1.27.3").Plan(t.Context()); err == nil {
		t.Error("a failed fetch planned")
	}
}
