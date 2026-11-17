package engine

import (
	"context"
	"io"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// Target is what a side gives the verbs: the runs that they drive.
type Target struct {
	// Prepare checks where the call runs and what it names, then readies an apply. Its errors are
	// the command's own, in the order that a side checks them.
	Prepare func(ctx context.Context, c Call) (Run, error)
	// Change says a change under --check in the side's way. Nil prints each as "PATH: change".
	Change func(step.Change) ([]string, bool)
	// State readies the report of state. Its errors are the command's own.
	State func(ctx context.Context, c Call) (Report, error)
}

// Report is a prepared state: what the roles report.
type Report struct {
	// Print reads the machine and prints the report on c.Out, as JSON where c.JSON.
	Print func(ctx context.Context, c Call) error
}

// Run is a prepared apply.
type Run struct {
	// M is the machine of the run: the server's is confined to WS_HOME.
	M effect.Machine
	// Do runs the steps on e with o, and returns the line to say after them, or "".
	Do func(ctx context.Context, e Effects, o step.Options) (string, error)
}

// Effects are what a run acts on: the machine, the lines it says and the text it prints on stdout.
type Effects struct {
	M    effect.Machine
	Out  effect.Out
	Text io.Writer
}
