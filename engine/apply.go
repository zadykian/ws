package engine

import (
	"context"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
)

// Apply runs a Target's apply. It draws the run's screen, or says lines where no screen is wanted,
// and --json prints its events on stdout.
func Apply(ctx context.Context, c Call, t Target) error {
	run, err := t.Prepare(ctx, c)
	if err != nil {
		return err
	}
	c.Machine = run.M // ws_home confines it.
	term := effect.NewTerminal(strings.NewReader(""), c.Err)
	lines := Lines{Out: term, Prog: c.Path, Seen: map[string]bool{}, Change: t.Change}
	o := step.Options{Check: c.Check}
	var events *JSONEvents
	var line string
	if DrawsScreen(c) {
		err = DrawRun(ctx, c, RunTitle(c, o.Check), run.Groups(o.Check), func(ctx context.Context, b *tui.Bridge) error {
			e := b.Effects()
			var err error
			line, err = run.Do(ctx, Effects{M: e.M, Out: b, Text: e.Text}, b.Options(o))
			if err == nil && line != "" { // A plain run says the line after the screen, and the log holds it.
				b.Log(line)
			}
			return err
		})
	} else {
		o.Observer = lines
		if c.JSON {
			events = NewJSONEvents(c.Out, lines)
			o.Observer = events
		}
		line, err = run.Do(ctx, Effects{M: run.M, Out: lines, Text: c.Out}, o)
	}
	term.Flush()
	if err != nil {
		return err
	}
	if events != nil && events.Err() != nil {
		return events.Err()
	}
	if line != "" {
		lines.Say("%s", line)
	}
	return nil
}
