package claude

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func restoreMachine(check string) (*effecttest.Machine, Restore) {
	m := effecttest.NewMachine()
	m.Dirs["/run/systemd/system"] = 0o755
	m.Files["/root/.local/bin/cld"] = nil
	r := Restore{M: m, Account: host.Account{User: root}, Out: &effecttest.Out{}}
	name, args := r.loginShell("-c", compare, r.cld())
	m.Results["env HOME=/root "+name+" "+strings.Join(args, " ")] = effecttest.Result{Out: []byte(check)}
	return m, r
}

func TestRestorePlan(t *testing.T) {
	enabled := "/root/.config/systemd/user/default.target.wants/cld-restore.service"
	cases := []struct {
		name, check string
		linked      bool
		want        int
	}{
		{"missing", "missing\n", true, 1},
		{"differs", "differs\n", true, 1},
		{"not enabled", "", false, 1},
		{"login shell prints", "Welcome\n", true, 0},
		{"done", "", true, 0},
	}
	for _, c := range cases {
		m, r := restoreMachine(c.check)
		if c.linked {
			m.Files[enabled] = nil
		}
		got, err := r.Plan(context.Background())
		if err != nil || len(got) != c.want {
			t.Errorf("%s: %v, %v", c.name, got, err)
		}
	}
}

func TestRestoreNeedsSystemd(t *testing.T) {
	m, r := restoreMachine("missing\n")
	delete(m.Dirs, "/run/systemd/system")
	if got, err := r.Plan(context.Background()); err != nil || len(got) != 0 {
		t.Fatalf("%v, %v", got, err)
	}
}

func TestRestoreFailsWithoutLingering(t *testing.T) {
	m, r := restoreMachine("missing\n")
	name, args := r.loginShell("-c", setup, r.cld())
	key := "env HOME=/root " + name + " " + strings.Join(args, " ")
	m.Results[key] = effecttest.Result{Out: []byte("Created x\nLingering is off\n")}
	if err := (setupRestore{r}).Apply(context.Background()); err == nil {
		t.Fatal("no error")
	}
	m.Results[key] = effecttest.Result{Out: []byte("Created x\nLingering is on: yes\n")}
	if err := (setupRestore{r}).Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
}
