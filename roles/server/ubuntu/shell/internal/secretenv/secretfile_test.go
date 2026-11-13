package secretenv

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host"
)

func envFixture(env map[string]string) (*effecttest.Machine, Env) {
	m := effecttest.NewMachine()
	a := host.Account{User: host.User{Name: "ws", Home: "/home/ws"}, UID: 1000, GID: 1000}
	return m, Env{M: m, A: a, Env: env}
}

func TestSecretEnvIsWrittenOnceAndNeverShown(t *testing.T) {
	m, s := envFixture(map[string]string{"TOKEN": "hunter2"})
	for range 2 {
		changes, err := s.Plan(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range changes {
			if strings.Contains(c.String(), "hunter2") {
				t.Errorf("a change shows the value: %q", c)
			}
			if err := c.Apply(context.Background()); err != nil {
				t.Fatal(err)
			}
		}
		if !strings.Contains(string(m.Files["/home/ws/.secrets/env"]), "export TOKEN=hunter2\n") {
			t.Fatalf("file = %q", m.Files["/home/ws/.secrets/env"])
		}
	}
	if m.Dirs["/home/ws/.secrets"] != 0o700 {
		t.Errorf("directory mode %o", m.Dirs["/home/ws/.secrets"])
	}
}

func TestSecretEnvLeavesTheFileWithoutValues(t *testing.T) {
	m, s := envFixture(nil)
	m.Files["/home/ws/.secrets/env"] = []byte("by hand")
	if changes, err := s.Plan(context.Background()); err != nil || len(changes) != 0 {
		t.Errorf("got %v, %v", changes, err)
	}
}

func TestSecretEnvRefusesAFileWrittenByHand(t *testing.T) {
	m, s := envFixture(map[string]string{"A": "new-secret"})
	m.Files["/home/ws/.secrets/env"] = []byte("export OLD=old-secret\n")
	_, err := s.Plan(context.Background())
	if err == nil {
		t.Fatalf("got %v", err)
	}
	if strings.Contains(err.Error(), "old-secret") || strings.Contains(err.Error(), "new-secret") {
		t.Errorf("the error shows a value: %v", err)
	}
}

func TestSecretEnvGoesWithTheWholeTruthAndAnEmptyTable(t *testing.T) {
	m, s := envFixture(nil)
	s.Whole = true
	m.Files["/home/ws/.secrets/env"] = []byte(envHeader + "export A=hunter2\n")
	changes, err := s.Plan(context.Background())
	if err != nil || len(changes) != 1 {
		t.Fatalf("got %v, %v", changes, err)
	}
	if strings.Contains(changes[0].String(), "hunter") {
		t.Errorf("the change shows a value: %q", changes[0])
	}
	if err := changes[0].Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Files["/home/ws/.secrets/env"]; ok {
		t.Error("the file stays")
	}
	if changes, err := s.Plan(context.Background()); err != nil || len(changes) != 0 {
		t.Errorf("second plan = %v, %v", changes, err)
	}
}

func TestSecretEnvWithTheWholeTruthKeepsAFileWrittenByHandOutOfTheWay(t *testing.T) {
	m, s := envFixture(nil)
	s.Whole = true
	m.Files["/home/ws/.secrets/env"] = []byte("export A=hunter2\n")
	if _, err := s.Plan(context.Background()); err == nil || strings.Contains(err.Error(), "hunter2") {
		t.Errorf("got %v", err)
	}
}
