package step

import "github.com/zadykian/ws/effect"

// Kind says what an Event reports of a step.
type Kind int

const (
	// Planned reports a step's plan. A step that a run skips before its plan has none.
	Planned Kind = iota
	// Started reports that a step starts to apply its plan.
	Started
	// Done ends a step: it has nothing left to make, or under check found Changes to make.
	Done
	// Failed ends a step, and the run with it.
	Failed
	// Skipped ends a step that a run didn't plan.
	Skipped
)

// Event is what a run reports of one step. A step's events end with one of Kind Done, Failed or
// Skipped, so that a screen can tell a step left hanging from one still at work.
type Event struct {
	Kind Kind
	// Parent is the run's Options.Parent.
	Parent string
	// Step is the step's name. Index is its place in the run, as a name can come twice: ws mac apply
	// runs ssh again.
	Step  string
	Index int
	// Check is set in a run that applies nothing.
	Check bool
	// Changes is the step's plan from Planned on. It stays on Done, which says what the step made,
	// or would make under Check.
	Changes []Change
	// After names the awaited step that a Skipped step is behind. It is empty where a failure stopped
	// the run first.
	After string
	// Err is what the run returns, on Failed alone.
	Err error
}

func (e Event) as(k Kind) Event {
	e.Kind = k
	return e
}

func (e Event) failed(err error) Event {
	e.Err = err
	return e.as(Failed)
}

// Observer follows a run. It sees each event on the run's goroutine, as it happens, so a step's
// own lines fall between them. It keeps Changes as they are, as the run shares them.
type Observer interface {
	Observe(e Event)
}

// Printer prints a run's events as lines, as Run does on its out.
type Printer struct{ Out effect.Out }

// Observe prints a plan under check, its size as a step applies it, and what a step is behind.
func (p Printer) Observe(e Event) {
	switch {
	case e.Kind == Planned && e.Check:
		for _, c := range e.Changes {
			p.Out.Say("%s: %s", e.Step, c)
		}
	case e.Kind == Started:
		p.Out.Say("%s: applying %d change(s)", e.Step, len(e.Changes))
	case e.Kind == Skipped && e.After != "":
		p.Out.Say("%s: after %s", e.Step, e.After)
	}
}
