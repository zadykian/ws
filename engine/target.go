package engine

import (
	"context"
	"io"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
)

// Target is what a side gives the verbs: the runs that they drive.
type Target struct {
	// Prepare checks where the call runs and what it names, then readies an apply. Its errors are
	// the command's own, in the order that a side checks them.
	Prepare func(ctx context.Context, c Call) (Run, error)
	// Change says a change under --check in the side's way. Nil prints each as "PATH: change".
	Change func(step.Change) (string, bool)
}

// Run is a prepared apply.
type Run struct {
	// M is the machine of the run and of its screen: the server's is confined to ws_home.
	M effect.Machine
	// Groups names the steps of the run screen, as the run would plan them with check.
	Groups func(check bool) []tui.Group
	// Listed is how many leading groups the list of steps leaves out: the server's Prepare group.
	Listed int
	// Do runs the steps on e with o, and returns the line to say after them, or "".
	Do func(ctx context.Context, e Effects, o step.Options) (string, error)
	// Choose returns the run and the options that the list's choice sets.
	Choose func(choice tui.Choice) (Run, step.Options, error)
}

// Effects are what a run acts on: the machine, the lines it says and the text it prints on stdout.
type Effects struct {
	M    effect.Machine
	Out  effect.Out
	Text io.Writer
}
