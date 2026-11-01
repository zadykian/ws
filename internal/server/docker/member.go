package docker

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// member adds a user to the docker group, which the packages create and Docker's socket answers.
// The group is root-equivalent, and the user has it from its next login.
type member struct {
	m    effect.Machine
	user string
}

type addToGroup struct {
	m    effect.Machine
	user string
}

func (c addToGroup) String() string { return "add " + c.user + " to the docker group" }

func (c addToGroup) Apply(ctx context.Context) error {
	_, err := c.m.RunRoot(ctx, "usermod", "--append", "--groups", "docker", c.user)
	return err
}

func (c member) Plan(context.Context) ([]host.Change, error) {
	passwd, err := c.m.ReadFile("/etc/passwd")
	if err != nil {
		return nil, err
	}
	gid, ok := primaryGroup(string(passwd), c.user)
	if !ok {
		return nil, fmt.Errorf("the user %s is missing", c.user)
	}
	groups, err := c.m.ReadFile("/etc/group")
	if err != nil {
		return nil, err
	}
	for line := range strings.Lines(string(groups)) {
		f := strings.Split(strings.TrimSpace(line), ":")
		if len(f) == 4 && f[0] == "docker" {
			if f[2] == gid || slices.Contains(strings.Split(f[3], ","), c.user) {
				return nil, nil
			}
			return []host.Change{addToGroup(c)}, nil
		}
	}
	return nil, fmt.Errorf("the group docker is missing: the packages create it")
}

func primaryGroup(passwd, user string) (string, bool) {
	for line := range strings.Lines(passwd) {
		if f := strings.Split(line, ":"); len(f) >= 4 && f[0] == user {
			return f[3], true
		}
	}
	return "", false
}
