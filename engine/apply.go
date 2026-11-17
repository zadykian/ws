package engine

import (
	"context"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// Apply runs a Target's apply. It says the run's lines on stderr, or prints its events on stdout
// with --json.
func Apply(ctx context.Context, c Call, t Target) error {
	run, err := t.Prepare(ctx, c)
	if err != nil {
		return err
	}
	c.Machine = run.M // WS_HOME confines it.
	term := effect.NewTerminal(strings.NewReader(""), c.Err)
	lines := Lines{Out: term, Prog: c.Path, Seen: map[string]bool{}, Change: t.Change}
	o := step.Options{Check: c.Check, Observer: lines}
	var events *jsonEvents
	if c.JSON {
		events = newJSONEvents(c.Out, lines)
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
