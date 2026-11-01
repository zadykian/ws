package jetbrains

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestDirsMakesTheMissingOnesTopFirst(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/h"] = 0o755
	m.Dirs["/h/.cache"] = 0o700
	s := Site{M: m, Home: "/h", UID: 1000, GID: 1000}
	changes, err := Dirs{S: s, Path: "/h/.cache/JetBrains/RemoteDev/dist"}.Plan(context.Background())
	if err != nil || len(changes) != 3 {
		t.Fatalf("%v, %v", changes, err)
	}
	for _, c := range changes {
		if err := c.Apply(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	for _, d := range []string{"/h/.cache/JetBrains", "/h/.cache/JetBrains/RemoteDev", "/h/.cache/JetBrains/RemoteDev/dist"} {
		if m.Dirs[d] != 0o755 || m.Owners[d] != [2]int{1000, 1000} {
			t.Errorf("%s: mode %o owner %v", d, m.Dirs[d], m.Owners[d])
		}
	}
	if m.Dirs["/h/.cache"] != 0o700 {
		t.Error("an existing directory's mode changed")
	}
	again, err := Dirs{S: s, Path: "/h/.cache/JetBrains/RemoteDev/dist"}.Plan(context.Background())
	if err != nil || len(again) != 0 {
		t.Errorf("a second plan: %v, %v", again, err)
	}
}

func TestDirsRefusesAPathOutsideTheHome(t *testing.T) {
	_, err := Dirs{S: Site{M: effecttest.NewMachine(), Home: "/h"}, Path: "/x/dist"}.Plan(context.Background())
	if err == nil || !strings.Contains(err.Error(), "outside") {
		t.Errorf("got %v", err)
	}
}
