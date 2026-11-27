package apt

import (
	"context"
	"github.com/zadykian/ws/engine/core/resource"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// Purged removes packages with their settings, and leaves those that aren't there.
type Purged struct {
	M     effect.Machine
	Names []string
}

type purge struct {
	m     effect.Machine
	names []string
}

func (c purge) String() string { return "purge " + strings.Join(c.names, " ") }

func (purge) Kind() step.ChangeKind { return step.Removal }

func (c purge) Apply(ctx context.Context) error {
	args := slices.Concat([]string{"DEBIAN_FRONTEND=noninteractive", "apt-get"}, installOptions, []string{"purge", "-y"}, c.names)
	return c.m.Pass(ctx, false, "env", args...)
}

// Plan reads dpkg once and purges each package with anything of it on the machine. One that left
// its settings alone still takes a purge. With no names, it plans nothing.
func (p Purged) Plan(ctx context.Context) ([]resource.Change, error) {
	states, err := PackageStates(ctx, p.M, p.Names...)
	if err != nil {
		return nil, err
	}
	var present []string
	for _, n := range p.Names {
		if states[n].present() {
			present = append(present, n)
		}
	}
	if len(present) == 0 {
		return nil, nil
	}
	return []resource.Change{purge{p.M, present}}, nil
}
