package host

import (
	"context"
	"slices"
	"strings"

	"github.com/zadykian/ws/effect"
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
func (p Purged) Plan(ctx context.Context) ([]Change, error) {
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
	return []Change{purge{p.M, present}}, nil
}

// Report shows each package as dpkg knows it. One that is installed stands against ws, as a run
// purges it.
func (p Purged) Report(ctx context.Context) ([]step.Item, error) {
	states, err := PackageStates(ctx, p.M, p.Names...)
	if err != nil {
		return nil, err
	}
	items := make([]step.Item, len(p.Names))
	for i, name := range p.Names {
		items[i] = step.Item{Kind: step.PackageItem, Name: name, Want: missing, Found: states[name].text()}
		if states[name].Installed() {
			items[i].Conflict = "installed, and ws purges it"
		}
	}
	return items, nil
}
