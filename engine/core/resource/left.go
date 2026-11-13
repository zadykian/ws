package resource

import (
	"context"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// Left takes a user out of a group. A session that is running keeps the group until it ends.
type Left struct {
	M     effect.Machine
	User  string
	Group string
}

type leaveGroup struct{ l Left }

func (c leaveGroup) String() string {
	return "remove " + c.l.User + " from the " + c.l.Group + " group"
}

func (leaveGroup) Kind() step.ChangeKind { return step.Removal }

func (c leaveGroup) Apply(ctx context.Context) error {
	_, err := c.l.M.RunRoot(ctx, "gpasswd", "--delete", c.l.User, c.l.Group)
	return err
}

// Plan reads /etc/group. A missing group has no member to lose.
func (l Left) Plan(context.Context) ([]Change, error) {
	groups, err := l.M.ReadFile("/etc/group")
	if err != nil {
		return nil, err
	}
	for line := range strings.Lines(string(groups)) {
		f := strings.Split(strings.TrimSpace(line), ":")
		if len(f) == 4 && f[0] == l.Group && slices.Contains(strings.Split(f[3], ","), l.User) {
			return []Change{leaveGroup{l}}, nil
		}
	}
	return nil, nil
}
