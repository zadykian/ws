package docker

import (
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func memberFixture(group string) *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Files["/etc/passwd"] = []byte("root:x:0:0::/root:/bin/bash\nalice:x:1000:1000::/home/alice:/bin/bash\n")
	m.Files["/etc/group"] = []byte("alice:x:1000:\n" + group + "\n")
	return m
}

func TestMemberAddsAUserOnce(t *testing.T) {
	m := memberFixture("docker:x:998:bob")
	changes, err := member{m, "alice"}.Plan(t.Context())
	if err != nil || len(changes) != 1 || changes[0].String() != "add alice to the docker group" {
		t.Fatalf("Plan = %v, %v; want the user added", changes, err)
	}
	m.Results["usermod --append --groups docker alice"] = effecttest.Result{}
	_ = changes[0].Apply(t.Context())
	m.Files["/etc/group"] = []byte("docker:x:998:bob,alice\n")
	if changes, err := (member{m, "alice"}).Plan(t.Context()); err != nil || changes != nil {
		t.Errorf("second Plan = %v, %v; want none", changes, err)
	}
}

func TestMemberRefusesAMissingUserOrGroup(t *testing.T) {
	for user, group := range map[string]string{"carol": "docker:x:998:", "alice": "wheel:x:10:"} {
		if _, err := (member{memberFixture(group), user}).Plan(t.Context()); err == nil {
			t.Errorf("%s with the group line %q passed", user, group)
		}
	}
}
