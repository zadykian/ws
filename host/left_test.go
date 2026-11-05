package host_test

import (
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func TestLeftTakesAUserOutOfAGroupOnce(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/etc/group"] = []byte("alice:x:1000:\ndocker:x:998:bob,alice\n")
	l := host.Left{M: m, User: "alice", Group: "docker"}
	changes, err := l.Plan(t.Context())
	if err != nil || len(changes) != 1 || changes[0].String() != "remove alice from the docker group" {
		t.Fatalf("Plan = %v, %v; want the user removed", changes, err)
	}
	m.Results["gpasswd --delete alice docker"] = effecttest.Result{}
	if err := changes[0].Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	m.Files["/etc/group"] = []byte("docker:x:998:bob\n")
	if changes, err := l.Plan(t.Context()); err != nil || changes != nil {
		t.Errorf("second Plan = %v, %v; want none", changes, err)
	}
}

func TestLeftLeavesAMissingGroupOrMemberAlone(t *testing.T) {
	for _, group := range []string{"wheel:x:10:alice\n", "docker:x:998:bob\n", ""} {
		m := effecttest.NewMachine()
		m.Files["/etc/group"] = []byte(group)
		if changes, err := (host.Left{M: m, User: "alice", Group: "docker"}).Plan(t.Context()); err != nil || changes != nil {
			t.Errorf("group file %q: Plan = %v, %v; want none", group, changes, err)
		}
	}
}
