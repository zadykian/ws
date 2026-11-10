package tui

import "context"

// Wizard is a wizard's screens, which RunWizard draws in turn. The lists open over a scan, and lead
// to the state screen and back, to the forms and a new scan, or to the run of what they chose.
type Wizard struct {
	// Title heads each screen. The state screen's names its machine past it.
	Title  string
	Config Config
	// Round gives a round's works. The forms lead to a new one, whose works read what they wrote.
	Round func() Round
	// Forms gives the sheets of the configure button's forms. done, or nil, ends what they hold.
	Forms func(ctx context.Context) (sheets []Sheet, done func(context.Context), err error)
}

// Round is the works of a scan and of what follows it, which may share what the scan asked.
type Round struct {
	Stages []Stage
	Scan   Work
	// Read gives the state screen's read of side's machine. done, or nil, ends what the read holds.
	Read func(side Side) (read StateReader, done func(context.Context))
	// Run runs what the lists chose, on a screen of its own.
	Run func(ctx context.Context, choice Choice) error
}

// RunWizard draws w's screens until a leave or a run, and returns the run's error or a screen's.
func RunWizard(ctx context.Context, w Wizard) error {
	return w.loop(ctx, terminalScreens{w})
}

// screens are what the loop draws. A test of the loop fakes them.
type screens interface {
	pick(ctx context.Context, r Round) (Choice, error)
	state(ctx context.Context, side Side, read StateReader) error
	reopen(ctx context.Context, left Choice) (Choice, error)
	configure(ctx context.Context, sheets []Sheet) error
}

// loop draws a round's lists, then what they chose. The state screen changes nothing, so the lists
// come back as it found them, with no scan. The forms may change what a scan reads.
func (w Wizard) loop(ctx context.Context, s screens) error {
	for {
		r := w.Round()
		choice, err := s.pick(ctx, r)
		for err == nil && choice.State {
			if err = w.state(ctx, s, r, choice.Side); err == nil {
				choice, err = s.reopen(ctx, choice)
			}
		}
		switch {
		case err != nil:
			return err
		case choice.Configure:
			if err := w.forms(ctx, s); err != nil {
				return err
			}
		case len(choice.Only) == 0:
			return nil
		default:
			return r.Run(ctx, choice)
		}
	}
}

// state draws the state screen of side's machine.
func (w Wizard) state(ctx context.Context, s screens, r Round, side Side) error {
	read, done := r.Read(side)
	if done != nil {
		defer done(ctx)
	}
	return s.state(ctx, side, read)
}

// forms draws the forms of w's values.
func (w Wizard) forms(ctx context.Context, s screens) error {
	sheets, done, err := w.Forms(ctx)
	if err != nil {
		return err
	}
	if done != nil {
		defer done(ctx)
	}
	return s.configure(ctx, sheets)
}

// terminalScreens is the loop's screens on the terminal.
type terminalScreens struct{ w Wizard }

func (d terminalScreens) pick(ctx context.Context, r Round) (Choice, error) {
	return PickStages(ctx, d.w.Config, d.w.Title, r.Stages, r.Scan)
}
func (d terminalScreens) state(ctx context.Context, side Side, read StateReader) error {
	return RunState(ctx, d.w.Config, d.w.Title+": "+side.String(), read)
}
func (d terminalScreens) reopen(ctx context.Context, left Choice) (Choice, error) {
	return Reopen(ctx, d.w.Config, left)
}
func (d terminalScreens) configure(ctx context.Context, sheets []Sheet) error {
	return Configure(ctx, d.w.Config, d.w.Title, sheets...)
}
