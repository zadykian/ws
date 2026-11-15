package etcws_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/internal/etcws"
)

func TestFilesMakeTheDirectoryBeforeTheFile(t *testing.T) {
	m := effecttest.NewMachine()
	var got []string
	for _, r := range etcws.Files(m, etcws.Dir+"/x", []byte("a\n")) {
		changes, err := r.Plan(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range changes {
			got = append(got, strings.SplitN(c.String(), "\n", 2)[0])
		}
	}
	want := []string{"create directory /etc/ws, owner 0:0, mode 755", "create /etc/ws/x"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("changes = %q, want %q", got, want)
	}
}

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
