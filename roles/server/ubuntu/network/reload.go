package network

import (
	"context"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/systemd"
)

// reloaded plans the resource, and the two commands that put a change in force. They run on a
// booted machine only, and still run after a failed change, as the handlers did.
func (d direct) reloaded(ctx context.Context, r resource.Resource) ([]resource.Change, error) {
	changes, err := r.Plan(ctx)
	if err != nil || len(changes) == 0 || !systemd.Booted(d.M) {
		return changes, err
	}
	return append(changes, command{m: d.M, why: "write networkd's files from netplan's", name: "netplan", args: []string{"generate"}},
		command{m: d.M, why: "reload networkd, which sets up again each link whose file changed", name: "networkctl", args: []string{"reload"}}), nil
}

// command is a command that a change calls for.
type command struct {
	m    effect.Machine
	why  string
	name string
	args []string
}

func (c command) String() string { return c.why }

// Forced makes a failed run still run the command, as a handler does.
func (command) Forced() bool { return true }

func (c command) Apply(ctx context.Context) error {
	_, err := c.m.RunRoot(ctx, c.name, c.args...)
	return err
}
