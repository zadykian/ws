package etcws_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/internal/etcws"
)

func TestTopIsTheDirectoryAlone(t *testing.T) {
	m := effecttest.NewMachine()
	changes, err := etcws.Top(m).Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || !strings.HasPrefix(changes[0].String(), "create directory /etc/ws, owner 0:0, mode 755") {
		t.Errorf("changes = %q, want one that makes /etc/ws at 755", changes)
	}
}

func TestInsideLeavesOutTheTop(t *testing.T) {
	m := effecttest.NewMachine()
	var got []string
	for _, r := range etcws.Inside(m, etcws.Dir+"/guests/a/x.pub", []byte("a\n")) {
		changes, err := r.Plan(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range changes {
			got = append(got, strings.SplitN(c.String(), "\n", 2)[0])
		}
	}
	want := []string{"create directory /etc/ws/guests, owner 0:0, mode 755",
		"create directory /etc/ws/guests/a, owner 0:0, mode 755", "create /etc/ws/guests/a/x.pub"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("changes = %q, want %q", got, want)
	}
}

func TestHolds(t *testing.T) {
	for p, want := range map[string]bool{
		"/etc/ws/x": true, "/etc/ws/servers/a/values.yml": true,
		"/etc/ws/../x": false, "/etc/wsx": false, "/etc/ws": false, "/h/x": false,
	} {
		if got := etcws.Holds(p); got != want {
			t.Errorf("Holds(%q) = %v, want %v", p, got, want)
		}
	}
}
