package jetbrains

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestDirsMakesTheMissingOnesTopFirst(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/h"] = 0o755
	m.Dirs["/h/.cache"] = 0o700
	s := Site{M: m, Home: "/h", AsUser: func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return m.Run(ctx, name, args...)
	}}
	changes, err := Dirs{S: s, Path: "/h/.cache/JetBrains/RemoteDev/dist"}.Plan(context.Background())
	if err != nil || len(changes) != 3 {
		t.Fatalf("%v, %v", changes, err)
	}
	for _, c := range changes {
		dir := strings.TrimPrefix(c.String(), "create directory ")
		dir = strings.TrimSuffix(dir, ", mode 755")
		m.Results["mkdir -m 0755 -- "+dir] = effecttest.Result{}
		if err := c.Apply(context.Background()); err != nil {
			t.Fatal(err)
		}
		m.Dirs[dir] = 0o755
	}
	var made []string
	for _, l := range m.Calls {
		if strings.HasPrefix(l, "run mkdir") {
			made = append(made, strings.TrimPrefix(l, "run mkdir -m 0755 -- "))
		}
	}
	if want := []string{"/h/.cache/JetBrains", "/h/.cache/JetBrains/RemoteDev", "/h/.cache/JetBrains/RemoteDev/dist"}; !slices.Equal(made, want) {
		t.Errorf("made %v, want %v", made, want)
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
