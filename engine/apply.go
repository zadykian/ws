package engine

import (
	"context"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// Apply runs a Target's apply. It says lines, and --json prints the run's events on stdout.
func Apply(ctx context.Context, c Call, t Target) error {
	run, err := t.Prepare(ctx, c)
	if err != nil {
		return err
	}
	c.Machine = run.M
	term := effect.NewTerminal(strings.NewReader(""), c.Err)
	lines := Lines{Out: term, Prog: c.Path, Seen: map[string]bool{}, Change: t.Change}
	o := step.Options{Check: c.Check}
	o.Observer = lines
	var events *JSONEvents
	if c.JSON {
		events = NewJSONEvents(c.Out, lines)
		o.Observer = events
	}
	line, err := run.Do(ctx, Effects{M: run.M, Out: lines, Text: c.Out}, o)
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
